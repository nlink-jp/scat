package cmd

import (
	"github.com/spf13/cobra"
)

// newUserCmd creates the command for user management.
func newUserCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "user",
		Short: "Manage and view user information",
		Long:  `The user command and its subcommands help you interact with user-related features, such as listing users.`,
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}

	cmd.AddCommand(newUserListCmd()) // from user_list.go

	return cmd
}
