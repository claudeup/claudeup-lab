package commands

import (
	"fmt"

	"github.com/claudeup/claudeup-lab/internal/lab"
	"github.com/spf13/cobra"
)

func newStopCmd() *cobra.Command {
	var labName string
	var all bool

	cmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop a running lab (volumes persist)",
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr := lab.NewManager(defaultBaseDir())

			if all {
				return stopAll(mgr)
			}

			resolver := lab.NewResolver(mgr.Store())

			meta, err := resolveLab(resolver, labName)
			if err != nil {
				return err
			}

			return stopOne(mgr, meta)
		},
	}

	cmd.Flags().StringVar(&labName, "lab", "", "Lab to stop (name, UUID, project, or profile)")
	cmd.Flags().BoolVar(&all, "all", false, "Stop all running labs")

	return cmd
}

func stopOne(mgr *lab.Manager, meta *lab.Metadata) error {
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
}

func stopAll(mgr *lab.Manager) error {
	labs, err := mgr.Store().List()
	if err != nil {
		return err
	}

	if len(labs) == 0 {
		fmt.Println("No labs found.")
		return nil
	}

	stopped := 0
	for _, meta := range labs {
		containerID, _ := mgr.Docker().FindContainer(meta.Worktree)
		if containerID == "" {
			continue
		}
		if err := mgr.Docker().StopContainer(containerID); err != nil {
			fmt.Printf("Failed to stop %s: %v\n", meta.DisplayName, err)
			continue
		}
		fmt.Printf("Stopped lab: %s\n", meta.DisplayName)
		stopped++
	}

	if stopped == 0 {
		fmt.Println("No running labs found.")
	} else {
		fmt.Printf("\nStopped %d lab(s).\n", stopped)
	}
	return nil
}
