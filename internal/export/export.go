package export

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/nlink-jp/scat/internal/slack"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Source is owned by the export consumer; tests do not need a runtime provider.
type Source interface {
	ResolveChannel(context.Context, string) (string, error)
	ChannelInfo(context.Context, string) (slack.Channel, error)
	Messages(context.Context, string, string, string, string) ([]slack.Message, error)
	UserName(context.Context, string) string
	Download(context.Context, slack.File, string) (string, error)
}
type Request struct{ Channel, Start, End, SaveDir string }
type Exporter struct {
	Source Source
	Now    func() time.Time
	Warn   func(string)
}

func ParseTimestamp(s string) (time.Time, error) {
	parts := strings.Split(s, ".")
	if len(parts) != 2 || len(parts[1]) == 0 || len(parts[1]) > 6 {
		return time.Time{}, errors.New("invalid Slack timestamp")
	}
	for _, part := range parts {
		if part == "" {
			return time.Time{}, errors.New("invalid Slack timestamp")
		}
		for _, c := range part {
			if c < '0' || c > '9' {
				return time.Time{}, errors.New("invalid Slack timestamp")
			}
		}
	}
	sec, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return time.Time{}, err
	}
	ns, err := strconv.ParseInt(parts[1]+strings.Repeat("0", 9-len(parts[1])), 10, 64)
	if err != nil {
		return time.Time{}, err
	}
	return time.Unix(sec, ns).UTC(), nil
}
func Bounds(start, end string) (string, string, error) {
	var a, b time.Time
	var err error
	if start != "" {
		a, err = time.Parse(time.RFC3339Nano, start)
		if err != nil {
			return "", "", errors.New("--start must be RFC3339")
		}
	}
	if end != "" {
		b, err = time.Parse(time.RFC3339Nano, end)
		if err != nil {
			return "", "", errors.New("--end must be RFC3339")
		}
	}
	if start != "" && end != "" && !a.Before(b) {
		return "", "", errors.New("--start must precede --end")
	}
	ts := func(t time.Time, present, upper bool) string {
		if !present {
			return ""
		}
		if upper && t.Nanosecond()%1000 != 0 {
			t = t.Add(time.Duration(1000 - t.Nanosecond()%1000))
		}
		return fmt.Sprintf("%d.%06d", t.Unix(), t.Nanosecond()/1000)
	}
	return ts(a, start != "", false), ts(b, end != "", true), nil
}
func (e Exporter) Run(ctx context.Context, r Request) (*ExportedLog, error) {
	now := e.Now
	if now == nil {
		now = time.Now
	}
	warn := e.Warn
	if warn == nil {
		warn = func(string) {}
	}
	start, end, err := Bounds(r.Start, r.End)
	if err != nil {
		return nil, err
	}
	id, err := e.Source.ResolveChannel(ctx, r.Channel)
	if err != nil {
		return nil, err
	}
	name := "#" + id
	if ch, er := e.Source.ChannelInfo(ctx, id); er != nil {
		warn(er.Error())
	} else if ch.Name != "" {
		name = "#" + ch.Name
	}
	history, err := e.Source.Messages(ctx, id, "", start, end)
	if err != nil {
		return nil, err
	}
	sortMessages := func(list []slack.Message) error {
		for _, m := range list {
			if _, err := ParseTimestamp(m.TS); err != nil {
				return err
			}
		}
		sort.SliceStable(list, func(i, j int) bool {
			a, _ := ParseTimestamp(list[i].TS)
			b, _ := ParseTimestamp(list[j].TS)
			return a.Before(b)
		})
		return nil
	}
	if err = sortMessages(history); err != nil {
		return nil, err
	}
	threads := map[string][]slack.Message{}
	for _, m := range history {
		if m.ReplyCount > 0 && (m.ThreadTS == "" || m.ThreadTS == m.TS) {
			replies, err := e.Source.Messages(ctx, id, m.TS, "", "")
			if err != nil {
				return nil, err
			}
			if err = sortMessages(replies); err != nil {
				return nil, err
			}
			threads[m.TS] = replies
		}
	}
	out := &ExportedLog{ExportTimestamp: now().UTC().Format(time.RFC3339), ChannelName: name, Messages: []ExportedMessage{}}
	seen := map[string]bool{}
	downloads := map[string]string{}
	appendMessage := func(m slack.Message) error {
		if seen[m.TS] {
			return nil
		}
		seen[m.TS] = true
		t, err := ParseTimestamp(m.TS)
		if err != nil {
			return err
		}
		user := m.User
		if user == "" {
			user = m.BotID
		}
		postType := "user"
		if m.BotID != "" {
			postType = "bot"
		}
		username := m.Username
		if m.BotID == "" && m.User != "" {
			if resolved := e.Source.UserName(ctx, m.User); resolved != "" {
				username = resolved
			}
		}
		v := ExportedMessage{UserID: user, UserName: username, PostType: postType, Timestamp: t.Format(time.RFC3339), TimestampUnix: m.TS, Text: m.Text, Files: []ExportedFile{}, ThreadTimestampUnix: m.ThreadTS, IsReply: m.ThreadTS != "" && m.ThreadTS != m.TS}
		if len(m.Attachments) > 0 && string(m.Attachments) != "null" {
			if err = json.Unmarshal(m.Attachments, &v.Attachments); err != nil {
				return errors.New("invalid attachments in API message")
			}
		}
		if len(m.Blocks) > 0 && string(m.Blocks) != "null" {
			var blocks []json.RawMessage
			if json.Unmarshal(m.Blocks, &blocks) != nil {
				return errors.New("invalid blocks in API message")
			}
			v.Blocks = m.Blocks
		}
		for _, f := range m.Files {
			ef := ExportedFile{ID: f.ID, Name: f.Name, Mimetype: f.Mimetype}
			if r.SaveDir != "" {
				path, ok := downloads[f.ID]
				if !ok {
					path, err = e.Source.Download(ctx, f, r.SaveDir)
					if err != nil {
						warn(fmt.Sprintf("file %s: %v", f.ID, err))
						path = ""
					}
					downloads[f.ID] = path
				}
				ef.LocalPath = path
			}
			v.Files = append(v.Files, ef)
		}
		out.Messages = append(out.Messages, v)
		return nil
	}
	for _, m := range history {
		if m.ThreadTS != "" && m.ThreadTS != m.TS {
			if _, ok := threads[m.ThreadTS]; ok {
				continue
			}
		}
		if err = appendMessage(m); err != nil {
			return nil, err
		}
		for _, reply := range threads[m.TS] {
			if err = appendMessage(reply); err != nil {
				return nil, err
			}
		}
	}
	if err = ctx.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
func Render(w io.Writer, log *ExportedLog, format string) error {
	switch format {
	case "json":
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(log)
	case "text":
		if _, err := fmt.Fprintf(w, "%s\n", log.ChannelName); err != nil {
			return err
		}
		for _, m := range log.Messages {
			who := m.UserName
			if who == "" {
				who = m.UserID
			}
			if _, err := fmt.Fprintf(w, "[%s] %s: %s\n", m.Timestamp, who, m.Text); err != nil {
				return err
			}
			for _, f := range m.Files {
				if _, err := fmt.Fprintf(w, "  file: %s (%s) %s\n", f.Name, f.ID, f.LocalPath); err != nil {
					return err
				}
			}
		}
		return nil
	default:
		return errors.New("export --format must be json or text")
	}
}
func Write(ctx context.Context, w io.Writer, path, format string, log *ExportedLog) error {
	if path == "" || path == "-" {
		if err := ctx.Err(); err != nil {
			return err
		}
		return Render(w, log, format)
	}
	dir := filepath.Dir(path)
	f, err := os.CreateTemp(dir, ".scat-export-")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	defer f.Close()
	if err = Render(f, log, format); err != nil {
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}
