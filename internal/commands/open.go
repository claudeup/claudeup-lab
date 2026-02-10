package commands

import (
	"encoding/hex"
	"fmt"
	"os/exec"

	"github.com/claudeup/claudeup-lab/internal/lab"
	"github.com/spf13/cobra"
)

func newOpenCmd() *cobra.Command {
	var labName string

	cmd := &cobra.Command{
		Use:   "open",
		Short: "Attach VS Code to a running lab",
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, err := exec.LookPath("code"); err != nil {
				return fmt.Errorf("VS Code CLI 'code' not found (see: https://code.visualstudio.com/docs/setup/mac)")
			}

			mgr := lab.NewManager(defaultBaseDir())
			resolver := lab.NewResolver(mgr.Store())

			meta, err := resolveLab(resolver, labName)
			if err != nil {
				return err
			}

			containerID, err := mgr.Docker().ContainerHostname(meta.Worktree)
			if err != nil {
				return fmt.Errorf("could not get container ID -- is the lab running? %w", err)
			}

			hexID := hex.EncodeToString([]byte(containerID))
			uri := fmt.Sprintf("vscode-remote://attached-container+%s/workspaces/%s", hexID, meta.DisplayName)

			codeCmd := exec.Command("code", "--folder-uri", uri)
			if err := codeCmd.Run(); err != nil {
				return fmt.Errorf("open VS Code: %w", err)
			}

			fmt.Printf("VS Code attached to lab: %s (%s)\n", meta.DisplayName, meta.ID[:8])
			return nil
		},
	}

	cmd.Flags().StringVar(&labName, "lab", "", "Lab to open (name, UUID, project, or profile)")

	return cmd
}
