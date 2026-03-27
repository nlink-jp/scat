package cmd

import (
	"fmt"

	"github.com/nlink-jp/scat/internal/appcontext"
	"github.com/nlink-jp/scat/internal/provider"
	"github.com/spf13/cobra"
)

func newChannelInviteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "invite [channel] [user-or-group...]",
		Short: "Invite users or user groups to a channel",
		Long: `Invite one or more users or user groups to an existing channel.

Users can be specified by display name (e.g. alice) or with an @ prefix (e.g. @alice).
User groups can be specified by handle (e.g. @team-infra).`,
		Args: cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			channelName := args[0]
			invitees := args[1:]

			appCtx := cmd.Context().Value(appcontext.CtxKey).(appcontext.Context)

			cfg := appCtx.Config
			if cfg == nil {
				return fmt.Errorf("configuration file not found. Please run 'scat config init' to create a default configuration")
			}

			profileName, _ := cmd.Flags().GetString("profile")
			if profileName == "" {
				profileName = cfg.CurrentProfile
			}
			profile, ok := cfg.Profiles[profileName]
			if !ok {
				return fmt.Errorf("profile '%s' not found", profileName)
			}

			p, err := GetProvider(appCtx, profile)
			if err != nil {
				return fmt.Errorf("failed to get provider: %w", err)
			}

			if !p.Capabilities().CanInviteToChannel {
				return fmt.Errorf("the provider for profile '%s' does not support inviting users to channels", profileName)
			}

			opts := provider.InviteToChannelOptions{
				Channel:  channelName,
				Invitees: invitees,
			}

			if err := p.InviteToChannel(opts); err != nil {
				return fmt.Errorf("failed to invite users: %w", err)
			}

			if !appCtx.Silent {
				fmt.Printf("info: Successfully invited %v to channel '%s'\n", invitees, channelName)
			}

			return nil
		},
	}

	cmd.Flags().StringP("profile", "p", "", "Profile to use for this command")

	return cmd
}
