package slack

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var timestampPattern = regexp.MustCompile(`^[0-9]+\.[0-9]{1,6}$`)

func ValidateThread(s string) error {
	if s != "" && !timestampPattern.MatchString(s) {
		return errors.New("--thread must be a parent Slack timestamp (seconds.microseconds)")
	}
	return nil
}

type PostOptions struct {
	Channel, User, Username, IconEmoji, Thread string
	Payload                                    map[string]any
}
type PostResult struct {
	TS      string `json:"ts"`
	Channel string `json:"channel"`
}

// ParseContent preserves only the documented message fields, never routing/identity from payload JSON.
func ParseContent(content, format string) (map[string]any, error) {
	p := map[string]any{}
	switch format {
	case "text":
		if strings.TrimSpace(content) == "" {
			return nil, errors.New("message is empty")
		}
		p["text"] = content
	case "blocks":
		var v any
		if json.Unmarshal([]byte(content), &v) != nil {
			return nil, errors.New("invalid blocks JSON")
		}
		if obj, ok := v.(map[string]any); ok {
			v = obj["blocks"]
		}
		blocks, ok := v.([]any)
		if !ok || len(blocks) == 0 {
			return nil, errors.New("blocks must be a nonempty JSON array")
		}
		p["blocks"] = blocks
	case "payload":
		var v map[string]json.RawMessage
		if json.Unmarshal([]byte(content), &v) != nil || v == nil {
			return nil, errors.New("payload must be a JSON object")
		}
		for _, key := range []string{"text", "blocks", "attachments", "unfurl_links", "unfurl_media", "mrkdwn"} {
			raw, ok := v[key]
			if !ok {
				continue
			}
			switch key {
			case "text":
				var s string
				if json.Unmarshal(raw, &s) != nil || string(raw) == "null" {
					return nil, errors.New("payload text must be a string")
				}
				p[key] = s
			case "blocks", "attachments":
				var a []any
				if json.Unmarshal(raw, &a) != nil || a == nil {
					return nil, fmt.Errorf("payload %s must be an array", key)
				}
				if len(a) > 0 {
					p[key] = a
				}
			default:
				var b bool
				if json.Unmarshal(raw, &b) != nil || string(raw) == "null" {
					return nil, fmt.Errorf("payload %s must be boolean", key)
				}
				p[key] = b
			}
		}
		if strings.TrimSpace(fmt.Sprint(p["text"])) == "" {
			delete(p, "text")
		}
		_, hasText := p["text"]
		_, hasBlocks := p["blocks"]
		_, hasAttachments := p["attachments"]
		if !hasText && !hasBlocks && !hasAttachments {
			return nil, errors.New("payload requires text, blocks or attachments")
		}
	default:
		return nil, errors.New("--format must be text, blocks or payload")
	}
	return p, nil
}
func (c *Client) Post(ctx context.Context, o PostOptions) (PostResult, error) {
	var result PostResult
	if err := ValidateThread(o.Thread); err != nil {
		return result, err
	}
	id, err := c.Destination(ctx, o.Channel, o.User)
	if err != nil {
		return result, err
	}
	body := map[string]any{}
	for k, v := range o.Payload {
		body[k] = v
	}
	body["channel"] = id
	if o.Thread != "" {
		body["thread_ts"] = o.Thread
	}
	if o.Username != "" {
		body["username"] = o.Username
	}
	if o.IconEmoji != "" {
		body["icon_emoji"] = o.IconEmoji
	}
	err = c.api(ctx, "chat.postMessage", body, &result)
	if IsCode(err, "not_in_channel") && strings.HasPrefix(id, "C") {
		info, e := c.ChannelInfo(ctx, id)
		if e != nil {
			return result, e
		}
		if info.PrivateKnown && !info.IsPrivate {
			if e = c.Join(ctx, id); e != nil {
				return result, e
			}
			err = c.api(ctx, "chat.postMessage", body, &result)
		}
	}
	if err == nil && (result.TS == "" || result.Channel == "") {
		err = errors.New("chat.postMessage: missing result; outcome unknown")
	}
	return result, err
}
func (c *Client) Invite(ctx context.Context, channel string, names []string) (string, []string, error) {
	id, err := c.ResolveChannel(ctx, channel)
	if err != nil {
		return "", nil, err
	}
	users, err := c.Invitees(ctx, names)
	if err != nil {
		return id, nil, err
	}
	err = c.api(ctx, "conversations.invite", map[string]string{"channel": id, "users": strings.Join(users, ",")}, nil)
	return id, users, err
}

type CreateOptions struct {
	Name, Topic, Description string
	Private                  bool
	Invite                   []string
}

func (c *Client) Create(ctx context.Context, o CreateOptions) (Channel, error) {
	var r struct {
		Channel Channel `json:"channel"`
	}
	var users []string
	if len(o.Invite) > 0 {
		var err error
		users, err = c.Invitees(ctx, o.Invite)
		if err != nil {
			return r.Channel, err
		}
	}
	if err := c.api(ctx, "conversations.create", map[string]any{"name": o.Name, "is_private": o.Private}, &r); err != nil {
		return r.Channel, err
	}
	if r.Channel.ID == "" {
		return r.Channel, errors.New("conversations.create: missing channel ID; outcome unknown")
	}
	steps := []struct{ method, key, value string }{{"conversations.setTopic", "topic", o.Topic}, {"conversations.setPurpose", "purpose", o.Description}, {"conversations.invite", "users", strings.Join(users, ",")}}
	for _, s := range steps {
		if s.value != "" {
			if err := c.api(ctx, s.method, map[string]string{"channel": r.Channel.ID, s.key: s.value}, nil); err != nil {
				return r.Channel, fmt.Errorf("created channel %s; subsequent step failed: %w", r.Channel.ID, err)
			}
		}
	}
	return r.Channel, nil
}
