package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var version = "dev"

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "claudeup-lab",
		Short: "Ephemeral devcontainer environments for testing Claude Code configurations",
		Long: `claudeup-lab creates disposable devcontainer environments for testing
Claude Code configurations (plugins, skills, agents, hooks, commands)
without affecting your host setup.`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.AddCommand(newVersionCmd())
	cmd.AddCommand(newStartCmd())
	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newExecCmd())
	cmd.AddCommand(newOpenCmd())

	return cmd
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(version)
		},
	}
}

func Execute() {
	if err := NewRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
