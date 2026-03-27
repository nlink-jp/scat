package mock

import (
	"fmt"
	"os"
	"time"

	"github.com/nlink-jp/scat/internal/appcontext"
	"github.com/nlink-jp/scat/internal/config"
	"github.com/nlink-jp/scat/internal/export"
	"github.com/nlink-jp/scat/internal/provider"
)

// Provider implements the provider.Interface for mocking.
type Provider struct {
	Profile config.Profile
	Context appcontext.Context // Use appcontext.Context
}

// NewProvider creates a new mock Provider.
func NewProvider(p config.Profile, ctx appcontext.Context) (provider.Interface, error) {
	return &Provider{Profile: p, Context: ctx}, nil
}

// Capabilities returns the features supported by the mock provider.
func (p *Provider) Capabilities() provider.Capabilities {
	return provider.Capabilities{
		CanListChannels:    false,
		CanListUsers:       false,
		CanPostFile:        true,
		CanUseIconEmoji:    false,
		CanExportLogs:      true,
		CanPostBlocks:      true,
		CanCreateChannel:   true,
		CanInviteToChannel: false,
	}
}

// PostMessage prints a mock message.
func (p *Provider) PostMessage(opts provider.PostMessageOptions) error {
	var destination string
	switch {
	case opts.TargetUserID != "":
		destination = fmt.Sprintf("User: %s", opts.TargetUserID)
	case opts.TargetChannel != "":
		destination = fmt.Sprintf("Channel: %s", opts.TargetChannel)
	default:
		if p.Profile.Channel == "" {
			return fmt.Errorf("no channel or user specified; please set a default channel in the profile or use the --channel or --user flag")
		}
		destination = fmt.Sprintf("Channel: %s (default)", p.Profile.Channel)
	}

	if !p.Context.Silent {
		fmt.Fprintln(os.Stderr, "--- [MOCK] PostMessage called ---")
		fmt.Fprintln(os.Stderr, destination)
		if len(opts.Blocks) > 0 {
			fmt.Fprintf(os.Stderr, "Blocks: %s\n", string(opts.Blocks))
		} else {
			fmt.Fprintf(os.Stderr, "Text: %s\n", opts.Text)
		}
	}
	if p.Context.Debug {
		fmt.Fprintf(os.Stderr, "[DEBUG] Mock PostMessage: Destination=\"%s\", Text=\"%s\", Username=\"%s\", IconEmoji=\"%s\", Blocks=\"%s\"\n", destination, opts.Text, opts.OverrideUsername, opts.IconEmoji, string(opts.Blocks))
	}
	return nil
}

// PostFile prints a mock message.
func (p *Provider) PostFile(opts provider.PostFileOptions) error {
	var destination string
	switch {
	case opts.TargetUserID != "":
		destination = fmt.Sprintf("User: %s", opts.TargetUserID)
	case opts.TargetChannel != "":
		destination = fmt.Sprintf("Channel: %s", opts.TargetChannel)
	default:
		if p.Profile.Channel == "" {
			return fmt.Errorf("no channel or user specified; please set a default channel in the profile or use the --channel or --user flag")
		}
		destination = fmt.Sprintf("Channel: %s (default)", p.Profile.Channel)
	}

	if !p.Context.Silent {
		fmt.Fprintln(os.Stderr, "--- [MOCK] PostFile called ---")
		fmt.Fprintln(os.Stderr, destination)
		fmt.Fprintf(os.Stderr, "File: %s\n", opts.FilePath)
	}
	if p.Context.Debug {
		fmt.Fprintf(os.Stderr, "[DEBUG] Mock PostFile: Destination=\"%s\", FilePath=\"%s\", Filename=\"%s\"\n", destination, opts.FilePath, opts.Filename)
	}
	return nil
}

// ListChannels returns an error as it's not supported.
func (p *Provider) ListChannels() ([]provider.Channel, error) {
	return nil, fmt.Errorf("ListChannels is not supported by the mock provider")
}

// ListUsers returns an error as it's not supported.
func (p *Provider) ListUsers() ([]provider.UserInfo, error) {
	return nil, fmt.Errorf("ListUsers is not supported by the mock provider")
}

// InviteToChannel returns an error as it's not supported.
func (p *Provider) InviteToChannel(opts provider.InviteToChannelOptions) error {
	return fmt.Errorf("InviteToChannel is not supported by the mock provider")
}

// ExportLog returns a dummy log for testing.
func (p *Provider) ExportLog(opts export.Options) (*export.ExportedLog, error) {
	if !p.Context.Silent {
		fmt.Fprintf(os.Stderr, "--- [MOCK] ExportLog called for channel %s ---", opts.ChannelName)
	}
	return &export.ExportedLog{
		ExportTimestamp: time.Now().UTC().Format(time.RFC3339),
		ChannelName:     opts.ChannelName,
		Messages: []export.ExportedMessage{
			{
				UserID:        "U012AB3CDE",
				UserName:      "Mock User",
				Timestamp:     time.Now().UTC().Format(time.RFC3339),
				TimestampUnix: fmt.Sprintf("%d.000000", time.Now().Unix()),
				Text:          "Hello from mock exporter!",
			},
		},
	},
nil
}

// CreateChannel simulates creating a channel.
func (p *Provider) CreateChannel(opts provider.CreateChannelOptions) (string, error) {
	if opts.Name == "error" {
		return "", fmt.Errorf("mock error creating channel")
	}
	if !p.Context.Silent {
		fmt.Fprintf(os.Stderr, "--- [MOCK] CreateChannel called with options: %+v ---", opts)
	}
	return "C0MOCKCHANNEL", nil
}
