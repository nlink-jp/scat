package slack

import "encoding/json"

type Channel struct {
	MemberKnown  bool   `json:"-"`
	PrivateKnown bool   `json:"-"`
	ID           string `json:"id"`
	Name         string `json:"name"`
	IsPrivate    bool   `json:"is_private"`
	IsMember     bool   `json:"is_member"`
	IsIM         bool   `json:"is_im"`
	IsMPIM       bool   `json:"is_mpim"`
}
type User struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	RealName string `json:"real_name"`
	Deleted  bool   `json:"deleted"`
	IsBot    bool   `json:"is_bot"`
	Profile  struct {
		DisplayName string `json:"display_name"`
		RealName    string `json:"real_name"`
	} `json:"profile"`
}

func (u User) DisplayName() string {
	if u.Profile.DisplayName != "" {
		return u.Profile.DisplayName
	}
	if u.Profile.RealName != "" {
		return u.Profile.RealName
	}
	if u.RealName != "" {
		return u.RealName
	}
	return u.Name
}

type File struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Mimetype   string `json:"mimetype"`
	Size       int64  `json:"size"`
	URL        string `json:"url_private_download"`
	PrivateURL string `json:"url_private"`
}
type Message struct {
	TS          string          `json:"ts"`
	User        string          `json:"user"`
	BotID       string          `json:"bot_id"`
	Username    string          `json:"username"`
	Text        string          `json:"text"`
	ThreadTS    string          `json:"thread_ts"`
	ReplyCount  int             `json:"reply_count"`
	Files       []File          `json:"files"`
	Attachments json.RawMessage `json:"attachments"`
	Blocks      json.RawMessage `json:"blocks"`
}

func (c *Channel) UnmarshalJSON(b []byte) error {
	type plain Channel
	var v plain
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	var presence struct {
		Member  *bool `json:"is_member"`
		Private *bool `json:"is_private"`
	}
	if err := json.Unmarshal(b, &presence); err != nil {
		return err
	}
	*c = Channel(v)
	c.MemberKnown = presence.Member != nil
	c.PrivateKnown = presence.Private != nil
	return nil
}
