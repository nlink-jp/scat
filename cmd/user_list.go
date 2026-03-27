package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"text/tabwriter"

	"github.com/nlink-jp/scat/internal/appcontext"
	"github.com/nlink-jp/scat/internal/provider"
	"github.com/spf13/cobra"
)

// newUserListCmd creates the command for listing users.
func newUserListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List users for supported providers",
		Long:  `Iterates through all configured profiles and lists the users for each profile whose provider supports this feature.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			appCtx := cmd.Context().Value(appcontext.CtxKey).(appcontext.Context)

			cfg := appCtx.Config
			if cfg == nil {
				return fmt.Errorf("configuration file not found. Please run 'scat config init' to create a default configuration")
			}

			jsonOutput, _ := cmd.Flags().GetBool("json")
			results := make(map[string][]provider.UserInfo)

			for profileName, profile := range cfg.Profiles {
				prov, err := GetProvider(appCtx, profile)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Warning: could not get provider for profile '%s': %v\n", profileName, err)
					continue
				}

				caps := prov.Capabilities()
				if !caps.CanListUsers {
					continue
				}

				users, err := prov.ListUsers()
				if err != nil {
					fmt.Fprintf(os.Stderr, "Warning: could not list users for profile '%s': %v\n", profileName, err)
					continue
				}

				sort.Slice(users, func(i, j int) bool {
					return users[i].Name < users[j].Name
				})

				if jsonOutput {
					results[profileName] = users
				} else {
					fmt.Fprintf(os.Stderr, "Users for profile: %s\n", profileName)
					w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
					fmt.Fprintln(w, "NAME\tID")
					for _, u := range users {
						fmt.Fprintf(w, "%s\t%s\n", u.Name, u.ID)
					}
					w.Flush()
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

	cmd.Flags().Bool("json", false, "Output the list of users in JSON format")

	return cmd
}
