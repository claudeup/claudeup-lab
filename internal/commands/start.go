package commands

import (
	"fmt"
	"os"

	"github.com/claudeup/claudeup-lab/internal/lab"
	"github.com/spf13/cobra"
)

func newStartCmd() *cobra.Command {
	var opts lab.StartOptions
	var features []string

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Create and start a lab",
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.Project == "" {
				cwd, err := os.Getwd()
				if err != nil {
					return fmt.Errorf("get working directory: %w", err)
				}
				opts.Project = cwd
			}
			opts.Features = features

			mgr := lab.NewManager(defaultBaseDir())

			meta, err := mgr.Start(&opts)
			if err != nil {
				return err
			}

			fmt.Println()
			fmt.Println("Lab ready!")
			fmt.Printf("  Name:     %s\n", meta.DisplayName)
			fmt.Printf("  ID:       %s\n", meta.ID[:8])
			fmt.Printf("  Worktree: %s\n", meta.Worktree)
			fmt.Printf("  Branch:   %s\n", meta.Branch)
			fmt.Printf("  Profile:  %s\n", meta.Profile)
			fmt.Println()
			fmt.Println("Next steps:")
			fmt.Printf("  claudeup-lab exec   --lab %s -- <command>\n", meta.DisplayName)
			fmt.Printf("  claudeup-lab exec   --lab %s -- claude\n", meta.DisplayName)
			fmt.Printf("  claudeup-lab open   --lab %s\n", meta.DisplayName)
			fmt.Printf("  claudeup-lab stop   --lab %s\n", meta.DisplayName)
			return nil
		},
	}

	cmd.Flags().StringVar(&opts.Project, "project", "", "Project directory (default: current directory)")
	cmd.Flags().StringVar(&opts.Profile, "profile", "", "claudeup profile (default: snapshot current config)")
	cmd.Flags().StringVar(&opts.Branch, "branch", "", "Git branch name (default: lab/<profile>)")
	cmd.Flags().StringVar(&opts.Name, "name", "", "Display name for the lab")
	cmd.Flags().StringSliceVar(&features, "feature", nil, "Devcontainer feature (repeatable, e.g. go:1.23)")
	cmd.Flags().StringVar(&opts.BaseProfile, "base-profile", "", "Apply base profile before main profile")

	return cmd
}
