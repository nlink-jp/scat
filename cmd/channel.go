package cmd

import (
	"github.com/spf13/cobra"
)

// newChannelCmd creates the command for channel management.
func newChannelCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "channel",
		Short: "Manage and view channel information",
		Long:  `The channel command and its subcommands help you interact with channel-related features, such as listing available channels for a provider.`,
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}

	// Add subcommands
	cmd.AddCommand(newChannelListCmd())   // from channel_list.go
	cmd.AddCommand(newChannelCreateCmd()) // from channel_create.go
	cmd.AddCommand(newChannelInviteCmd()) // from channel_invite.go

	return cmd
}