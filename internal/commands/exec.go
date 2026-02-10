package commands

import (
	"os"
	"os/exec"

	"github.com/claudeup/claudeup-lab/internal/lab"
	"github.com/spf13/cobra"
)

func newExecCmd() *cobra.Command {
	var labName string

	cmd := &cobra.Command{
		Use:   "exec [-- command...]",
		Short: "Run a command inside a running lab",
		Long:  "Run a command inside a running lab. Without arguments after --, opens an interactive bash shell.",
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr := lab.NewManager(defaultBaseDir())
			resolver := lab.NewResolver(mgr.Store())

			meta, err := resolveLab(resolver, labName)
			if err != nil {
				return err
			}

			execArgs := []string{"exec", "--workspace-folder", meta.Worktree}

			// Args after "--" are the command to run
			dashIdx := cmd.ArgsLenAtDash()
			if dashIdx >= 0 && dashIdx < len(args) {
				execArgs = append(execArgs, args[dashIdx:]...)
			} else {
				execArgs = append(execArgs, "bash")
			}

			devCmd := exec.Command("devcontainer", execArgs...)
			devCmd.Stdin = os.Stdin
			devCmd.Stdout = os.Stdout
			devCmd.Stderr = os.Stderr
			return devCmd.Run()
		},
	}

	cmd.Flags().StringVar(&labName, "lab", "", "Lab to exec into (name, UUID, project, or profile)")

	return cmd
}
