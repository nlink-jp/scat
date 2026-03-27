package cmd

import (
	"fmt"

	"github.com/nlink-jp/scat/internal/appcontext"
	"github.com/nlink-jp/scat/internal/config"
	"github.com/spf13/cobra"
)

func newConfigInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a new configuration file",
		Long:  `Creates a new default configuration file at the default location (~/.config/scat/config.json). If a configuration file already exists, this command will do nothing.`, 
		RunE: func(cmd *cobra.Command, args []string) error {
			appCtx := cmd.Context().Value(appcontext.CtxKey).(appcontext.Context)
			if err := requireCLIMode(appCtx); err != nil {
				return err
			}

			// Guardrail: config was already loaded at startup; non-nil means file exists.
			if appCtx.Config != nil {
				fmt.Printf("Configuration file already exists at: %s\n", appCtx.ConfigPath)
				return nil
			}

			// File does not exist, so create a new one.
			cfg := config.NewDefaultConfig()
			if err := cfg.Save(appCtx.ConfigPath); err != nil {
				return fmt.Errorf("failed to save new configuration file: %w", err)
			}

			fmt.Printf("Successfully created a new configuration file at: %s\n", appCtx.ConfigPath)
			return nil
		},
	}
	return cmd
}