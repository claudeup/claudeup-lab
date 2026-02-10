package commands

import (
	"fmt"

	"github.com/claudeup/claudeup-lab/internal/lab"
	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Short:   "Show all labs and their status",
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr := lab.NewManager(defaultBaseDir())

			labs, err := mgr.Store().List()
			if err != nil {
				return err
			}

			if len(labs) == 0 {
				fmt.Println("No labs found.")
				return nil
			}

			fmt.Printf("%-30s %-10s %-20s %-15s %s\n", "NAME", "ID", "PROJECT", "PROFILE", "STATUS")
			fmt.Printf("%-30s %-10s %-20s %-15s %s\n", "----", "--", "-------", "-------", "------")

			for _, m := range labs {
				status := mgr.LabStatus(m)
				fmt.Printf("%-30s %-10s %-20s %-15s %s\n",
					m.DisplayName, m.ID[:8], m.ProjectName, m.Profile, status)
			}

			return nil
		},
	}
}
