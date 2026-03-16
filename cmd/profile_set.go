package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/magifd2/scat/internal/appcontext"
	"github.com/spf13/cobra"
)

func newProfileSetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set [key] [value]",
		Short: "Set a value in the current profile",
		Long: `Set a configuration value for the currently active profile.
For the 'token' key, run 'scat profile set token' and you will be prompted to enter the value securely.`, 
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			appCtx := cmd.Context().Value(appcontext.CtxKey).(appcontext.Context)
			if err := requireCLIMode(appCtx); err != nil {
				return err
			}
			cfg := appCtx.Config
			if cfg == nil {
				return fmt.Errorf("configuration file not found. Please run 'scat config init' to create a default configuration")
			}

			key := args[0]

			profile, ok := cfg.Profiles[cfg.CurrentProfile]
			if !ok {
				return fmt.Errorf("active profile '%s' not found", cfg.CurrentProfile)
			}

			var value string
			if key == "token" {
				if len(args) != 1 {
					return fmt.Errorf("'set token' does not accept a value argument. Run it alone to be prompted.")
				}
				// Use the new utility function
				token, err := GetPasswordFromPrompt("Enter new Token (will not be displayed): ")
				if err != nil {
					return fmt.Errorf("failed to read token: %w", err)
				}
				value = token
			} else {
				if len(args) != 2 {
					return fmt.Errorf("key '%s' requires a value", key)
				}
				value = args[1]
			}

			switch key {
			case "provider":
				if value != "mock" && value != "slack" {
					return fmt.Errorf("invalid provider '%s'. avalid values are 'mock' or 'slack'", value)
				}
				profile.Provider = value
			case "channel":
				profile.Channel = value
			case "token":
				profile.Token = value
			case "username":
				profile.Username = value
			case "limits.max_file_size_bytes":
				size, err := strconv.ParseInt(value, 10, 64)
				if err != nil {
					return fmt.Errorf("invalid integer value for %s: %s", key, value)
				}
				profile.Limits.MaxFileSizeBytes = size
			case "limits.max_stdin_size_bytes":
				size, err := strconv.ParseInt(value, 10, 64)
				if err != nil {
					return fmt.Errorf("invalid integer value for %s: %s", key, value)
				}
				profile.Limits.MaxStdinSizeBytes = size
			default:
				availableKeys := []string{"provider", "channel", "token", "username", "limits.max_file_size_bytes", "limits.max_stdin_size_bytes"}
				return fmt.Errorf("unknown configuration key '%s'.\nAvailable keys: %s", key, strings.Join(availableKeys, ", "))
			}

			cfg.Profiles[cfg.CurrentProfile] = profile
			if err := cfg.Save(appCtx.ConfigPath); err != nil {
				return fmt.Errorf("saving config: %w", err)
			}
			fmt.Fprintf(os.Stderr, "Set %s in profile %s\n", key, cfg.CurrentProfile)
			return nil
		},
	}
	return cmd
}