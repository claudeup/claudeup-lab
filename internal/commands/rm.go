package commands

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/claudeup/claudeup-lab/internal/lab"
	"github.com/spf13/cobra"
)

func newRmCmd() *cobra.Command {
	var labName string
	var force bool

	cmd := &cobra.Command{
		Use:   "rm",
		Short: "Destroy a lab and all its data",
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr := lab.NewManager(defaultBaseDir())
			resolver := lab.NewResolver(mgr.Store())

			meta, err := resolveLab(resolver, labName)
			if err != nil {
				return err
			}

			if !force {
				fmt.Println("This will:")
				fmt.Println("  - Stop the container")
				fmt.Printf("  - Remove Docker volumes (claudeup-lab-*-%s)\n", meta.ID)
				fmt.Printf("  - Remove worktree: %s\n", meta.Worktree)
				fmt.Println("  - Remove lab metadata")
				fmt.Println()
				if !confirm("Continue?") {
					fmt.Println("Aborted.")
					return nil
				}
			}

			err = mgr.Remove(meta, true)

			var prompt *lab.BareRepoCleanupPrompt
			if errors.As(err, &prompt) {
				fmt.Printf("\nBare repo %s has no remaining worktrees.\n", prompt.BareRepo)
				if force || confirm("Remove bare repo?") {
					os.RemoveAll(prompt.BareRepo)
					fmt.Printf("Removed bare repo: %s\n", prompt.BareRepo)
				}
				return nil
			}

			return err
		},
	}

	cmd.Flags().StringVar(&labName, "lab", "", "Lab to remove (name, UUID, project, or profile)")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "Skip confirmation prompt")

	return cmd
}

func confirm(prompt string) bool {
	fmt.Printf("%s [y/N] ", prompt)
	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}
