package cmd

import (
	"fmt"
	"os"

	"github.com/magifd2/scat/internal/appcontext"
	"github.com/magifd2/scat/internal/config"
	"github.com/spf13/cobra"
)

// newProfileAddCmd creates the command for adding a new profile.
func newProfileAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add [profile_name]",
		Short: "Add a new profile",
		Long:  `Adds a new profile. You will be prompted to enter the authentication token securely.`, 
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			appCtx := cmd.Context().Value(appcontext.CtxKey).(appcontext.Context)
			if err := requireCLIMode(appCtx); err != nil {
				return err
			}
			cfg := appCtx.Config
			if cfg == nil {
				return fmt.Errorf("configuration file not found. Please run 'scat config init' to create a default configuration before adding a profile")
			}

			profileName := args[0]

			if _, ok := cfg.Profiles[profileName]; ok {
				return fmt.Errorf("Error: Profile '%s' already exists", profileName)
			}

			provider, _ := cmd.Flags().GetString("provider")
			channel, _ := cmd.Flags().GetString("channel")
			username, _ := cmd.Flags().GetString("username")
			maxFile, _ := cmd.Flags().GetInt64("limits-max-file-size-bytes")
			maxStdin, _ := cmd.Flags().GetInt64("limits-max-stdin-size-bytes")

			newProfile := config.Profile{
				Provider: provider,
				Channel:  channel,
				Username: username,
				Limits: config.Limits{
					MaxFileSizeBytes: maxFile,
					MaxStdinSizeBytes: maxStdin,
				},
			}

			// Prompt for token securely using the new utility function
			token, err := GetPasswordFromPrompt("Enter Token (will not be displayed): ")
			if err != nil {
				return fmt.Errorf("failed to read token: %w", err)
			}
			newProfile.Token = token

			cfg.Profiles[profileName] = newProfile

			if err := cfg.Save(appCtx.ConfigPath); err != nil {
				return fmt.Errorf("Error saving config: %w", err)
			}

			fmt.Fprintf(os.Stderr, "Profile '%s' added.\n", profileName)
			return nil
		},
	}

	cmd.Flags().String("provider", "slack", "Provider type: 'slack' or 'mock'")
	cmd.Flags().String("channel", "", "Channel name or ID (for slack provider)")
	cmd.Flags().String("username", "", "Default username for posts")
	cmd.Flags().Int64("limits-max-file-size-bytes", 1024*1024*1024, "Max file size for uploads in bytes (1GB)")
	cmd.Flags().Int64("limits-max-stdin-size-bytes", 10*1024*1024, "Max size for stdin in bytes (10MB)")

	return cmd
}
