package commands

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/claudeup/claudeup-lab/internal/docker"
	"github.com/claudeup/claudeup-lab/internal/lab"
	"github.com/spf13/cobra"
)

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check system health and prerequisites",
		RunE: func(cmd *cobra.Command, args []string) error {
			issues := 0

			// Docker
			client := docker.NewClient()
			if client.IsRunning() {
				fmt.Println("[OK] Docker is running")
			} else {
				fmt.Println("[FAIL] Docker is not running")
				issues++
			}

			// devcontainer CLI
			if path, err := exec.LookPath("devcontainer"); err == nil {
				fmt.Printf("[OK] devcontainer CLI found: %s\n", path)
			} else {
				fmt.Println("[FAIL] devcontainer CLI not found (install: npm install -g @devcontainers/cli)")
				issues++
			}

			// Git
			if _, err := exec.LookPath("git"); err == nil {
				fmt.Println("[OK] git found")
			} else {
				fmt.Println("[FAIL] git not found")
				issues++
			}

			// claudeup (optional)
			if _, err := exec.LookPath("claudeup"); err == nil {
				fmt.Println("[OK] claudeup found")
			} else {
				fmt.Println("[WARN] claudeup not found (needed for profile snapshotting)")
			}

			// Base image
			im := docker.NewImageManager()
			image := docker.ImageTag()
			if im.ExistsLocally(image) {
				fmt.Printf("[OK] Base image available: %s\n", image)
			} else {
				fmt.Printf("[WARN] Base image not found locally: %s (will be pulled on first start)\n", image)
			}

			// Orphaned labs
			mgr := lab.NewManager(defaultBaseDir())
			labs, _ := mgr.Store().List()
			orphaned := 0
			for _, m := range labs {
				if _, err := os.Stat(m.Worktree); os.IsNotExist(err) {
					orphaned++
					fmt.Printf("[WARN] Orphaned lab: %s (%s) -- worktree missing\n", m.DisplayName, m.ID[:8])
				}
			}
			if orphaned == 0 && len(labs) > 0 {
				fmt.Printf("[OK] %d lab(s) found, no orphans\n", len(labs))
			} else if len(labs) == 0 {
				fmt.Println("[OK] No labs found")
			}

			fmt.Println()
			if issues > 0 {
				fmt.Printf("%d issue(s) found.\n", issues)
				return fmt.Errorf("doctor found %d issue(s)", issues)
			}
			fmt.Println("All checks passed.")
			return nil
		},
	}
}
