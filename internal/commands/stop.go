package commands

import (
	"fmt"

	"github.com/claudeup/claudeup-lab/internal/lab"
	"github.com/spf13/cobra"
)

func newStopCmd() *cobra.Command {
	var labName string

	cmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop a running lab (volumes persist)",
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr := lab.NewManager(defaultBaseDir())
			resolver := lab.NewResolver(mgr.Store())

			meta, err := resolveLab(resolver, labName)
			if err != nil {
				return err
			}

			fmt.Printf("Stopping lab: %s...\n", meta.DisplayName)

			containerID, err := mgr.Docker().FindContainer(meta.Worktree)
			if err != nil {
				return err
			}

			if containerID == "" {
				fmt.Printf("No running container found for lab: %s\n", meta.DisplayName)
				return nil
			}

			if err := mgr.Docker().StopContainer(containerID); err != nil {
				return err
			}

			fmt.Printf("Stopped lab: %s\n", meta.DisplayName)
			return nil
		},
	}

	cmd.Flags().StringVar(&labName, "lab", "", "Lab to stop (name, UUID, project, or profile)")

	return cmd
}
