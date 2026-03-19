package commands

import (
	"fmt"
	"path/filepath"

	"github.com/claudeup/claudeup-lab/internal/lab"
	"github.com/spf13/cobra"
)

func newPresetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "preset",
		Short: "Manage reusable lab configurations",
	}

	cmd.AddCommand(newPresetSaveCmd())

	return cmd
}

// startFlagNames lists all flag names that correspond to start options.
// Used to detect mutual exclusivity with --from-lab.
var startFlagNames = []string{
	"project", "profile", "branch", "feature", "base-profile",
	"firewall", "init-script", "env", "env-file",
}

func newPresetSaveCmd() *cobra.Command {
	var fromLab string
	var force bool
	var opts lab.StartOptions
	var features []string
	var envFlags []string

	cmd := &cobra.Command{
		Use:   "save <name>",
		Short: "Save a preset from flags or a lab's tracked configuration",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			mgr := lab.NewManager(baseDirFn())

			// Check mutual exclusivity: --from-lab vs start flags
			fromLabSet := cmd.Flags().Changed("from-lab")
			anyStartFlagSet := false
			for _, f := range startFlagNames {
				if cmd.Flags().Changed(f) {
					anyStartFlagSet = true
					break
				}
			}

			if fromLabSet && anyStartFlagSet {
				return fmt.Errorf("--from-lab cannot be combined with other flags")
			}
			if !fromLabSet && !anyStartFlagSet {
				return fmt.Errorf("provide --from-lab or at least one start flag (--project, --profile, etc.)")
			}

			var preset *lab.Preset

			if fromLabSet {
				// Build preset from lab's tracked config
				resolver := lab.NewResolver(mgr.Store())
				meta, err := resolver.Resolve(fromLab)
				if err != nil {
					return fmt.Errorf("lab '%s' not found", fromLab)
				}
				if meta.StartConfig == nil {
					return fmt.Errorf("lab '%s' has no tracked configuration", fromLab)
				}
				preset = presetFromStartConfig(name, meta.StartConfig)
			} else {
				// Build preset from explicit flags
				opts.Features = features

				envVars, err := parseEnvFlags(envFlags)
				if err != nil {
					return err
				}
				opts.EnvVars = envVars

				// Resolve paths to absolute
				if opts.Project != "" {
					abs, err := filepath.Abs(opts.Project)
					if err != nil {
						return fmt.Errorf("resolve project path: %w", err)
					}
					opts.Project = abs
				}

				resolved, err := resolveInitScript(opts.InitScript)
				if err != nil {
					return err
				}
				opts.InitScript = resolved

				if opts.EnvFile != "" {
					abs, err := filepath.Abs(opts.EnvFile)
					if err != nil {
						return fmt.Errorf("resolve env-file path: %w", err)
					}
					opts.EnvFile = abs
				}

				preset = lab.PresetFromStartOptions(name, &opts)
			}

			// Check for existing preset
			_, err := mgr.Presets().Load(name)
			if err == nil && !force {
				if !confirm(fmt.Sprintf("Preset '%s' already exists. Overwrite?", name)) {
					fmt.Println("Aborted.")
					return nil
				}
			}

			if err := mgr.Presets().Save(preset); err != nil {
				return err
			}

			fmt.Printf("Preset saved: %s\n", name)
			fields := lab.FormatPresetFields(preset)
			if fields != "" {
				fmt.Println(fields)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&fromLab, "from-lab", "", "Capture configuration from a named lab")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "Overwrite existing preset without prompting")

	cmd.Flags().StringVar(&opts.Project, "project", "", "Project directory")
	cmd.Flags().StringVar(&opts.Profile, "profile", "", "claudeup profile")
	cmd.Flags().StringVar(&opts.Branch, "branch", "", "Git branch name")
	cmd.Flags().StringSliceVar(&features, "feature", nil, "Devcontainer feature (repeatable)")
	cmd.Flags().StringVar(&opts.BaseProfile, "base-profile", "", "Base profile")
	cmd.Flags().BoolVar(&opts.Firewall, "firewall", false, "Enable container firewall")
	cmd.Flags().StringVar(&opts.InitScript, "init-script", "", "Host script to run after setup")
	cmd.Flags().StringArrayVar(&envFlags, "env", nil, "Set environment variable as KEY=VALUE (repeatable)")
	cmd.Flags().StringVar(&opts.EnvFile, "env-file", "", "Read environment variables from file")

	return cmd
}

// presetFromStartConfig creates a Preset from a lab's tracked StartConfig.
func presetFromStartConfig(name string, cfg *lab.StartConfig) *lab.Preset {
	return &lab.Preset{
		Name:        name,
		Project:     cfg.Project,
		Profile:     cfg.Profile,
		Branch:      cfg.Branch,
		LabName:     cfg.LabName,
		Features:    cfg.Features,
		BaseProfile: cfg.BaseProfile,
		Firewall:    cfg.Firewall,
		InitScript:  cfg.InitScript,
		EnvVars:     cfg.EnvVars,
		EnvFile:     cfg.EnvFile,
	}
}
