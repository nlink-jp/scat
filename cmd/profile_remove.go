
package cmd

import (
	"fmt"
	"os"

	"github.com/nlink-jp/scat/internal/appcontext"
	"github.com/spf13/cobra"
)

// newProfileRemoveCmd creates the command for removing a profile.
func newProfileRemoveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove [profile_name]",
		Short: "Remove a profile",
		Long:  `Removes a specified profile from the configuration.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			appCtx := cmd.Context().Value(appcontext.CtxKey).(appcontext.Context)
			if err := requireCLIMode(appCtx); err != nil {
				return err
			}
			cfg := appCtx.Config
			if cfg == nil {
				return fmt.Errorf("configuration file not found. Please run 'scat config init' to create a default configuration")
			}

			profileName := args[0]

			if profileName == "default" {
				return fmt.Errorf("the 'default' profile cannot be removed")
			}

			if _, ok := cfg.Profiles[profileName]; !ok {
				return fmt.Errorf("profile '%s' not found", profileName)
			}

			if cfg.CurrentProfile == profileName {
				return fmt.Errorf("cannot remove the currently active profile. Please switch to another profile first")
			}

			delete(cfg.Profiles, profileName)

			if err := cfg.Save(appCtx.ConfigPath); err != nil {
				return fmt.Errorf("saving config: %w", err)
			}
			fmt.Fprintf(os.Stderr, "Profile '%s' removed.\n", profileName)
			return nil
		},
	}
	return cmd
}
