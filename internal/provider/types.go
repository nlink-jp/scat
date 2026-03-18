package provider

// This file defines common, provider-agnostic data structures for API responses
// and method options.

// --- Options Structs ---

// PostMessageOptions defines the parameters for a PostMessage call.
type PostMessageOptions struct {
	// TargetChannel specifies the destination channel name or ID.
	// If provided, this overrides the default channel configured in the provider's profile.
	// If left empty, the provider's default channel is used as a fallback.
	// This field is mutually exclusive with TargetUserID.
	TargetChannel string

	// TargetUserID specifies the destination user ID for a direct message.
	// If provided, the provider should open a DM channel with this user.
	// This field is mutually exclusive with TargetChannel.
	TargetUserID string

	Text             string
	OverrideUsername string
	IconEmoji        string
	Blocks           []byte
}

// PostFileOptions defines the parameters for a PostFile call.
type PostFileOptions struct {
	// TargetChannel specifies the destination channel name or ID.
	// If provided, this overrides the default channel configured in the provider's profile.
	// If left empty, the provider's default channel is used as a fallback.
	// This field is mutually exclusive with TargetUserID.
	TargetChannel string

	// TargetUserID specifies the destination user ID for a direct message.
	// If provided, the provider should open a DM channel with this user.
	// This field is mutually exclusive with TargetChannel.
	TargetUserID string

	FilePath         string
	Filename         string
	Filetype         string
	Comment          string
	OverrideUsername string
	IconEmoji        string
}

// GetConversationHistoryOptions defines the parameters for a GetConversationHistory call.
type GetConversationHistoryOptions struct {
	ChannelName string // Required
	Latest      string // Optional
	Oldest      string // Optional
	Limit       int    // Optional
	Cursor      string // Optional
}

// CreateChannelOptions defines the parameters for a CreateChannel call.
type CreateChannelOptions struct {
	Name        string
	Description string
	Topic       string
	IsPrivate   bool
	Invitees    []string
}

// InviteToChannelOptions defines the parameters for an InviteToChannel call.
type InviteToChannelOptions struct {
	Channel  string   // Channel name or ID
	Invitees []string // User names, user IDs, or user group handles
}

// Channel represents a channel with its name and ID.
type Channel struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// UserInfo represents a user with their name and ID.
type UserInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// --- Response Structs ---

// ConversationHistoryResponse represents the response from a conversation history API call.
type ConversationHistoryResponse struct {
	Messages         []Message
	HasMore          bool
	NextCursor       string
	ResponseMetadata ResponseMetadata
}

// UserInfoResponse represents the response from a user info API call.
type UserInfoResponse struct {
	User User
}

// Message represents a single message in a channel.
type Message struct {
	Type      string `json:"type"`
	Timestamp string `json:"ts"`
	UserID    string `json:"user"`
	Text      string `json:"text"`
	Files     []File `json:"files,omitempty"`
}

// File represents a file attached to a message.
type File struct {
	ID                 string `json:"id"`
	Created            int    `json:"created"`
	Timestamp          int    `json:"timestamp"`
	Name               string `json:"name"`
	Title              string `json:"title"`
	Mimetype           string `json:"mimetype"`
	Filetype           string `json:"filetype"`
	PrettyType         string `json:"pretty_type"`
	User               string `json:"user"`
	Editable           bool   `json:"editable"`
	Size               int    `json:"size"`
	Mode               string `json:"mode"`
	IsExternal         bool   `json:"is_external"`
	ExternalType       string `json:"external_type"`
	IsPublic           bool   `json:"is_public"`
	PublicURLShared    bool   `json:"public_url_shared"`
	DisplayAsBot       bool   `json:"display_as_bot"`
	Username           string `json:"username"`
	URLPrivate         string `json:"url_private"`
	URLPrivateDownload string `json:"url_private_download"`
	Permalink          string `json:"permalink"`
	PermalinkPublic    string `json:"permalink_public"`
}

// User represents a user.
type User struct {
	ID       string `json:"id"`
	TeamID   string `json:"team_id"`
	Name     string `json:"name"`
	RealName string `json:"real_name"`
}

// ResponseMetadata contains metadata about the response, like cursors for pagination.
type ResponseMetadata struct {
	NextCursor string `json:"next_cursor"`
}
