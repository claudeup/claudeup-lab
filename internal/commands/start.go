package commands

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/claudeup/claudeup-lab/internal/lab"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func newStartCmd() *cobra.Command {
	var opts lab.StartOptions
	var features []string
	var envFlags []string

	cmd := &cobra.Command{
		Use:   "start [preset-name]",
		Short: "Create and start a lab",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Features = features

			// Parse --env KEY=VALUE flags into map
			envVars, err := parseEnvFlags(envFlags)
			if err != nil {
				return err
			}
			opts.EnvVars = envVars

			resolved, err := resolveInitScript(opts.InitScript)
			if err != nil {
				return err
			}
			opts.InitScript = resolved

			mgr := lab.NewManager(baseDirFn())

			// Determine the final start options, handling preset if specified.
			finalOpts := &opts
			if len(args) > 0 {
				presetName := args[0]
				preset, loadErr := mgr.Presets().Load(presetName)
				if loadErr != nil {
					return fmt.Errorf("preset '%s' not found", presetName)
				}

				baseOpts := preset.ToStartOptions()

				// Detect which flags were explicitly set on the command line.
				changed := make(map[string]bool)
				cmd.Flags().Visit(func(f *pflag.Flag) {
					changed[f.Name] = true
				})

				merged, overrides := lab.MergeWithOverrides(baseOpts, &opts, changed)
				finalOpts = merged

				// Print preset summary to stderr.
				fmt.Fprint(cmd.ErrOrStderr(), lab.FormatOverrides(preset, overrides))
			}

			// Apply cwd default for project only if still empty after preset merge.
			if finalOpts.Project == "" {
				cwd, cwdErr := os.Getwd()
				if cwdErr != nil {
					return fmt.Errorf("get working directory: %w", cwdErr)
				}
				finalOpts.Project = cwd
			}

			meta, err := mgr.Start(finalOpts)
			if err != nil && meta == nil {
				return err
			}

			fmt.Println()
			var postCreateErr *lab.PostCreateError
			if errors.As(err, &postCreateErr) {
				fmt.Fprintf(os.Stderr, "Warning: %s\n", postCreateErr)
				fmt.Fprintf(os.Stderr, "The container is running but may not be fully configured.\n\n")
			}
			fmt.Println("Lab ready!")
			fmt.Printf("  Name:     %s\n", meta.DisplayName)
			fmt.Printf("  ID:       %s\n", meta.ID[:8])
			fmt.Printf("  Worktree: %s\n", meta.Worktree)
			fmt.Printf("  Branch:   %s\n", meta.Branch)
			fmt.Printf("  Profile:  %s\n", meta.Profile)
			fmt.Println()
			fmt.Println("Next steps:")
			fmt.Printf("  claudeup-lab exec --lab %s -- <command>\n", meta.DisplayName)
			fmt.Printf("  claudeup-lab exec --lab %s -- claude\n", meta.DisplayName)
			fmt.Printf("  claudeup-lab open --lab %s\n", meta.DisplayName)
			fmt.Printf("  claudeup-lab stop --lab %s\n", meta.DisplayName)
			return err
		},
	}

	cmd.Flags().StringVar(&opts.Project, "project", "", "Project directory (default: current directory)")
	cmd.Flags().StringVar(&opts.Profile, "profile", "", "claudeup profile (default: snapshot current config)")
	cmd.Flags().StringVar(&opts.Branch, "branch", "", "Git branch name (default: lab/<profile>)")
	cmd.Flags().StringVar(&opts.Name, "name", "", "Display name for the lab")
	cmd.Flags().StringSliceVar(&features, "feature", nil, "Devcontainer feature (repeatable, e.g. go:1.23)")
	cmd.Flags().StringVar(&opts.BaseProfile, "base-profile", "", "Apply base profile before main profile")
	cmd.Flags().BoolVar(&opts.Firewall, "firewall", false, "Enable container firewall restricting network to allowed services")
	cmd.Flags().StringVar(&opts.InitScript, "init-script", "", "Host script to run after devcontainer setup")
	cmd.Flags().StringArrayVar(&envFlags, "env", nil, "Set container environment variable as KEY=VALUE (repeatable)")
	cmd.Flags().StringVar(&opts.EnvFile, "env-file", "", "Read container environment variables from file")

	return cmd
}

// resolveInitScript validates and resolves an init script path to an absolute path.
// It verifies the path exists and refers to a regular file.
// Returns an empty string if path is empty (no init script specified).
func resolveInitScript(path string) (string, error) {
	if path == "" {
		return "", nil
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve init script path: %w", err)
	}
	info, err := os.Stat(absPath)
	if os.IsNotExist(err) {
		return "", fmt.Errorf("init script not found: %s", absPath)
	}
	if err != nil {
		return "", fmt.Errorf("init script not accessible: %w", err)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("init script is not a regular file: %s", absPath)
	}
	return absPath, nil
}

// parseEnvFlags parses a slice of "KEY=VALUE" strings into a map.
func parseEnvFlags(flags []string) (map[string]string, error) {
	if len(flags) == 0 {
		return nil, nil
	}

	env := make(map[string]string, len(flags))
	for _, e := range flags {
		idx := strings.IndexByte(e, '=')
		if idx < 0 {
			return nil, fmt.Errorf("invalid --env format %q: expected KEY=VALUE", e)
		}
		key := e[:idx]
		if key == "" {
			return nil, fmt.Errorf("invalid --env format %q: empty key", e)
		}
		env[key] = e[idx+1:]
	}
	return env, nil
}
