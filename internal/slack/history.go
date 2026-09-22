package slack

import "context"

func (c *Client) Messages(ctx context.Context, channel, thread, start, end string) ([]Message, error) {
	out := []Message{}
	seen := map[string]bool{}
	cursor := ""
	method := "conversations.history"
	if thread != "" {
		method = "conversations.replies"
	}
	for {
		body := map[string]any{"channel": channel, "limit": 200, "cursor": cursor}
		if thread != "" {
			body["ts"] = thread
		} else {
			body["inclusive"] = false
			if start != "" {
				body["oldest"] = start
			}
			if end != "" {
				body["latest"] = end
			}
		}
		var page struct {
			Messages []Message `json:"messages"`
			More     bool      `json:"has_more"`
			Metadata metadata  `json:"response_metadata"`
		}
		err := c.api(ctx, method, body, &page)
		if err != nil {
			return nil, err
		}
		out = append(out, page.Messages...)
		cursor, err = nextCursor(page.Metadata, page.More, seen)
		if err != nil {
			return nil, err
		}
		if cursor == "" {
			return out, nil
		}
	}
}
func (c *Client) UserName(ctx context.Context, id string) string { return c.userName(ctx, id) }
