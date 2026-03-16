package cmd

import (
	"fmt"

	"github.com/magifd2/scat/internal/appcontext"
	"github.com/spf13/cobra"
)

// newProfileListCmd creates the command for listing profiles.
func newProfileListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all available profiles",
		Long:  `Lists all saved profiles and indicates which one is currently active.`, 
		RunE: func(cmd *cobra.Command, args []string) error {
			appCtx := cmd.Context().Value(appcontext.CtxKey).(appcontext.Context)
			if err := requireCLIMode(appCtx); err != nil {
				return err
			}
			cfg := appCtx.Config
			if cfg == nil {
				return fmt.Errorf("configuration file not found. Please run 'scat config init' to create a default configuration")
			}

			// Note: This output goes to stdout, not stderr, for easier parsing.
			for name, p := range cfg.Profiles {
				activeMarker := " "
				if name == cfg.CurrentProfile {
					activeMarker = "*"
				}
				fmt.Printf("%s %s (provider: %s)\n", activeMarker, name, p.Provider)
			}
			return nil
		},
	}
	return cmd
}