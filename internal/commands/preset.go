package commands

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/claudeup/claudeup-lab/internal/lab"
	"github.com/spf13/cobra"
)

func newPresetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "preset",
		Short: "Manage reusable lab configurations",
	}

	cmd.AddCommand(newPresetSaveCmd())
	cmd.AddCommand(newPresetDeleteCmd())
	cmd.AddCommand(newPresetListCmd())
	cmd.AddCommand(newPresetShowCmd())

	return cmd
}

// startFlagNames lists all flag names that correspond to start options.
// Used to detect mutual exclusivity with --from-lab.
// SYNC: must match the flags registered in newStartCmd() and newPresetSaveCmd().
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
					var notFound *lab.NotFoundError
					if errors.As(err, &notFound) {
						return fmt.Errorf("lab '%s' not found", fromLab)
					}
					return fmt.Errorf("resolve lab '%s': %w", fromLab, err)
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

			// Check for existing preset before saving.
			_, loadErr := mgr.Presets().Load(name)
			if loadErr == nil && !force {
				// Preset exists and is valid -- confirm overwrite.
				if !confirm(fmt.Sprintf("Preset '%s' already exists. Overwrite?", name)) {
					fmt.Println("Aborted.")
					return nil
				}
			} else if loadErr != nil && !errors.Is(loadErr, os.ErrNotExist) {
				// File exists but failed to load. This could be a corrupt file
				// or a name validation error. Only treat it as corrupt if the
				// file actually exists on disk.
				presetPath := filepath.Join(baseDirFn(), "presets", name+".json")
				if _, statErr := os.Stat(presetPath); statErr == nil {
					fmt.Fprintf(os.Stderr, "Warning: existing preset '%s' is corrupt: %v\n", name, loadErr)
					if !force {
						if !confirm(fmt.Sprintf("Overwrite corrupt preset '%s'?", name)) {
							fmt.Println("Aborted.")
							return nil
						}
					}
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

func newPresetDeleteCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:     "delete <name>",
		Aliases: []string{"rm"},
		Short:   "Delete a saved preset",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			mgr := lab.NewManager(baseDirFn())

			if _, err := mgr.Presets().Load(name); err != nil {
				if errors.Is(err, os.ErrNotExist) {
					return fmt.Errorf("preset '%s' not found", name)
				}
				return fmt.Errorf("failed to load preset %q: %w", name, err)
			}

			if !force {
				if !confirm(fmt.Sprintf("Delete preset '%s'?", name)) {
					fmt.Println("Aborted.")
					return nil
				}
			}

			if err := mgr.Presets().Delete(name); err != nil {
				return err
			}

			fmt.Printf("Deleted preset: %s\n", name)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Skip confirmation prompt")

	return cmd
}

func newPresetListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List saved presets",
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr := lab.NewManager(baseDirFn())

			presets, err := mgr.Presets().List()
			if err != nil {
				return err
			}

			if len(presets) == 0 {
				fmt.Println("No presets found.")
				return nil
			}

			// Calculate dynamic column widths
			nameW, projW, profW, flagsW := len("NAME"), len("PROJECT"), len("PROFILE"), len("FLAGS")
			type row struct {
				name, project, profile, flags string
			}
			rows := make([]row, len(presets))

			for i, p := range presets {
				r := row{
					name:    p.Name,
					project: presetProjectColumn(p),
					profile: presetProfileColumn(p),
					flags:   presetFlagsColumn(p),
				}
				rows[i] = r
				if len(r.name) > nameW {
					nameW = len(r.name)
				}
				if len(r.project) > projW {
					projW = len(r.project)
				}
				if len(r.profile) > profW {
					profW = len(r.profile)
				}
				if len(r.flags) > flagsW {
					flagsW = len(r.flags)
				}
			}

			fmtStr := fmt.Sprintf("%%-%ds  %%-%ds  %%-%ds  %%s\n", nameW, projW, profW)
			fmt.Printf(fmtStr, "NAME", "PROJECT", "PROFILE", "FLAGS")
			fmt.Printf(fmtStr, strings.Repeat("-", nameW), strings.Repeat("-", projW), strings.Repeat("-", profW), strings.Repeat("-", flagsW))

			for _, r := range rows {
				fmt.Printf(fmtStr, r.name, r.project, r.profile, r.flags)
			}

			return nil
		},
	}
}

func presetProjectColumn(p *lab.Preset) string {
	if p.Project == "" {
		return "(none)"
	}
	return filepath.Base(p.Project)
}

func presetProfileColumn(p *lab.Preset) string {
	if p.Profile == "" {
		return "(none)"
	}
	return p.Profile
}

// presetFlagsColumn builds a condensed FLAGS string for the list table.
func presetFlagsColumn(p *lab.Preset) string {
	var parts []string

	if p.BaseProfile != "" {
		parts = append(parts, "--base-profile="+p.BaseProfile)
	}
	if p.Branch != "" {
		parts = append(parts, "--branch="+p.Branch)
	}
	if len(p.EnvVars) > 0 {
		keys := make([]string, 0, len(p.EnvVars))
		for k := range p.EnvVars {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		parts = append(parts, "--env="+strings.Join(keys, ","))
	}
	if p.EnvFile != "" {
		parts = append(parts, "--env-file="+filepath.Base(p.EnvFile))
	}
	if len(p.Features) > 0 {
		parts = append(parts, "--feature="+strings.Join(p.Features, ","))
	}
	if p.Firewall {
		parts = append(parts, "--firewall")
	}
	if p.InitScript != "" {
		parts = append(parts, "--init-script="+filepath.Base(p.InitScript))
	}
	if p.LabName != "" {
		parts = append(parts, "--name="+p.LabName)
	}

	if len(parts) == 0 {
		return "(none)"
	}
	return strings.Join(parts, " ")
}

func newPresetShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <name>",
		Short: "Display details of a saved preset",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			mgr := lab.NewManager(baseDirFn())

			p, err := mgr.Presets().Load(name)
			if err != nil {
				if errors.Is(err, os.ErrNotExist) {
					return fmt.Errorf("preset '%s' not found", name)
				}
				return fmt.Errorf("failed to load preset %q: %w", name, err)
			}

			fmt.Printf("Preset: %s\n", p.Name)
			fields := lab.FormatPresetFields(p)
			if fields != "" {
				fmt.Println(fields)
			}

			return nil
		},
	}
}

// presetFromStartConfig creates a Preset from a lab's tracked StartConfig.
func presetFromStartConfig(name string, cfg *lab.StartConfig) *lab.Preset {
	return &lab.Preset{
		Name:        name,
		CreatedAt:   time.Now().UTC(),
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
