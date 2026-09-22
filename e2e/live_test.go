//go:build e2e

// Package e2e exercises the built CLI against a dedicated real Slack channel.
package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nlink-jp/scat/internal/config"
	"github.com/nlink-jp/scat/internal/export"
)

type live struct {
	t                              *testing.T
	binary, config, token, channel string
	root, tee, botID, userID       string
	env                            []string
	posts, files                   []string
}

func newLive(t *testing.T) *live {
	t.Helper()
	path, channel := os.Getenv("SCAT_E2E_CONFIG"), os.Getenv("SCAT_E2E_CHANNEL")
	if path == "" || channel == "" {
		t.Fatal("live tests require SCAT_E2E_CONFIG and SCAT_E2E_CHANNEL (dedicated channel ID); no tests skipped")
	}
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	token := cfg.Profiles[cfg.CurrentProfile].Token
	if !strings.HasPrefix(token, "xoxb-") {
		t.Fatal("live tests require a bot token")
	}
	binary, err := filepath.Abs("../dist/scat")
	if err != nil {
		t.Fatal(err)
	}
	l := &live{t: t, binary: binary, config: path, token: token, channel: channel}
	for _, e := range os.Environ() {
		if !strings.HasPrefix(e, "SCAT_") {
			l.env = append(l.env, e)
		}
	}
	t.Cleanup(l.cleanup)
	var auth struct {
		BotID  string `json:"bot_id"`
		UserID string `json:"user_id"`
	}
	l.mustAPI("auth.test", url.Values{}, &auth)
	l.botID, l.userID = auth.BotID, auth.UserID
	if auth.BotID == "" {
		t.Fatal("auth.test did not identify a bot")
	}
	var info struct {
		Channel struct {
			IsMember   bool `json:"is_member"`
			IsArchived bool `json:"is_archived"`
		} `json:"channel"`
	}
	l.mustAPI("conversations.info", url.Values{"channel": {channel}}, &info)
	if !info.Channel.IsMember || info.Channel.IsArchived {
		t.Fatal("use an active dedicated channel the bot already belongs to")
	}
	return l
}

// The verification/cleanup client is independent of the implementation under test.
// It never logs tokens, bodies, workspace IDs or signed URLs, and denies redirects.
func (l *live) api(method string, values url.Values, out any) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://slack.com/api/"+method, strings.NewReader(values.Encode()))
	if err != nil {
		return fmt.Errorf("%s: request construction failed", method)
	}
	req.Header.Set("Authorization", "Bearer "+l.token)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	res, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("%s: transport failed", method)
	}
	defer res.Body.Close()
	b, err := io.ReadAll(io.LimitReader(res.Body, 4<<20))
	if err != nil {
		return fmt.Errorf("%s: read failed", method)
	}
	var status struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
	}
	if json.Unmarshal(b, &status) != nil {
		return fmt.Errorf("%s: non-JSON response", method)
	}
	if res.StatusCode != 200 || !status.OK {
		return fmt.Errorf("%s: HTTP %d code %s", method, res.StatusCode, status.Error)
	}
	if out != nil {
		return json.Unmarshal(b, out)
	}
	return nil
}
func (l *live) mustAPI(method string, v url.Values, out any) {
	l.t.Helper()
	if err := l.api(method, v, out); err != nil {
		l.t.Fatal(err)
	}
}
func (l *live) run(stdin []byte, server bool, args ...string) ([]byte, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	env := append([]string{}, l.env...)
	if server {
		env = append(env, "SCAT_MODE=server", "SCAT_TOKEN="+l.token, "SCAT_CHANNEL="+l.channel)
	} else {
		args = append([]string{"--config", l.config}, args...)
	}
	cmd := exec.CommandContext(ctx, l.binary, args...)
	cmd.Env = env
	cmd.Stdin = bytes.NewReader(stdin)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.Bytes(), strings.ReplaceAll(stderr.String(), l.token, "[redacted]"), err
}
func (l *live) checkResult(command string, stderr string, err error) {
	l.t.Helper()
	if err != nil {
		l.t.Fatalf("CLI %s failed: %v; %s", command, err, stderr)
	}
	if strings.Contains(strings.ToLower(stderr), "warning:") {
		l.t.Fatalf("CLI warning: %s", stderr)
	}
}
func (l *live) must(stdin []byte, server bool, args ...string) []byte {
	l.t.Helper()
	out, stderr, err := l.run(stdin, server, args...)
	l.checkResult(args[0], stderr, err)
	return out
}
func (l *live) post(stdin []byte, args ...string) string {
	l.t.Helper()
	b, stderr, err := l.run(stdin, false, append([]string{"post", "-c", l.channel, "--json"}, args...)...)
	var r struct{ TS, Channel string }
	parseErr := json.Unmarshal(b, &r)
	if parseErr == nil && r.TS != "" && r.Channel == l.channel {
		l.posts = append(l.posts, r.TS)
	}
	l.checkResult("post", stderr, err)
	if parseErr != nil || r.TS == "" || r.Channel != l.channel {
		l.t.Fatal("invalid post result")
	}
	return r.TS
}

// Discover only the exact tee reply by the authenticated bot under this run's
// known parent. Never derive deletion targets from the export being tested.
func (l *live) discoverTee() error {
	if l.root == "" || l.tee == "" {
		return nil
	}
	var r struct {
		Messages []struct {
			TS, Text string
			Thread   string `json:"thread_ts"`
			BotID    string `json:"bot_id"`
			User     string `json:"user"`
		}
	}
	if err := l.api("conversations.replies", url.Values{"channel": {l.channel}, "ts": {l.root}, "limit": {"100"}}, &r); err != nil {
		return err
	}
	for _, m := range r.Messages {
		if m.Text == l.tee && m.Thread == l.root && m.BotID == l.botID && m.User == l.userID {
			l.posts = append(l.posts, m.TS)
			l.tee = ""
			return nil
		}
	}
	return fmt.Errorf("tee reply not found by independent API")
}
func (l *live) cleanup() {
	if err := l.discoverTee(); err != nil {
		l.t.Errorf("cleanup discovery: %v", err)
	}
	for i := len(l.files) - 1; i >= 0; i-- {
		if err := l.api("files.delete", url.Values{"file": {l.files[i]}}, nil); err != nil {
			l.t.Errorf("cleanup own file: %v", err)
		}
	}
	seen := map[string]bool{}
	for i := len(l.posts) - 1; i >= 0; i-- {
		ts := l.posts[i]
		if seen[ts] {
			continue
		}
		seen[ts] = true
		if err := l.api("chat.delete", url.Values{"channel": {l.channel}, "ts": {ts}}, nil); err != nil && !strings.Contains(err.Error(), "message_not_found") {
			l.t.Errorf("cleanup own message: %v", err)
		}
	}
}
func decodeLog(t *testing.T, b []byte) export.ExportedLog {
	t.Helper()
	var log export.ExportedLog
	if json.Unmarshal(b, &log) != nil || log.Messages == nil || log.ChannelName == "" {
		t.Fatal("invalid export envelope")
	}
	if _, err := time.Parse(time.RFC3339, log.ExportTimestamp); err != nil {
		t.Fatal("invalid export timestamp")
	}
	// Assert wire-level scli fields, independently of Go zero values.
	var wire struct{ Messages []map[string]json.RawMessage }
	if err := json.Unmarshal(b, &wire); err != nil {
		t.Fatal(err)
	}
	for _, m := range wire.Messages {
		if f, ok := m["files"]; !ok || string(f) == "null" {
			t.Fatal("files must always be an array")
		}
	}
	return log
}

func TestLiveRoundTrip(t *testing.T) {
	l := newLive(t)
	marker := fmt.Sprintf("scat-e2e-%d", time.Now().UnixNano())
	root := l.post(nil, marker+" root")
	l.root = root
	rootTime, err := export.ParseTimestamp(root)
	if err != nil {
		t.Fatal(err)
	}
	start := rootTime.Add(-time.Microsecond).Format(time.RFC3339Nano)
	end := rootTime.Add(time.Microsecond).Format(time.RFC3339Nano)
	payload := fmt.Sprintf(`{"text":%q,"blocks":[{"type":"section","text":{"type":"plain_text","text":%q}}],"attachments":[{"fallback":"e2e attachment","text":"attachment text","color":"#123456","fields":[{"title":"key","value":"value","short":true}]}]}`, marker+" reply", marker+" rich")
	reply := l.post([]byte(payload), "--format", "payload", "--thread", root)
	t.Log("PASS: real bot text post and rich thread reply")

	// Stream crosses the Unicode threshold and then flushes its remaining input at EOF.
	stream := marker + " " + strings.Repeat("界", 4001)
	out, streamErrText, streamErr := l.run([]byte(stream), false, "post", "--stream", "--thread", root, "-c", l.channel, "--json")
	dec := json.NewDecoder(bytes.NewReader(out))
	streamTS := []string{}
	for {
		var r struct{ TS string }
		err := dec.Decode(&r)
		if err == io.EOF {
			break
		}
		if err != nil || r.TS == "" {
			t.Fatal("invalid stream results")
		}
		streamTS = append(streamTS, r.TS)
		l.posts = append(l.posts, r.TS)
	}
	l.checkResult("stream", streamErrText, streamErr)
	if len(streamTS) != 2 {
		t.Fatalf("expected two stream batches, got %d", len(streamTS))
	}
	tee := []byte(marker + " tee\n")
	l.tee = string(tee)
	if b := l.must(tee, true, "post", "--tee", "--thread", root); !bytes.Equal(b, tee) {
		t.Fatal("server-mode tee output differs from stdin")
	}
	if err := l.discoverTee(); err != nil {
		t.Fatal(err)
	}
	t.Log("PASS: real stream threshold/EOF and environment-only service-mode tee")

	fixtures := []struct {
		name   string
		body   []byte
		thread bool
	}{
		{"binary.dat", bytes.Repeat([]byte{0, 1, 2, 127, 128, 255, '\n'}, 8192), false},
		{"thread.dat", []byte{255, 0, 128, 't', 'h', 'r', 'e', 'a', 'd'}, true},
		{"page.html", []byte("<!doctype html><html><body>scat E2E legitimate HTML attachment</body></html>\n"), true},
		{"response.json", []byte(`{"ok":false,"error":"fixture_content"}` + strings.Repeat(" ", 8192)), true},
	}
	expected := map[string][]byte{}
	fileThread := map[string]bool{}
	for _, f := range fixtures {
		args := []string{"upload", "--file", "-", "--filename", marker + "-" + f.name, "--comment", marker + " " + f.name, "-c", l.channel, "--json"}
		input := f.body
		if !f.thread {
			path := filepath.Join(t.TempDir(), f.name)
			if err := os.WriteFile(path, f.body, 0600); err != nil {
				t.Fatal(err)
			}
			args[2] = path
			input = nil
		} else {
			args = append(args, "--thread", root)
		}
		b, stderr, commandErr := l.run(input, false, args...)
		var r struct {
			Files   []struct{ ID string }
			Channel string
		}
		parseErr := json.Unmarshal(b, &r)
		if parseErr == nil && r.Channel == l.channel {
			for _, f := range r.Files {
				if f.ID != "" {
					l.files = append(l.files, f.ID)
				}
			}
		}
		l.checkResult("upload", stderr, commandErr)
		if parseErr != nil || len(r.Files) != 1 || r.Files[0].ID == "" || r.Channel != l.channel {
			t.Fatal("invalid completed-upload output")
		}
		id := r.Files[0].ID
		expected[id] = f.body
		fileThread[id] = f.thread
	}
	t.Log("PASS: root file and threaded binary/HTML/JSON uploads completed")

	// Slack file sharing may become visible after completion; poll read-only exports.
	var log export.ExportedLog
	for attempt := 0; attempt < 8; attempt++ {
		b := l.must(nil, false, "channel", "export", l.channel, "--start", start)
		log = decodeLog(t, b)
		found := map[string]bool{}
		for _, m := range log.Messages {
			for _, f := range m.Files {
				if _, ok := expected[f.ID]; ok {
					found[f.ID] = true
				}
			}
		}
		if len(found) == len(expected) {
			break
		}
		if attempt == 7 {
			t.Fatal("completed uploads did not become visible in channel/thread history")
		}
		time.Sleep(2 * time.Second)
	}
	dir := t.TempDir()
	output := filepath.Join(t.TempDir(), "export.json")
	l.must(nil, false, "channel", "export", l.channel, "--start", start, "--save-dir", dir, "--output", output)
	b, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	log = decodeLog(t, b)
	seen := map[string]bool{}
	found := map[string]bool{}
	messages := map[string]export.ExportedMessage{}
	rootIndex, replyIndex := -1, -1
	for i, m := range log.Messages {
		if seen[m.TimestampUnix] {
			t.Fatal("duplicate exported timestamp")
		}
		seen[m.TimestampUnix] = true
		messages[m.TimestampUnix] = m
		if m.TimestampUnix == root {
			rootIndex = i
			if m.PostType != "bot" || m.UserID == "" || m.Text != marker+" root" {
				t.Fatal("bot root export differs")
			}
		}
		if m.TimestampUnix == reply {
			replyIndex = i
			if !m.IsReply || m.ThreadTimestampUnix != root || len(m.Blocks) == 0 || len(m.Attachments) != 1 || len(m.Attachments[0].Fields) != 1 || m.Attachments[0].Fields[0].Value != "value" {
				t.Fatal("rich reply export differs")
			}
		}
		for _, f := range m.Files {
			want, ok := expected[f.ID]
			if !ok {
				continue
			}
			found[f.ID] = true
			if f.LocalPath == "" || !filepath.IsAbs(f.LocalPath) {
				t.Fatal("download missing absolute local_path")
			}
			got, err := os.ReadFile(f.LocalPath)
			if err != nil || !bytes.Equal(got, want) {
				t.Fatal("download bytes differ from original upload")
			}
			if m.IsReply != fileThread[f.ID] || (m.IsReply && m.ThreadTimestampUnix != root) {
				t.Fatal("file shared into wrong thread/root")
			}
		}
	}
	if rootIndex < 0 || replyIndex != rootIndex+1 || len(found) != len(expected) {
		t.Fatal("missing/grouped export messages or files")
	}
	combined := ""
	for _, ts := range streamTS {
		combined += messages[ts].Text
	}
	if combined != stream {
		t.Fatal("stream delivered text differs")
	}
	teeFound := false
	for _, m := range log.Messages {
		if m.Text == string(tee) && m.IsReply && m.ThreadTimestampUnix == root {
			teeFound = true
		}
	}
	if !teeFound {
		t.Fatal("tee output succeeded without matching Slack delivery")
	}
	t.Log("PASS: scli export schema, bot identity, rich fields, thread grouping, stream contents and all four downloaded byte sequences")

	selected := decodeLog(t, l.must(nil, false, "channel", "export", l.channel, "--start", start, "--end", end))
	expectedThread := map[string]bool{}
	for _, m := range log.Messages {
		if m.TimestampUnix == root || (m.IsReply && m.ThreadTimestampUnix == root) {
			expectedThread[m.TimestampUnix] = true
		}
	}
	if len(selected.Messages) != len(expectedThread) {
		t.Fatal("parent interval lost or added thread messages")
	}
	for _, m := range selected.Messages {
		if !expectedThread[m.TimestampUnix] {
			t.Fatal("parent interval changed the thread timestamp set")
		}
		delete(expectedThread, m.TimestampUnix)
	}
	if len(expectedThread) != 0 {
		t.Fatal("parent interval duplicated or omitted thread messages")
	}
	for _, m := range selected.Messages {
		if m.TimestampUnix != root && (!m.IsReply || m.ThreadTimestampUnix != root) {
			t.Fatal("parent interval included unrelated roots")
		}
		for _, f := range m.Files {
			if f.LocalPath != "" {
				t.Fatal("metadata-only export unexpectedly saved files")
			}
		}
	}
	text := l.must(nil, false, "channel", "export", l.channel, "--start", start, "--end", end, "--format", "text")
	if !bytes.Contains(text, []byte(marker+" root")) || !bytes.Contains(text, []byte(marker+" reply")) {
		t.Fatal("text export omitted root/reply")
	}
	t.Log("PASS: exclusive parent interval retains later replies; JSON/file output and text output")

	// Invalid credentials exercise real Slack failure, not a mocked API response.
	badEnv := append([]string{}, l.env...)
	l.env = append(badEnv, "SCAT_MODE=server", "SCAT_TOKEN=xoxb-invalid-e2e", "SCAT_CHANNEL="+l.channel)
	stdout, stderr, err := l.run(nil, false, "post", "must not be delivered")
	l.env = badEnv
	// run(false) supplies --config, so test server-mode conflict separately below.
	if err == nil || len(stdout) != 0 || !strings.Contains(stderr, "server mode") {
		t.Fatal("server mode accepted local configuration")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, l.binary, "post", "must not be delivered")
	cmd.Env = append(badEnv, "SCAT_MODE=server", "SCAT_TOKEN=xoxb-invalid-e2e", "SCAT_CHANNEL="+l.channel)
	var sout, serr bytes.Buffer
	cmd.Stdout = &sout
	cmd.Stderr = &serr
	if err := cmd.Run(); err == nil || sout.Len() != 0 || !strings.Contains(serr.String(), "invalid_auth") || strings.Contains(serr.String(), "xoxb-invalid-e2e") {
		t.Fatal("real invalid-token failure lacks safe API diagnostic or reports success")
	}
	t.Log("PASS: real invalid credentials fail without success output or token disclosure")
}
