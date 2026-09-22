package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/nlink-jp/scat/internal/config"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type roundTrip func(*http.Request) (*http.Response, error)

func (f roundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func reply(r *http.Request, s string) *http.Response {
	return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(s)), ContentLength: int64(len(s)), Request: r}
}
func run(t *testing.T, env map[string]string, input string, transport roundTrip, args ...string) (string, string, error) {
	t.Helper()
	out, diag := new(bytes.Buffer), new(bytes.Buffer)
	if transport == nil {
		transport = func(*http.Request) (*http.Response, error) { t.Fatal("unexpected network request"); return nil, nil }
	}
	c := NewCommand(Dependencies{Getenv: func(k string) string { return env[k] }, HTTP: &http.Client{Transport: transport}, Prompt: func(io.Writer) (string, error) { return "fixture-token", nil }, Now: func() time.Time { return time.Date(2026, 9, 22, 0, 0, 0, 0, time.UTC) }})
	c.SetIn(strings.NewReader(input))
	c.SetOut(out)
	c.SetErr(diag)
	c.SetArgs(args)
	e := c.Execute()
	return out.String(), diag.String(), e
}
func service() map[string]string {
	return map[string]string{"SCAT_MODE": "server", "SCAT_TOKEN": "fixture-token", "SCAT_CHANNEL": "C1"}
}
func postTransport(t *testing.T, check func(map[string]any)) roundTrip {
	return func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/api/auth.test":
			return reply(r, `{"ok":true,"bot_id":"B1","team_id":"T1"}`), nil
		case "/api/chat.postMessage":
			var b map[string]any
			if e := json.NewDecoder(r.Body).Decode(&b); e != nil {
				t.Fatal(e)
			}
			if check != nil {
				check(b)
			}
			return reply(r, `{"ok":true,"ts":"1.000001","channel":"C1"}`), nil
		case "/api/conversations.open":
			return reply(r, `{"ok":true,"channel":{"id":"D1"}}`), nil
		}
		t.Fatal(r.URL.Path)
		return nil, nil
	}
}
func TestPostInputPrecedenceAndOutput(t *testing.T) {
	path := filepath.Join(t.TempDir(), "input")
	os.WriteFile(path, []byte("from file"), 0600)
	for _, tt := range []struct {
		name, input, want string
		args              []string
	}{{"argument", "stdin", "arg", []string{"post", "arg", "--from-file", path}}, {"file", "stdin", "from file", []string{"post", "--from-file", path}}, {"stdin", "stdin", "stdin", []string{"post"}}} {
		t.Run(tt.name, func(t *testing.T) {
			out, _, e := run(t, service(), tt.input, postTransport(t, func(p map[string]any) {
				if p["text"] != tt.want || p["channel"] != "C1" {
					t.Fatal(p)
				}
			}), tt.args...)
			if e != nil || out != "1.000001\n" {
				t.Fatal(out, e)
			}
		})
	}
	out, _, e := run(t, service(), "hi", postTransport(t, nil), "post", "--json", "--quiet")
	if e != nil || out != "{\"ts\":\"1.000001\",\"channel\":\"C1\"}\n" {
		t.Fatal(out, e)
	}
	out, _, e = run(t, service(), "hi\n", postTransport(t, nil), "post", "--tee")
	if e != nil || out != "hi\n" {
		t.Fatal(out, e)
	}
}
func TestPostRichAndDMOptions(t *testing.T) {
	for _, format := range []string{"blocks", "payload"} {
		input := `{"blocks":[{"type":"divider"}],"text":"fallback","unfurl_links":true}`
		_, _, e := run(t, service(), input, postTransport(t, func(p map[string]any) {
			if p["channel"] != "D1" || p["thread_ts"] != "1.123456" || p["icon_emoji"] != ":robot_face:" || p["username"] != "service" || p["blocks"] == nil {
				t.Fatal(p)
			}
			if format == "payload" && (p["text"] != "fallback" || p["unfurl_links"] != true) {
				t.Fatal(p)
			}
		}), "post", "--user", "U1", "--format", format, "--thread", "1.123456", "--username", "service", "--icon-emoji", ":robot_face:")
		if e != nil {
			t.Fatal(e)
		}
	}
}
func TestLocalValidationAndMigrationNeverCallsSlack(t *testing.T) {
	cases := [][]string{{"post", "x", "--user", "U1", "-c", "C1"}, {"post", "--format", "bad"}, {"post", "--format", "blocks"}, {"post", "--stream", "--format", "blocks"}, {"post", "--stream", "--dry-run"}, {"post", "--tee", "--json"}, {"post", "--thread", "bad"}, {"post", "--noop"}, {"post", "--silent"}, {"post", "--iconemoji", ":a:"}, {"upload", "--filetype", "text"}, {"channel", "export", "C1", "--start", "bad"}, {"channel", "export", "C1", "--format", "text", "--json"}, {"channel", "list", "--dry-run"}, {"upload"}, {"upload", "-f", "-"}, {"profile", "set", "token", "secret"}, {"export", "log", "-c", "C1"}}
	for _, args := range cases {
		_, _, e := run(t, service(), "", nil, args...)
		if e == nil {
			t.Fatal(args)
		}
	}
	for _, args := range [][]string{{"post", "hello", "--dry-run"}, {"upload", "-f", "-", "--filename", "a.txt", "--dry-run"}, {"channel", "create", "new-channel", "--dry-run"}, {"channel", "invite", "C1", "U1", "--dry-run"}} {
		out, diag, e := run(t, service(), "data", nil, args...)
		if e != nil || out != "" || !strings.Contains(diag, "Dry run") {
			t.Fatal(args, out, diag, e)
		}
	}
}
func TestServerIsolationAndLimits(t *testing.T) {
	for _, args := range [][]string{{"profile", "list"}, {"profile", "add", "p"}, {"config", "init"}, {"post", "x", "--config", "/missing"}, {"post", "x", "--profile", "p"}} {
		if _, _, e := run(t, service(), "", nil, args...); e == nil {
			t.Fatal(args)
		}
	}
	for _, env := range []map[string]string{{"SCAT_MODE": "other"}, {"SCAT_MODE": "server"}, {"SCAT_MODE": "server", "SCAT_PROVIDER": "slack", "SCAT_TOKEN": "fixture"}, {"SCAT_MODE": "server", "SCAT_TOKEN": "fixture", "SCAT_MAX_FILE_SIZE": "-1"}} {
		if _, _, e := run(t, env, "", nil, "post", "x"); e == nil {
			t.Fatal(env)
		}
	}
	env := service()
	env["SCAT_MAX_STDIN_SIZE"] = "3"
	if _, _, e := run(t, env, "abcd", nil, "post"); e == nil {
		t.Fatal("limit bypass")
	}
	for _, args := range [][]string{{"--help"}, {"--version"}, {"-V"}} {
		if _, _, e := run(t, map[string]string{"SCAT_MODE": "bad"}, "", nil, args...); e != nil {
			t.Fatal(args, e)
		}
	}
}
func TestProfilesLifecycleAndNoTokenOutput(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	env := map[string]string{}
	invoke := func(args ...string) (string, string, error) {
		return run(t, env, "", nil, append([]string{"--config", path}, args...)...)
	}
	if _, _, e := invoke("profile", "list"); e == nil {
		t.Fatal("missing config accepted")
	}
	if _, _, e := invoke("config", "init"); e != nil {
		t.Fatal(e)
	}
	if _, _, e := invoke("config", "init"); e != nil {
		t.Fatal(e)
	}
	for _, args := range [][]string{{"profile", "add", "second", "--channel", "C2"}, {"profile", "set", "token"}, {"profile", "set", "channel", "C1"}, {"profile", "set", "username", "service"}, {"profile", "set", "limits.max_file_size_bytes", "0"}, {"profile", "use", "second"}, {"--profile", "default", "profile", "set", "channel", "C3"}} {
		if _, _, e := invoke(args...); e != nil {
			t.Fatal(args, e)
		}
	}
	out, _, e := invoke("profile", "list")
	if e != nil || !strings.Contains(out, "* second") || strings.Contains(out, "fixture-token") {
		t.Fatal(out, e)
	}
	for _, args := range [][]string{{"profile", "add", "second"}, {"profile", "remove", "default"}, {"profile", "remove", "second"}, {"profile", "remove", "missing"}, {"profile", "use", "missing"}, {"profile", "set", "unknown", "x"}, {"profile", "set", "provider", "mock"}, {"profile", "set", "limits.max_file_size_bytes", "-1"}, {"profile", "set", "token", "secret"}, {"profile", "list", "--json"}} {
		if _, _, e := invoke(args...); e == nil {
			t.Fatal(args)
		}
	}
	if _, _, e := invoke("profile", "use", "default"); e != nil {
		t.Fatal(e)
	}
	if _, _, e := invoke("profile", "remove", "second"); e != nil {
		t.Fatal(e)
	}
	cfg, e := config.Load(path)
	if e != nil || len(cfg.Profiles) != 1 || cfg.Profiles["default"].Channel != "C3" || cfg.Profiles["default"].Limits.MaxFileSizeBytes != 0 {
		t.Fatal(cfg, e)
	}
}
func TestSelectedProfileOnlyAndListArrays(t *testing.T) {
	cfg := config.NewDefaultConfig()
	cfg.Profiles["default"] = config.Profile{Token: "first", Limits: config.NewDefaultLimits()}
	cfg.Profiles["second"] = config.Profile{Token: "second", Limits: config.NewDefaultLimits()}
	path := filepath.Join(t.TempDir(), "c.json")
	cfg.Save(path)
	for _, kind := range []string{"channel", "user"} {
		calls := 0
		out, _, e := run(t, nil, "", func(r *http.Request) (*http.Response, error) {
			calls++
			if r.Header.Get("Authorization") != "Bearer second" {
				t.Fatal("wrong profile")
			}
			if r.URL.Path == "/api/auth.test" {
				return reply(r, `{"ok":true,"bot_id":"B2","team_id":"T2"}`), nil
			}
			if kind == "channel" {
				return reply(r, `{"ok":true,"channels":[]}`), nil
			}
			return reply(r, `{"ok":true,"members":[]}`), nil
		}, "--config", path, "-p", "second", kind, "list", "--json")
		if e != nil || out != "[]\n" || calls != 2 {
			t.Fatal(out, e, calls)
		}
	}
}
func TestUploadCommandFileAndStdin(t *testing.T) {
	path := filepath.Join(t.TempDir(), "file.txt")
	os.WriteFile(path, []byte("data"), 0600)
	for _, file := range []string{path, "-"} {
		out, _, e := run(t, service(), "data", func(r *http.Request) (*http.Response, error) {
			switch r.URL.Path {
			case "/api/auth.test":
				return reply(r, `{"ok":true,"bot_id":"B1","team_id":"T1"}`), nil
			case "/api/conversations.info":
				return reply(r, `{"ok":true,"channel":{"id":"C1","is_member":true}}`), nil
			case "/api/files.getUploadURLExternal":
				return reply(r, `{"ok":true,"upload_url":"https://files.slack.com/bytes","file_id":"F1"}`), nil
			case "/bytes":
				b, _ := io.ReadAll(r.Body)
				if string(b) != "data" {
					t.Fatal(string(b))
				}
				return reply(r, "OK"), nil
			case "/api/files.completeUploadExternal":
				var p map[string]any
				json.NewDecoder(r.Body).Decode(&p)
				if p["initial_comment"] != "comment" || p["thread_ts"] != "1.0" {
					t.Fatal(p)
				}
				return reply(r, `{"ok":true,"files":[{"id":"F1"}]}`), nil
			}
			t.Fatal(r.URL.Path)
			return nil, nil
		}, "upload", "-f", file, "--filename", "file.txt", "--comment", "comment", "--thread", "1.0", "--json")
		if e != nil || out != "{\"files\":[{\"id\":\"F1\"}],\"channel\":\"C1\"}\n" {
			t.Fatal(out, e)
		}
	}
}
func TestCreateInviteCommands(t *testing.T) {
	for _, args := range [][]string{{"channel", "create", "new", "--topic", "topic", "--description", "purpose", "--invite", "U1", "--json"}, {"channel", "invite", "C1", "U1", "--json"}} {
		out, _, e := run(t, service(), "", func(r *http.Request) (*http.Response, error) {
			if r.URL.Path == "/api/auth.test" {
				return reply(r, `{"ok":true,"bot_id":"B1","team_id":"T1"}`), nil
			}
			if r.URL.Path == "/api/conversations.create" {
				return reply(r, `{"ok":true,"channel":{"id":"C1","name":"new"}}`), nil
			}
			return reply(r, `{"ok":true}`), nil
		}, args...)
		if e != nil || !strings.Contains(out, "C1") {
			t.Fatal(out, e)
		}
	}
}
func TestExportFailureDoesNotReplaceFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "export.json")
	os.WriteFile(path, []byte("old"), 0600)
	out, _, e := run(t, service(), "", func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/api/auth.test":
			return reply(r, `{"ok":true,"bot_id":"B1","team_id":"T1"}`), nil
		case "/api/conversations.info":
			return reply(r, `{"ok":true,"channel":{"id":"C1","name":"general"}}`), nil
		default:
			return reply(r, `{"ok":false,"error":"missing_scope","needed":"channels:history"}`), nil
		}
	}, "channel", "export", "C1", "--output", path)
	b, _ := os.ReadFile(path)
	if e == nil || out != "" || string(b) != "old" {
		t.Fatal(out, e, string(b))
	}
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errors.New("input broke") }
func TestStreamSplitsFlushesErrorsAndCancellation(t *testing.T) {
	var batches []string
	send := func(s string) error { batches = append(batches, s); return nil }
	input := strings.Repeat("あ", 4001)
	out := new(bytes.Buffer)
	if e := streamText(context.Background(), strings.NewReader(input), out, true, 0, nil, send); e != nil {
		t.Fatal(e)
	}
	if len(batches) != 2 || len([]rune(batches[0])) != 4000 || batches[1] != "あ" || out.String() != input {
		t.Fatal("stream split/tee mismatch")
	}
	if e := streamText(context.Background(), errReader{}, io.Discard, false, 0, nil, send); e == nil {
		t.Fatal("reader failure hidden")
	}
	calls := 0
	if e := streamText(context.Background(), strings.NewReader(input), io.Discard, false, 0, nil, func(string) error { calls++; return errors.New("post failed") }); e == nil || calls != 1 {
		t.Fatal("delivery failure hidden", calls, e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if e := streamText(ctx, strings.NewReader("x"), io.Discard, false, 0, nil, send); !errors.Is(e, context.Canceled) {
		t.Fatal(e)
	}
	if e := streamText(context.Background(), strings.NewReader("abcd"), io.Discard, false, 3, nil, send); e == nil {
		t.Fatal("stream limit ignored")
	}
}

func TestCanceledEmptyStreamNeverSucceeds(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for i := 0; i < 100; i++ {
		if err := streamText(ctx, strings.NewReader(""), io.Discard, false, 0, nil, func(string) error { t.Fatal("sent after cancellation"); return nil }); !errors.Is(err, context.Canceled) {
			t.Fatal("EOF won over cancellation", err)
		}
	}
}
