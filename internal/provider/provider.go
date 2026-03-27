package provider

import "github.com/nlink-jp/scat/internal/export"

// Capabilities defines what features a provider supports.
type Capabilities struct {
	CanListChannels  bool // Whether the provider can list channels.
	CanListUsers     bool // Whether the provider can list users.
	CanPostFile      bool // Whether the provider can post files.
	CanUseIconEmoji  bool // Whether the provider supports custom icon emojis.
	CanExportLogs    bool // Whether the provider can export channel logs.
	CanPostBlocks    bool // Whether the provider can post Block Kit messages.
	CanCreateChannel bool // Whether the provider can create channels.
	CanInviteToChannel bool // Whether the provider can invite users to a channel.
}

// Interface defines the methods that a provider must implement.
type Interface interface {
	// Capabilities returns a struct indicating supported features.
	Capabilities() Capabilities

	// PostMessage sends a text-based message.
	PostMessage(opts PostMessageOptions) error

	// PostFile sends a file.
	PostFile(opts PostFileOptions) error

	// ListChannels lists available channels with their IDs.
	// This should only be called if Capabilities().CanListChannels is true.
	ListChannels() ([]Channel, error)

	// ListUsers lists available users with their IDs.
	// This should only be called if Capabilities().CanListUsers is true.
	ListUsers() ([]UserInfo, error)

	// ExportLog performs the entire export operation.
	// This should only be called if Capabilities().CanExportLogs is true.
	ExportLog(opts export.Options) (*export.ExportedLog, error)

	// CreateChannel creates a new channel.
	// This should only be called if Capabilities().CanCreateChannel is true.
	CreateChannel(opts CreateChannelOptions) (string, error)

	// InviteToChannel invites users or user groups to an existing channel.
	// This should only be called if Capabilities().CanInviteToChannel is true.
	InviteToChannel(opts InviteToChannelOptions) error
}
