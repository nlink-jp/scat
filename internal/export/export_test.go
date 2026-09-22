package export

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/nlink-jp/scat/internal/slack"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

type source struct {
	history      []slack.Message
	fail         bool
	downloads    int
	start, end   string
	threadBounds bool
}

func (s *source) ResolveChannel(context.Context, string) (string, error) { return "C1", nil }
func (s *source) ChannelInfo(context.Context, string) (slack.Channel, error) {
	return slack.Channel{ID: "C1", Name: "general"}, nil
}
func (s *source) Messages(_ context.Context, _, thread, start, end string) ([]slack.Message, error) {
	if thread != "" {
		if s.fail {
			return nil, errors.New("missing_scope")
		}
		s.threadBounds = start != "" || end != ""
		return []slack.Message{s.history[2], s.history[1]}, nil
	}
	s.start = start
	s.end = end
	return append([]slack.Message(nil), s.history...), nil
}
func (s *source) UserName(_ context.Context, id string) string {
	return map[string]string{"U1": "Alice", "U2": "Bob"}[id]
}
func (s *source) Download(context.Context, slack.File, string) (string, error) {
	s.downloads++
	return "", errors.New("missing files:read")
}
func fixture(t *testing.T) *source {
	t.Helper()
	b, e := os.ReadFile("../../testdata/export/history.json")
	if e != nil {
		t.Fatal(e)
	}
	s := &source{}
	if e = json.Unmarshal(b, &s.history); e != nil {
		t.Fatal(e)
	}
	return s
}
func TestSCLIParityAndBroadcastDedup(t *testing.T) {
	s := fixture(t)
	warnings := 0
	now := func() time.Time { return time.Date(2026, 9, 22, 0, 0, 0, 0, time.UTC) }
	out, err := (Exporter{s, now, func(string) { warnings++ }}).Run(context.Background(), Request{Channel: "C1", Start: "2023-12-31T23:59:59.123456Z", End: "2024-01-01T00:00:04Z", SaveDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	got, _ := json.Marshal(out)
	expected, e := os.ReadFile("../../testdata/export/expected.json")
	if e != nil {
		t.Fatal(e)
	}
	var a, b any
	json.Unmarshal(got, &a)
	json.Unmarshal(expected, &b)
	if !reflect.DeepEqual(a, b) {
		t.Fatalf("parity mismatch:\n%s", got)
	}
	if warnings != 1 || s.downloads != 1 || s.threadBounds || s.start != "1704067199.123456" || s.end != "1704067204.000000" {
		t.Fatalf("bounds/warnings/downloads: %+v, %d", s, warnings)
	}
}
func TestFailureAndEmptyCollections(t *testing.T) {
	s := fixture(t)
	s.fail = true
	if out, err := (Exporter{Source: s}).Run(context.Background(), Request{}); err == nil || out != nil {
		t.Fatal("thread failure concealed")
	}
	s.history = nil
	s.fail = false
	out, err := (Exporter{Source: s}).Run(context.Background(), Request{})
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(out)
	if !bytes.Contains(b, []byte(`"messages":[]`)) {
		t.Fatal(string(b))
	}
}
func TestOrphanBroadcastAndBotWithoutUser(t *testing.T) {
	s := &source{history: []slack.Message{{TS: "1704067201.000001", BotID: "B1", ThreadTS: "1704067000.000001", Text: "orphan"}}}
	out, e := (Exporter{Source: s}).Run(context.Background(), Request{})
	if e != nil || len(out.Messages) != 1 || out.Messages[0].UserID != "B1" || !out.Messages[0].IsReply {
		t.Fatalf("%+v %v", out, e)
	}
}
func TestBoundsAndPrecision(t *testing.T) {
	for _, r := range [][2]string{{"bad", ""}, {"2024-01-01T00:00:00Z", "2024-01-01T00:00:00Z"}, {"2025-01-01T00:00:00Z", "2024-01-01T00:00:00Z"}} {
		if _, _, e := Bounds(r[0], r[1]); e == nil {
			t.Fatal(r)
		}
	}
	a, _, e := Bounds("2024-01-01T09:00:00.123456+09:00", "")
	if e != nil || a != "1704067200.123456" {
		t.Fatal(a, e)
	}
	v, e := ParseTimestamp("1704067200.999999")
	if e != nil || v.Nanosecond() != 999999000 {
		t.Fatal(v, e)
	}
}

type brokenWriter struct{}

func (brokenWriter) Write([]byte) (int, error) { return 0, errors.New("broken pipe") }
func TestAtomicOutputCancellationAndPipe(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.json")
	os.WriteFile(path, []byte("old"), 0600)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	log := &ExportedLog{Messages: []ExportedMessage{}}
	if Write(ctx, nil, path, "json", log) == nil {
		t.Fatal("cancel success")
	}
	data, _ := os.ReadFile(path)
	if string(data) != "old" {
		t.Fatal("replaced output")
	}
	if Write(context.Background(), brokenWriter{}, "-", "json", log) == nil {
		t.Fatal("broken pipe concealed")
	}
	if e := Write(context.Background(), nil, path, "json", log); e != nil {
		t.Fatal(e)
	}
}

func TestFractionalExclusiveEndRoundsUp(t *testing.T) {
	for _, tc := range []struct{ end, want string }{{"2024-01-01T00:00:00.123456789Z", "1704067200.123457"}, {"2024-01-01T00:00:00.999999999Z", "1704067201.000000"}, {"2024-01-01T00:00:00.123456Z", "1704067200.123456"}} {
		_, end, err := Bounds("", tc.end)
		if err != nil || end != tc.want {
			t.Fatal(tc, end, err)
		}
	}
}
