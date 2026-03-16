package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/magifd2/scat/internal/appcontext"
	"github.com/magifd2/scat/internal/provider"
	"github.com/spf13/cobra"
)

// newPostCmd creates the command for posting messages.
func newPostCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "post [message text]",
		Short: "Post a text message from an argument, file, or stdin",
		Long:  `Posts a text message. The message content is sourced in the following order of precedence: 1. Command-line argument. 2. --from-file flag. 3. Standard input.`, 
		RunE: func(cmd *cobra.Command, args []string) error {
			appCtx := cmd.Context().Value(appcontext.CtxKey).(appcontext.Context)

			cfg := appCtx.Config
			if cfg == nil {
				return fmt.Errorf("configuration file not found. Please run 'scat config init' to create a default configuration")
			}

			// Determine profile
			profileName, _ := cmd.Flags().GetString("profile")
			if profileName == "" {
				profileName = cfg.CurrentProfile
			}
			profile, ok := cfg.Profiles[profileName]
			if !ok {
				return fmt.Errorf("profile '%s' not found", profileName)
			}

			// Get optional flags
			username, _ := cmd.Flags().GetString("username")
			iconEmoji, _ := cmd.Flags().GetString("iconemoji")
			channel, _ := cmd.Flags().GetString("channel")
			user, _ := cmd.Flags().GetString("user")
			tee, _ := cmd.Flags().GetBool("tee")
			fromFile, _ := cmd.Flags().GetString("from-file")
			format, _ := cmd.Flags().GetString("format")

			// --- Flag Validation and Exclusive Handling ---
			if user != "" && channel != "" {
				return fmt.Errorf("cannot use --user and --channel flags simultaneously")
			}

			// Get provider instance
			prov, err := GetProvider(appCtx, profile)
			if err != nil {
				return err
			}

			stream, _ := cmd.Flags().GetBool("stream")

			// Validate format flag value
			if format != "text" && format != "blocks" {
				return fmt.Errorf("invalid value for --format: %s. Must be 'text' or 'blocks'", format)
			}

			// Exclusive handling for --stream and --format blocks
			if stream && format == "blocks" {
				return fmt.Errorf("cannot use --stream with --format blocks")
			}

			if stream {
				return handleStream(prov, channel, user, profileName, username, iconEmoji, tee, appCtx.Silent)
			}

			// --- Determine message content and format ---
			var content string
			var blocks json.RawMessage

			// Read content from args, file, or stdin
			if len(args) > 0 {
				content = strings.Join(args, " ")
			} else if fromFile != "" {
				fileContent, err := os.ReadFile(fromFile)
				if err != nil {
					return fmt.Errorf("failed to read from file %s: %w", fromFile, err)
				}
				content = string(fileContent)
			} else {
				stat, _ := os.Stdin.Stat()
				if (stat.Mode() & os.ModeCharDevice) == 0 {
					limit := profile.Limits.MaxStdinSizeBytes
					var limitedReader io.Reader = os.Stdin
					if limit > 0 {
						limitedReader = io.LimitReader(os.Stdin, limit+1)
					}
					stdinContent, err := io.ReadAll(limitedReader)
					if err != nil {
						return fmt.Errorf("failed to read from stdin: %w", err)
					}
					if limit > 0 && int64(len(stdinContent)) > limit {
						return fmt.Errorf("stdin size exceeds the configured limit (%d bytes)", limit)
					}
					content = string(stdinContent)
				} else {
					return fmt.Errorf("no message content provided via argument, --from-file, or stdin")
				}
			}

			// If format is blocks, parse content as JSON
			if format == "blocks" {
				// Attempt to unmarshal into a temporary map to check for the "blocks" key
				var tempMap map[string]json.RawMessage
				if err := json.Unmarshal([]byte(content), &tempMap); err != nil {
					// If it's not a map, or unmarshalling fails, try to unmarshal directly as an array
					var tempArray []interface{}
					if err := json.Unmarshal([]byte(content), &tempArray); err != nil {
						return fmt.Errorf("failed to parse block kit JSON: expected a JSON object with a 'blocks' key or a JSON array of blocks: %w", err)
					}
					// If it's a direct array, use the content as is
					blocks = json.RawMessage(content)
				} else if rawBlocks, ok := tempMap["blocks"]; ok {
					// If it's a map with a "blocks" key, extract the value of "blocks"
					blocks = rawBlocks
				} else {
					// If it's a map but no "blocks" key, it's an invalid format for Block Kit
					return fmt.Errorf("failed to parse block kit JSON: expected a JSON object with a 'blocks' key or a JSON array of blocks")
				}
			}

			// Tee output if requested (only for stdin, and not for blocks as it's structured data)
			if tee && fromFile == "" && len(args) == 0 && format == "text" { // only tee stdin for text format
				fmt.Print(content)
			}

			// Post the message
			opts := provider.PostMessageOptions{
				TargetChannel:    channel,
				TargetUserID:     user,
				Text:             content,
				OverrideUsername: username,
				IconEmoji:        iconEmoji,
				Blocks:           blocks,
			}
			// If blocks are present, clear text to ensure blocks are prioritized by provider
			if len(opts.Blocks) > 0 {
				opts.Text = ""
			}

			// Check if provider supports blocks if format is blocks
			if format == "blocks" && !prov.Capabilities().CanPostBlocks {
				return fmt.Errorf("the provider for profile '%s' does not support posting Block Kit messages", profileName)
			}

			if err := prov.PostMessage(opts); err != nil {
				return fmt.Errorf("failed to post message: %w", err)
			}
			if !appCtx.Silent {
				fmt.Fprintf(os.Stderr, "Message posted successfully to profile '%s'.\n", profileName)
			}

			return nil
		},
	}

	cmd.Flags().StringP("profile", "p", "", "Profile to use for this post")
	cmd.Flags().StringP("channel", "c", "", "Override the destination channel for this post")
	cmd.Flags().String("user", "", "Send a direct message to a user by ID")
	cmd.Flags().String("from-file", "", "Read message body from a file")
	cmd.Flags().BoolP("stream", "s", false, "Stream messages from stdin continuously")
	cmd.Flags().BoolP("tee", "t", false, "Print stdin to screen before posting")
	cmd.Flags().StringP("username", "u", "", "Override the username for this post")
	cmd.Flags().StringP("iconemoji", "i", "", "Icon emoji to use for the post (slack provider only)")
	cmd.Flags().String("format", "text", "Message format (text or blocks)")

	return cmd
}

func handleStream(prov provider.Interface, channel, user, profileName, overrideUsername, iconEmoji string, tee bool, silent bool) error {
	if !silent {
		fmt.Fprintf(os.Stderr, "Starting stream to profile '%s'. Press Ctrl+C to exit.\n", profileName)
	}
	lines := make(chan string)
	scanner := bufio.NewScanner(os.Stdin)

	go func() {
		for scanner.Scan() {
			line := scanner.Text()
			if tee {
				fmt.Println(line)
			}
			lines <- line
		}
		close(lines)
	}()

	var buffer []string
	ticker := CreateTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case line, ok := <-lines:
			if !ok {
				if len(buffer) > 0 {
					fmt.Fprintf(os.Stderr, "Flushing %d remaining lines...\n", len(buffer))
					opts := provider.PostMessageOptions{
						TargetChannel:    channel,
						TargetUserID:     user,
						Text:             strings.Join(buffer, "\n"),
						OverrideUsername: overrideUsername,
						IconEmoji:        iconEmoji,
					}
							if err := prov.PostMessage(opts); err != nil {
								fmt.Fprintf(os.Stderr, "Error flushing remaining lines: %v\n", err)
							}
					}
					if !silent {
						fmt.Fprintln(os.Stderr, "Stream finished.")
					}
					return nil
			}
			buffer = append(buffer, line)
		case <-ticker.C:
			if len(buffer) > 0 {
				opts := provider.PostMessageOptions{
					TargetChannel:    channel,
					TargetUserID:     user,
					Text:             strings.Join(buffer, "\n"),
					OverrideUsername: overrideUsername,
					IconEmoji:        iconEmoji,
				}
							if err := prov.PostMessage(opts); err != nil {
								fmt.Fprintf(os.Stderr, "Error posting message: %v\n", err)
							}
							if !silent {
								fmt.Fprintf(os.Stderr, "Posted %d lines to profile '%s'.\n", len(buffer), profileName)
							}
					buffer = nil
				}
		}
	}
}