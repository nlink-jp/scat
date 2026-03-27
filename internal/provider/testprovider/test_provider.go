package testprovider

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/nlink-jp/scat/internal/appcontext"
	"github.com/nlink-jp/scat/internal/config"
	"github.com/nlink-jp/scat/internal/export"
	"github.com/nlink-jp/scat/internal/provider"
)

var PostMessageSignal chan struct{}

// Provider implements the provider.Interface for testing purposes.
type Provider struct {
	Profile config.Profile
	Context appcontext.Context
}

// NewProvider creates a new test Provider.
func NewProvider(p config.Profile, ctx appcontext.Context) (provider.Interface, error) {
	fmt.Fprintf(os.Stderr, "[TESTPROVIDER] NewProvider called with profile: %s, context: %+v\n", p.Provider, ctx)
	return &Provider{Profile: p, Context: ctx},
		nil
}

// Capabilities returns the features supported by the test provider.
func (p *Provider) Capabilities() provider.Capabilities {
	fmt.Fprintf(os.Stderr, "[TESTPROVIDER] Capabilities called\n")
	return provider.Capabilities{
		CanListChannels:    true,
		CanListUsers:       true,
		CanPostFile:        true,
		CanUseIconEmoji:    true,
		CanExportLogs:      true,
		CanPostBlocks:      true,
		CanCreateChannel:   true,
		CanInviteToChannel: true,
	}
}

// PostMessage logs the message options to stderr.
func (p *Provider) PostMessage(opts provider.PostMessageOptions) error {
	if opts.Text == `{"test_command": "signal_done"}` {
		if PostMessageSignal != nil {
			PostMessageSignal <- struct{}{}
		}
		return nil
	}

	// Note: No complex logic for channel/user resolution in test provider.
	// We just log the raw options to verify that the command layer is sending them correctly.
	fmt.Fprintf(os.Stderr, "[TESTPROVIDER] PostMessage called with opts: {TargetChannel:%s TargetUserID:%s Text:%s OverrideUsername:%s IconEmoji:%s Blocks:%s}\n", opts.TargetChannel, opts.TargetUserID, opts.Text, opts.OverrideUsername, opts.IconEmoji, string(opts.Blocks))
	return nil
}

// PostFile logs the file options to stderr.
func (p *Provider) PostFile(opts provider.PostFileOptions) error {
	// Create a temporary struct for logging that includes all relevant fields.
	logOpts := struct {
		TargetChannel    string
		TargetUserID     string
		FilePath         string
		Filename         string
		Filetype         string
		Comment          string
		OverrideUsername string
		IconEmoji        string
	}{
		TargetChannel:    opts.TargetChannel,
		TargetUserID:     opts.TargetUserID,
		FilePath:         opts.FilePath,
		Filename:         opts.Filename,
		Filetype:         opts.Filetype,
		Comment:          opts.Comment,
		OverrideUsername: opts.OverrideUsername,
		IconEmoji:        opts.IconEmoji,
	}

	fmt.Fprintf(os.Stderr, "[TESTPROVIDER] PostFile called with opts: %+v\n", logOpts)
	return nil
}

// ListChannels logs the call and returns dummy data.
func (p *Provider) ListChannels() ([]provider.Channel, error) {
	fmt.Fprintf(os.Stderr, "[TESTPROVIDER] ListChannels called\n")
	return []provider.Channel{
		{ID: "C0000000001", Name: "test-channel-1"},
		{ID: "C0000000002", Name: "test-channel-2"},
	}, nil
}

// ListUsers logs the call and returns dummy data.
func (p *Provider) ListUsers() ([]provider.UserInfo, error) {
	fmt.Fprintf(os.Stderr, "[TESTPROVIDER] ListUsers called\n")
	return []provider.UserInfo{
		{ID: "U0000000001", Name: "test-user-1"},
		{ID: "U0000000002", Name: "test-user-2"},
	}, nil
}

// InviteToChannel logs the call.
func (p *Provider) InviteToChannel(opts provider.InviteToChannelOptions) error {
	fmt.Fprintf(os.Stderr, "[TESTPROVIDER] InviteToChannel called with opts: %+v\n", opts)
	return nil
}

// ExportLog logs the export options and returns dummy data that reflects the options.
func (p *Provider) ExportLog(opts export.Options) (*export.ExportedLog, error) {
	fmt.Fprintf(os.Stderr, "[TESTPROVIDER] ExportLog called with opts: %+v\n", opts)

	// Create a dummy message
	message := export.ExportedMessage{
		Text:      "Test message from ExportLog",
		UserName:  "testuser",
		Timestamp: "1672531200.000000", // 2023-01-01 00:00:00 UTC
	}

	// If file export is requested, add dummy file info
	if opts.IncludeFiles {
		message.Files = []export.ExportedFile{
			{
				ID:        "F12345678",
				Name:      "test-file.txt",
				Mimetype:  "text/plain",
				LocalPath: filepath.Join(opts.OutputDir, "test-file.txt"),
			},
		}
	}

	return &export.ExportedLog{
		ChannelName:     opts.ChannelName,
		ExportTimestamp: time.Now().UTC().Format(time.RFC3339),
		Messages:        []export.ExportedMessage{message},
	}, nil
}

// CreateChannel logs the call and returns a dummy channel ID.
func (p *Provider) CreateChannel(opts provider.CreateChannelOptions) (string, error) {
	fmt.Fprintf(os.Stderr, "[TESTPROVIDER] CreateChannel called with opts: %+v\n", opts)
	return "C1234567890", nil
}
