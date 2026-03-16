package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/magifd2/scat/internal/appcontext"
	"github.com/spf13/cobra"
)

// newChannelListCmd creates the command for listing channels.
func newChannelListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available channels for supported providers",
		Long:  `Iterates through all configured profiles and lists the available channels for each profile whose provider supports this feature.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			appCtx := cmd.Context().Value(appcontext.CtxKey).(appcontext.Context)

			cfg := appCtx.Config
			if cfg == nil {
				return fmt.Errorf("configuration file not found. Please run 'scat config init' to create a default configuration")
			}

			jsonOutput, _ := cmd.Flags().GetBool("json")
			results := make(map[string][]string)

			for profileName, profile := range cfg.Profiles {
				prov, err := GetProvider(appCtx, profile) // noop is false for listing
				if err != nil {
					fmt.Fprintf(os.Stderr, "Warning: could not get provider for profile '%s': %v\n", profileName, err)
					continue
				}

				caps := prov.Capabilities()
				if !caps.CanListChannels {
					continue // Skip providers that don't support listing channels
				}

				channels, err := prov.ListChannels()
				if err != nil {
					fmt.Fprintf(os.Stderr, "Warning: could not list channels for profile '%s': %v\n", profileName, err)
					continue
				}

				if jsonOutput {
					results[profileName] = channels
				} else {
					fmt.Fprintf(os.Stderr, "Channels for profile: %s\n", profileName)
					for _, ch := range channels {
						fmt.Printf("  - %s\n", ch)
					}
				}
			}

			if jsonOutput {
				jsonBytes, err := json.MarshalIndent(results, "", "  ")
				if err != nil {
					return fmt.Errorf("failed to marshal results to json: %w", err)
				}
				fmt.Println(string(jsonBytes))
			}

			return nil
		},
	}

	cmd.Flags().Bool("json", false, "Output the list of channels in JSON format")

	return cmd
}