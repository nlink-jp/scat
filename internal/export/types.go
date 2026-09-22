package export

import "encoding/json"

// ExportedFile holds metadata for a file attached to an exported message.
// LocalPath is non-empty only when the file was downloaded via --save-dir.
type ExportedFile struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Mimetype  string `json:"mimetype"`
	LocalPath string `json:"local_path"`
}

// ExportAttachment holds a legacy rich attachment from a Slack message.
type ExportAttachment struct {
	Fallback  string                  `json:"fallback,omitempty"`
	Color     string                  `json:"color,omitempty"`
	Pretext   string                  `json:"pretext,omitempty"`
	Title     string                  `json:"title,omitempty"`
	TitleLink string                  `json:"title_link,omitempty"`
	Text      string                  `json:"text,omitempty"`
	Fields    []ExportAttachmentField `json:"fields,omitempty"`
	Footer    string                  `json:"footer,omitempty"`
	ImageURL  string                  `json:"image_url,omitempty"`
}

// ExportAttachmentField is a key-value pair inside a legacy attachment.
type ExportAttachmentField struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

// ExportedMessage is the per-message record written by channel export.
// Schema baseline: scli 854e6a0. See ADR-0001 and parity fixtures.
type ExportedMessage struct {
	UserID              string             `json:"user_id"`
	UserName            string             `json:"user_name,omitempty"`
	PostType            string             `json:"post_type"` // "user" or "bot"
	Timestamp           string             `json:"timestamp"` // RFC3339
	TimestampUnix       string             `json:"timestamp_unix"`
	Text                string             `json:"text"`
	Files               []ExportedFile     `json:"files"`
	Attachments         []ExportAttachment `json:"attachments,omitempty"`
	Blocks              json.RawMessage    `json:"blocks,omitempty"`
	ThreadTimestampUnix string             `json:"thread_timestamp_unix,omitempty"`
	IsReply             bool               `json:"is_reply"`
}

// ExportedLog is the top-level structure of an exported channel.
type ExportedLog struct {
	ExportTimestamp string            `json:"export_timestamp"`
	ChannelName     string            `json:"channel_name"`
	Messages        []ExportedMessage `json:"messages"`
}
