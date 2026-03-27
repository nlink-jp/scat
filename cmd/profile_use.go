package cmd

import (
	"fmt"
	"os"

	"github.com/nlink-jp/scat/internal/appcontext"
	"github.com/spf13/cobra"
)

// newProfileUseCmd creates the command for setting the active profile.
func newProfileUseCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "use [profile_name]",
		Short: "Set the active profile",
		Long:  `Set the active profile for scat.`,
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
			if _, ok := cfg.Profiles[profileName]; !ok {
				return fmt.Errorf("profile '%s' not found", profileName)
			}

			cfg.CurrentProfile = profileName
			if err := cfg.Save(appCtx.ConfigPath); err != nil {
				return fmt.Errorf("saving config: %w", err)
			}
			fmt.Fprintf(os.Stderr, "Switched to profile: %s\n", profileName)
			return nil
		},
	}
	return cmd
}