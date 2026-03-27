
package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/nlink-jp/scat/internal/appcontext"
	"github.com/nlink-jp/scat/internal/export"
	"github.com/spf13/cobra"
)

// newExportLogCmd creates the command for exporting channel logs.
func newExportLogCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "log",
		Short: "Export a channel log",
		Long:  `Exports a channel log from a supported provider, saving messages and optionally files to a local directory.`,
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

			// Get provider
			prov, err := GetProvider(appCtx, profile)
			if err != nil {
				return err
			}

			// Check capability
			if !prov.Capabilities().CanExportLogs {
				return fmt.Errorf("the provider for profile '%s' does not support exporting logs", profileName)
			}

			// Get flags
			channelName, _ := cmd.Flags().GetString("channel")
			startTimeStr, _ := cmd.Flags().GetString("start-time")
			endTimeStr, _ := cmd.Flags().GetString("end-time")
			outputFile, _ := cmd.Flags().GetString("output")
			outputFiles, _ := cmd.Flags().GetString("output-files")
			outputFormat, _ := cmd.Flags().GetString("output-format")

			// Determine file output behavior
			includeFiles := outputFiles != ""
			filesDir := ""
			if includeFiles {
				if outputFiles == "auto" {
					filesDir = fmt.Sprintf("./scat-export-%s-%s", strings.TrimPrefix(channelName, "#"), time.Now().UTC().Format("20060102T150405Z"))
				} else {
					filesDir = outputFiles
				}
				if err := os.MkdirAll(filesDir, 0700); err != nil {
					return fmt.Errorf("failed to create files directory %s: %w", filesDir, err)
				}
			}

			// Parse timestamps
			startTime, err := parseTime(startTimeStr)
			if err != nil {
				return fmt.Errorf("invalid start time: %w", err)
			}
			endTime, err := parseTime(endTimeStr)
			if err != nil {
				return fmt.Errorf("invalid end time: %w", err)
			}

			if !appCtx.Silent {
				var timeRangeStr strings.Builder
				timeRangeStr.WriteString("for all time")
				if !startTime.IsZero() || !endTime.IsZero() {
					timeRangeStr.Reset()
					timeRangeStr.WriteString(fmt.Sprintf("from %s to %s (UTC: %s to %s)",
						displayTime(startTime, "(beginning of time)"), displayTime(endTime, "now"),
						displayTime(startTime.UTC(), "(beginning of time)"), displayTime(endTime.UTC(), "now")))
				}
				fmt.Fprintf(os.Stderr, "Exporting messages for channel %s %s\n", channelName, timeRangeStr.String())
			}

			// Create options and run the export
			opts := export.Options{
				ChannelName:  channelName,
				StartTime:    toUnixTimestampString(startTime),
				EndTime:      toUnixTimestampString(endTime),
				IncludeFiles: includeFiles,
				OutputDir:    filesDir,
			}

			exportedLog, err := prov.ExportLog(opts)
			if err != nil {
				return fmt.Errorf("failed to export log: %w", err)
			}

			// Save the log to the specified output
			if err := saveExportedLog(exportedLog, outputFile, outputFormat); err != nil {
				return err
			}

			// Construct and print the final status message
			if !appCtx.Silent {
				var parts []string
				parts = append(parts, "Log export completed successfully.")
				if outputFile != "-" && outputFile != "" {
					parts = append(parts, fmt.Sprintf("Log saved to %s.", outputFile))
				}
				if includeFiles {
					parts = append(parts, fmt.Sprintf("Files saved in %s.", filesDir))
				}
				fmt.Fprintln(os.Stderr, strings.Join(parts, " "))
			}

			return nil
		},
	}

	cmd.Flags().StringP("profile", "p", "", "Profile to use for this export")
	cmd.Flags().StringP("channel", "c", "", "Channel to export from (required)")
	_ = cmd.MarkFlagRequired("channel")

	cmd.Flags().String("output", "-", "Output file path for the log. Use '-' for stdout.")
	cmd.Flags().String("output-files", "", "Directory to save downloaded files. If set to 'auto', a directory is auto-generated.")
	cmd.Flags().String("output-format", "json", "Output format (json or text)")
	cmd.Flags().String("start-time", "", "Start of time range (RFC3339 format, e.g., 2023-01-01T15:04:05Z)")
	cmd.Flags().String("end-time", "", "End of time range (RFC3339 format)")

	return cmd
}

func saveExportedLog(log *export.ExportedLog, outputFile, format string) error {
	// Determine output writer
	var writer io.Writer
	if outputFile == "-" || outputFile == "" {
		writer = os.Stdout
	} else {
		// Use OpenFile to create with specific permissions
		f, err := os.OpenFile(outputFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer f.Close()
		writer = f
	}

	switch format {
	case "json":
		encoder := json.NewEncoder(writer)
		encoder.SetIndent("", "  ")
		return encoder.Encode(log)
	case "text":
		var content strings.Builder
		content.WriteString(fmt.Sprintf("# Log export for channel %s on %s\n", log.ChannelName, log.ExportTimestamp))
		for _, msg := range log.Messages {
			content.WriteString("---\n")
			indent := ""
			if msg.IsReply {
				indent = "    " // 4 spaces for indentation
			}
			content.WriteString(fmt.Sprintf("%s[%s] %s: %s\n", indent, msg.Timestamp, msg.UserName, msg.Text))
			for _, file := range msg.Files {
				content.WriteString(fmt.Sprintf("%s  - Attachment: %s (saved to: %s)\n", indent, file.Name, file.LocalPath))
			}
		}
		_, err := writer.Write([]byte(content.String()))
		return err
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}

// parseTime parses a string into a time.Time object.
// It accepts RFC3339 format or a local time format.
func parseTime(timeStr string) (time.Time, error) {
	if timeStr == "" {
		return time.Time{}, nil
	}
	// Try parsing with timezone first
	t, err := time.Parse(time.RFC3339, timeStr)
	if err == nil {
		return t, nil
	}
	// If that fails, try parsing as local time
	return time.Parse("2006-01-02T15:04:05", timeStr)
}

// displayTime formats a time for display, showing a fallback if the time is zero.
func displayTime(t time.Time, fallback string) string {
	if t.IsZero() {
		return fallback
	}
	return t.Format(time.RFC3339)
}

// toUnixTimestampString converts a time.Time object to a Slack-compatible Unix timestamp string.
func toUnixTimestampString(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return fmt.Sprintf("%d.000000", t.Unix())
}
