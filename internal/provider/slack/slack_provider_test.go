package slack

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/magifd2/scat/internal/appcontext"
	"github.com/magifd2/scat/internal/config"
	"github.com/magifd2/scat/internal/export"
	"github.com/magifd2/scat/internal/provider"
)

// mockServerTransport is a custom http.RoundTripper that rewrites request URLs to the mock server.
type mockServerTransport struct {
	serverURL *url.URL
}

// RoundTrip rewrites the request URL's Scheme and Host to point to the mock server
// before delegating the request to the default transport.
func (t *mockServerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// For the actual file upload, the URL is already the mock server's URL, so we don't rewrite it.
	if strings.HasPrefix(req.URL.String(), t.serverURL.String()) {
		return http.DefaultTransport.RoundTrip(req)
	}
	req.URL.Scheme = t.serverURL.Scheme
	req.URL.Host = t.serverURL.Host
	return http.DefaultTransport.RoundTrip(req)
}

// newTestProvider creates a provider configured to use the mock server.
func newTestProvider(server *httptest.Server, channelName string) *Provider {
	profile := config.Profile{Token: "test-token", Channel: channelName}
	ctx := appcontext.NewContext(false, false, false, "", false, nil)
	serverURL, _ := url.Parse(server.URL)

	// Create a custom http client with our transport that rewrites URLs.
	client := &http.Client{
		Transport: &mockServerTransport{serverURL: serverURL},
	}

	p := &Provider{
		Profile:        profile,
		Context:        ctx,
		httpClient:     client,
		channelIDCache: make(map[string]string),
		userIDCache:    make(map[string]string),
	}
	// Pre-populate caches for tests
	p.channelIDCache[channelName] = "C01TEST"
	p.userIDCache["user_one"] = "U01"
	p.userIDCache["user_two"] = "U02"
	p.userIDCache["user_three"] = "U03"

	return p
}

func TestPostMessage(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/chat.postMessage", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok": true, "ts": "12345.67890"}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	p := newTestProvider(server, "general")

	opts := provider.PostMessageOptions{
		Text: "hello world",
	}

	if err := p.PostMessage(opts); err != nil {
		t.Errorf("PostMessage() returned an unexpected error: %v", err)
	}
}

func TestPostFile(t *testing.T) {
	// Create a dummy file to upload
	tempDir := t.TempDir()
	filePath := tempDir + "/test.txt"
	if err := os.WriteFile(filePath, []byte("hello file"), 0666); err != nil {
		t.Fatal(err)
	}

	var server *httptest.Server
	mux := http.NewServeMux()
	mux.HandleFunc("/api/files.getUploadURLExternal", func(w http.ResponseWriter, r *http.Request) {
		uploadURL := server.URL + "/upload-here"
		resp := fmt.Sprintf(`{"ok": true, "upload_url": "%s", "file_id": "F01"}`, uploadURL)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(resp))
	})
	mux.HandleFunc("/upload-here", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/api/files.completeUploadExternal", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok": true, "files": []}`))
	})

	server = httptest.NewServer(mux)
	defer server.Close()

	p := newTestProvider(server, "general")

	opts := provider.PostFileOptions{
		FilePath: filePath,
		Filename: "test.txt",
		Comment:  "a test file",
	}

	if err := p.PostFile(opts); err != nil {
		t.Errorf("PostFile() returned an unexpected error: %v", err)
	}
}

func TestExportLog(t *testing.T) {
	var server *httptest.Server
	mux := http.NewServeMux()

	// conversations.history handler with pagination
	mux.HandleFunc("/api/conversations.history", func(w http.ResponseWriter, r *http.Request) {
		cursor := r.URL.Query().Get("cursor")
		var resp string
		if cursor == "" {
			// First page
			resp = fmt.Sprintf(`{
				"ok": true,
				"messages": [{"type": "message", "user": "U01", "text": "message 2 <@U02>", "ts": "1678886402.000000", "files": [{"id": "F01", "name": "file1.txt", "url_private_download": "%s/download/file1.txt"}]}],
				"has_more": true,
				"response_metadata": {"next_cursor": "cursor123"}
			}`, server.URL)
		} else if cursor == "cursor123" {
			// Second page
			resp = `{
				"ok": true,
				"messages": [{"type": "message", "user": "U02", "text": "message 1", "ts": "1678886401.000000"}],
				"has_more": false
			}`
		}
		_, _ = w.Write([]byte(resp))
	})

	// users.info handler
	mux.HandleFunc("/api/users.info", func(w http.ResponseWriter, r *http.Request) {
		userID := r.URL.Query().Get("user")
		userName := ""
		if userID == "U01" {
			userName = "user_one"
		} else if userID == "U02" {
			userName = "user_two"
		}
		resp := fmt.Sprintf(`{"ok": true, "user": {"id": "%s", "name": "%s"}}`, userID, userName)
		_, _ = w.Write([]byte(resp))
	})

	// File download handler
	mux.HandleFunc("/download/file1.txt", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("dummy file content"))
	})

	server = httptest.NewServer(mux)
	defer server.Close()

	tempDir := t.TempDir()
	p := newTestProvider(server, "test-export")
	opts := export.Options{
		ChannelName:  "test-export",
		IncludeFiles: true,
		OutputDir:    tempDir,
	}

	log, err := p.ExportLog(opts)
	if err != nil {
		t.Fatalf("ExportLog() returned an unexpected error: %v", err)
	}

	if len(log.Messages) != 2 {
		t.Fatalf("Expected 2 messages, got %d", len(log.Messages))
	}

	// Messages are now sorted chronologically
	if log.Messages[0].UserName != "user_two" {
		t.Errorf("Expected first message from user_two, got %s", log.Messages[0].UserName)
	}
	if log.Messages[1].UserName != "user_one" {
		t.Errorf("Expected second message from user_one, got %s", log.Messages[1].UserName)
	}
	if !strings.Contains(log.Messages[1].Text, "@user_two") {
		t.Errorf("Expected mention to be resolved, got: %s", log.Messages[1].Text)
	}
	if len(log.Messages[1].Files) != 1 {
		t.Fatal("Expected 1 file attached to the message")
	}
	if log.Messages[1].Files[0].LocalPath == "" {
		t.Error("Expected LocalPath to be set for downloaded file")
	}

	// Check downloaded file content
	fileContent, err := os.ReadFile(log.Messages[1].Files[0].LocalPath)
	if err != nil {
		t.Fatalf("Failed to read downloaded file: %v", err)
	}
	if string(fileContent) != "dummy file content" {
		t.Errorf("Unexpected file content: got %s", string(fileContent))
	}
}

func TestExportLog_WithThread(t *testing.T) {
	mux := http.NewServeMux()

	// conversations.history handler
	mux.HandleFunc("/api/conversations.history", func(w http.ResponseWriter, r *http.Request) {
		resp := `{
			"ok": true,
			"messages": [
				{"type": "message", "user": "U01", "text": "Thread parent", "ts": "1678886400.000000", "reply_count": 2},
				{"type": "message", "user": "U02", "text": "A regular message", "ts": "1678886410.000000"}
			],
			"has_more": false
		}`
		_, _ = w.Write([]byte(resp))
	})

	// conversations.replies handler
	mux.HandleFunc("/api/conversations.replies", func(w http.ResponseWriter, r *http.Request) {
		ts := r.URL.Query().Get("ts")
		if ts != "1678886400.000000" {
			t.Errorf("Expected replies request for ts 1678886400.000000, got %s", ts)
		}
		resp := `{
			"ok": true,
			"messages": [
				{"type": "message", "user": "U01", "text": "Thread parent", "ts": "1678886400.000000", "thread_ts": "1678886400.000000"},
				{"type": "message", "user": "U02", "text": "Reply 1", "ts": "1678886401.000000", "thread_ts": "1678886400.000000"},
				{"type": "message", "user": "U03", "text": "Reply 2", "ts": "1678886402.000000", "thread_ts": "1678886400.000000"}
			],
			"has_more": false
		}`
		_, _ = w.Write([]byte(resp))
	})

	// users.info handler
	mux.HandleFunc("/api/users.info", func(w http.ResponseWriter, r *http.Request) {
		userID := r.URL.Query().Get("user")
		userName := ""
		switch userID {
		case "U01":
			userName = "user_one"
		case "U02":
			userName = "user_two"
		case "U03":
			userName = "user_three"
		}
		resp := fmt.Sprintf(`{"ok": true, "user": {"id": "%s", "name": "%s"}}`, userID, userName)
		_, _ = w.Write([]byte(resp))
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	p := newTestProvider(server, "test-thread-export")
	opts := export.Options{ChannelName: "test-thread-export"}

	log, err := p.ExportLog(opts)
	if err != nil {
		t.Fatalf("ExportLog() returned an unexpected error: %v", err)
	}

	if len(log.Messages) != 4 {
		t.Fatalf("Expected 4 messages (1 parent, 2 replies, 1 regular), got %d", len(log.Messages))
	}

	// Check chronological order
	expectedOrder := []string{"1678886400.000000", "1678886401.000000", "1678886402.000000", "1678886410.000000"}
	for i, ts := range expectedOrder {
		if log.Messages[i].TimestampUnix != ts {
			t.Errorf("Message %d: expected timestamp %s, got %s", i, ts, log.Messages[i].TimestampUnix)
		}
	}

	// Check parent message
	if log.Messages[0].IsReply {
		t.Error("Expected parent message IsReply to be false")
	}

	// Check replies
	if !log.Messages[1].IsReply {
		t.Error("Expected first reply IsReply to be true")
	}
	if log.Messages[1].ThreadTimestampUnix != "1678886400.000000" {
		t.Errorf("Expected first reply ThreadTimestampUnix to be 1678886400.000000, got %s", log.Messages[1].ThreadTimestampUnix)
	}
	if !log.Messages[2].IsReply {
		t.Error("Expected second reply IsReply to be true")
	}

	// Check regular message
	if log.Messages[3].IsReply {
		t.Error("Expected regular message IsReply to be false")
	}
	if log.Messages[3].ThreadTimestampUnix != "" {
		t.Errorf("Expected regular message ThreadTimestampUnix to be empty, got %s", log.Messages[3].ThreadTimestampUnix)
	}
}

func TestCreateChannel(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/conversations.create", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok": true, "channel": {"id": "C024BE91L", "name": "new-channel"}}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	p := newTestProvider(server, "general")

	opts := provider.CreateChannelOptions{
		Name: "new-channel",
	}
	channelID, err := p.CreateChannel(opts)
	if err != nil {
		t.Errorf("CreateChannel() returned an unexpected error: %v", err)
	}

	if channelID != "C024BE91L" {
		t.Errorf("Expected channel ID C024BE91L, got %s", channelID)
	}
}