package slack

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

var idPattern = regexp.MustCompile(`^[A-Z][A-Z0-9]+$`)

func isID(s, kinds string) bool {
	return len(s) > 1 && strings.ContainsRune(kinds, rune(s[0])) && idPattern.MatchString(s)
}
func (c *Client) Channels(ctx context.Context) ([]Channel, error) {
	out := []Channel{}
	seen := map[string]bool{}
	cursor := ""
	for {
		var page struct {
			Channels []Channel `json:"channels"`
			Metadata metadata  `json:"response_metadata"`
			More     bool      `json:"has_more"`
		}
		err := c.api(ctx, "conversations.list", map[string]any{"types": "public_channel,private_channel", "exclude_archived": true, "limit": 200, "cursor": cursor}, &page)
		if err != nil {
			return nil, err
		}
		out = append(out, page.Channels...)
		cursor, err = nextCursor(page.Metadata, page.More, seen)
		if err != nil {
			return nil, err
		}
		if cursor == "" {
			break
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}
func (c *Client) Users(ctx context.Context) ([]User, error) {
	out := []User{}
	seen := map[string]bool{}
	cursor := ""
	for {
		var page struct {
			Members  []User   `json:"members"`
			Metadata metadata `json:"response_metadata"`
			More     bool     `json:"has_more"`
		}
		err := c.api(ctx, "users.list", map[string]any{"limit": 200, "cursor": cursor}, &page)
		if err != nil {
			return nil, err
		}
		out = append(out, page.Members...)
		cursor, err = nextCursor(page.Metadata, page.More, seen)
		if err != nil {
			return nil, err
		}
		if cursor == "" {
			break
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}
func unique(ids []string, kind string) (string, error) {
	if len(ids) == 0 {
		return "", fmt.Errorf("%s not found", kind)
	}
	if len(ids) > 1 {
		return "", fmt.Errorf("ambiguous %s; use an ID", kind)
	}
	return ids[0], nil
}
func (c *Client) ResolveChannel(ctx context.Context, s string) (string, error) {
	s = strings.TrimPrefix(s, "#")
	if s == "" {
		return "", errors.New("channel is required")
	}
	if isID(s, "CGD") {
		return s, nil
	}
	list, err := c.Channels(ctx)
	if err != nil {
		return "", err
	}
	var ids []string
	for _, v := range list {
		if v.Name == s {
			ids = append(ids, v.ID)
		}
	}
	return unique(ids, "channel")
}
func (c *Client) ResolveUser(ctx context.Context, s string) (string, error) {
	s = strings.TrimPrefix(s, "@")
	if isID(s, "UW") {
		return s, nil
	}
	list, err := c.Users(ctx)
	if err != nil {
		return "", err
	}
	var ids []string
	for _, v := range list {
		if !v.Deleted && (v.Name == s || v.Profile.DisplayName == s) {
			ids = append(ids, v.ID)
		}
	}
	return unique(ids, "user")
}
func (c *Client) ChannelInfo(ctx context.Context, id string) (Channel, error) {
	var v struct {
		Channel Channel `json:"channel"`
	}
	err := c.api(ctx, "conversations.info", map[string]string{"channel": id}, &v)
	if err == nil && v.Channel.ID == "" {
		err = errors.New("conversations.info: missing channel ID")
	}
	return v.Channel, err
}
func (c *Client) Destination(ctx context.Context, channel, user string) (string, error) {
	if user == "" {
		return c.ResolveChannel(ctx, channel)
	}
	if channel != "" {
		return "", errors.New("--user and --channel are mutually exclusive")
	}
	id, err := c.ResolveUser(ctx, user)
	if err != nil {
		return "", err
	}
	var v struct {
		Channel Channel `json:"channel"`
	}
	err = c.api(ctx, "conversations.open", map[string]string{"users": id}, &v)
	if err == nil && v.Channel.ID == "" {
		err = errors.New("conversations.open: missing channel ID")
	}
	return v.Channel.ID, err
}
func (c *Client) userName(ctx context.Context, id string) string {
	if v, ok := c.names[id]; ok {
		return v
	}
	var r struct {
		User User `json:"user"`
	}
	name := id
	if err := c.api(ctx, "users.info", map[string]string{"user": id}, &r); err != nil {
		c.warn(err.Error())
	} else {
		name = r.User.DisplayName()
	}
	c.names[id] = name
	return name
}
func (c *Client) Join(ctx context.Context, id string) error {
	return c.api(ctx, "conversations.join", map[string]string{"channel": id}, nil)
}
func (c *Client) UploadDestination(ctx context.Context, channel, user string) (string, error) {
	id, err := c.Destination(ctx, channel, user)
	if err != nil {
		return "", err
	}
	if user != "" {
		return id, nil
	}
	info, err := c.ChannelInfo(ctx, id)
	if err != nil {
		return "", err
	}
	if info.IsIM || info.IsMPIM {
		return id, nil
	}
	if !info.MemberKnown {
		return "", errors.New("upload channel membership is unknown")
	}
	if info.IsMember {
		return id, nil
	}
	if !info.PrivateKnown {
		return "", errors.New("upload channel privacy is unknown")
	}
	if info.IsPrivate {
		return "", errors.New("upload requires private-channel membership")
	}
	if err = c.Join(ctx, id); err != nil {
		return "", err
	}
	return id, nil
}
func (c *Client) Invitees(ctx context.Context, names []string) ([]string, error) {
	ids := []string{}
	seen := map[string]bool{}
	for _, name := range names {
		clean := strings.TrimPrefix(name, "@")
		var members []string
		if isID(clean, "S") {
			var r struct {
				Users []string `json:"users"`
			}
			if err := c.api(ctx, "usergroups.users.list", map[string]string{"usergroup": clean}, &r); err != nil {
				return nil, err
			}
			members = r.Users
		} else if isID(clean, "UW") {
			members = []string{clean}
		} else {
			users, err := c.Users(ctx)
			if err != nil {
				return nil, err
			}
			var matches []string
			for _, u := range users {
				if !u.Deleted && (u.Name == clean || u.Profile.DisplayName == clean) {
					matches = append(matches, u.ID)
				}
			}
			if len(matches) > 1 {
				return nil, errors.New("ambiguous invitee; use an ID")
			}
			if len(matches) == 1 {
				members = matches
			} else {
				var groups struct {
					Groups []struct {
						ID     string `json:"id"`
						Handle string `json:"handle"`
					} `json:"usergroups"`
				}
				if err = c.api(ctx, "usergroups.list", map[string]bool{"include_users": false}, &groups); err != nil {
					return nil, err
				}
				var gids []string
				for _, g := range groups.Groups {
					if g.Handle == clean {
						gids = append(gids, g.ID)
					}
				}
				gid, err := unique(gids, "user group")
				if err != nil {
					return nil, err
				}
				var r struct {
					Users []string `json:"users"`
				}
				if err = c.api(ctx, "usergroups.users.list", map[string]string{"usergroup": gid}, &r); err != nil {
					return nil, err
				}
				members = r.Users
			}
		}
		for _, id := range members {
			if !seen[id] {
				seen[id] = true
				ids = append(ids, id)
			}
		}
	}
	if len(ids) == 0 {
		return nil, errors.New("no users to invite")
	}
	return ids, nil
}
