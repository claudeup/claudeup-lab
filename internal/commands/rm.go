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
	var all bool

	cmd := &cobra.Command{
		Use:   "rm",
		Short: "Destroy a lab and all its data",
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr := lab.NewManager(defaultBaseDir())

			if all {
				return removeAll(mgr, force)
			}

			resolver := lab.NewResolver(mgr.Store())

			meta, err := resolveLab(resolver, labName)
			if err != nil {
				return err
			}

			return removeOne(mgr, meta, force)
		},
	}

	cmd.Flags().StringVar(&labName, "lab", "", "Lab to remove (name, UUID, project, or profile)")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "Skip confirmation prompt")
	cmd.Flags().BoolVar(&all, "all", false, "Remove all labs")
	cmd.MarkFlagsMutuallyExclusive("all", "lab")

	return cmd
}

func removeBareRepo(repo string, force bool) {
	fmt.Printf("\nBare repo %s has no remaining worktrees.\n", repo)
	if force || confirm("Remove bare repo?") {
		if err := os.RemoveAll(repo); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to remove bare repo %s: %v\n", repo, err)
		} else {
			fmt.Printf("Removed bare repo: %s\n", repo)
		}
	}
}

func shortID(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}

func removeOne(mgr *lab.Manager, meta *lab.Metadata, force bool) error {
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

	err := mgr.Remove(meta, true)

	var prompt *lab.BareRepoCleanupPrompt
	if errors.As(err, &prompt) {
		removeBareRepo(prompt.BareRepo, force)
		return nil
	}

	return err
}

func removeAll(mgr *lab.Manager, force bool) error {
	labs, err := mgr.Store().List()
	if err != nil {
		return err
	}

	if len(labs) == 0 {
		fmt.Println("No labs found.")
		return nil
	}

	if !force {
		fmt.Printf("This will destroy %d lab(s) and all their data:\n", len(labs))
		for _, meta := range labs {
			fmt.Printf("  - %s (%s)\n", meta.DisplayName, shortID(meta.ID))
		}
		fmt.Println()
		if !confirm("Continue?") {
			fmt.Println("Aborted.")
			return nil
		}
	}

	removed := 0
	var bareRepos []string
	var errs []string
	for _, meta := range labs {
		err := mgr.Remove(meta, true)

		var prompt *lab.BareRepoCleanupPrompt
		if errors.As(err, &prompt) {
			bareRepos = append(bareRepos, prompt.BareRepo)
		} else if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", meta.DisplayName, err))
			continue
		}
		removed++
	}

	fmt.Printf("\nRemoved %d lab(s).\n", removed)

	for _, repo := range bareRepos {
		removeBareRepo(repo, force)
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to remove some labs:\n  %s", strings.Join(errs, "\n  "))
	}
	return nil
}

func confirm(prompt string) bool {
	fmt.Printf("%s [y/N] ", prompt)
	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}
