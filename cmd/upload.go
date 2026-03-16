package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/magifd2/scat/internal/appcontext"
	"github.com/magifd2/scat/internal/provider"
	"github.com/spf13/cobra"
)

// newUploadCmd creates the command for uploading files.
func newUploadCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "upload",
		Short: "Upload a file from a path or stdin",
		Long:  `Uploads a file as a multipart/form-data request. The file content is sourced from the path specified in the --file flag, or from stdin if --file is set to "-".`,
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
			channel, _ := cmd.Flags().GetString("channel")
			user, _ := cmd.Flags().GetString("user")
			filePath, _ := cmd.Flags().GetString("file")

			// --- Flag Validation and Exclusive Handling ---
			if user != "" && channel != "" {
				return fmt.Errorf("cannot use --user and --channel flags simultaneously")
			}

			// Get provider instance
			prov, err := GetProvider(appCtx, profile)
			if err != nil {
				return err
			}

			// Handle file upload from path or stdin
			filename, _ := cmd.Flags().GetString("filename")
			filetype, _ := cmd.Flags().GetString("filetype")
			comment, _ := cmd.Flags().GetString("comment")

			if filePath == "-" {
				// Reading from stdin requires creating a temporary file
				tmpFile, err := os.CreateTemp("", "scat-stdin-")
				if err != nil {
					return fmt.Errorf("failed to create temp file for stdin: %w", err)
				}
				defer os.Remove(tmpFile.Name())

				limit := profile.Limits.MaxStdinSizeBytes
				var limitedReader io.Reader = os.Stdin
				if limit > 0 {
					limitedReader = io.LimitReader(os.Stdin, limit+1)
				}
				written, err := io.Copy(tmpFile, limitedReader)
				if err != nil {
					return fmt.Errorf("failed to write stdin to temp file: %w", err)
				}
				if limit > 0 && written > limit {
					return fmt.Errorf("stdin size exceeds the configured limit (%d bytes)", limit)
				}
				filePath = tmpFile.Name()
				if filename == "" {
					filename = "stdin-upload"
				}
			} else {
				// Check file size before proceeding
				fileInfo, err := os.Stat(filePath)
					if err != nil {
						return fmt.Errorf("failed to get file info: %w", err)
					}
					if profile.Limits.MaxFileSizeBytes > 0 && fileInfo.Size() > profile.Limits.MaxFileSizeBytes {
						return fmt.Errorf("file size (%d bytes) exceeds the configured limit (%d bytes)", fileInfo.Size(), profile.Limits.MaxFileSizeBytes)
					}
				if filename == "" {
					filename = filePath
				}
			}

			opts := provider.PostFileOptions{
				TargetChannel: channel,
				TargetUserID:  user,
				FilePath:      filePath,
				Filename:      filename,
				Filetype:      filetype,
				Comment:       comment,
			}
			if err := prov.PostFile(opts); err != nil {
				return fmt.Errorf("failed to post file: %w", err)
			}
			if !appCtx.Silent {
				fmt.Fprintf(os.Stderr, "File '%s' uploaded successfully to profile '%s'.\n", filename, profileName)
			}

			return nil
		},
	}

	cmd.Flags().StringP("profile", "p", "", "Profile to use for this upload")
	cmd.Flags().StringP("channel", "c", "", "Override the destination channel for this upload")
	cmd.Flags().String("user", "", "Send a direct message to a user by ID")
	cmd.Flags().StringP("file", "f", "", "Path to the file to upload, or \"-\" for stdin")
	_ = cmd.MarkFlagRequired("file")

	cmd.Flags().StringP("comment", "m", "", "A comment to post with the file")
	cmd.Flags().StringP("filename", "n", "", "Filename for the upload")
	cmd.Flags().String("filetype", "", "Filetype for syntax highlighting")

	return cmd
}
