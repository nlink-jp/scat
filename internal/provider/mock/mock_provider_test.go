package mock

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/nlink-jp/scat/internal/appcontext"
	"github.com/nlink-jp/scat/internal/config"
	"github.com/nlink-jp/scat/internal/export"
	"github.com/nlink-jp/scat/internal/provider"
)

// captureStderr captures everything written to os.Stderr during the execution of a function.
func captureStderr(f func()) string {
	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		log.Fatalf("failed to create pipe: %v", err)
	}
	os.Stderr = w

	defer func() {
		os.Stderr = oldStderr
	}()

	f()
	_ = w.Close()

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	return buf.String()
}

func TestNewProvider(t *testing.T) {
	p, err := NewProvider(config.Profile{}, appcontext.Context{})
	if err != nil {
		t.Fatalf("NewProvider() error = %v", err)
	}
	if p == nil {
		t.Fatal("NewProvider() returned nil provider")
	}
}

func TestCapabilities(t *testing.T) {
	p, _ := NewProvider(config.Profile{}, appcontext.Context{})
	caps := p.Capabilities()

	if caps.CanListChannels {
		t.Error("Expected CanListChannels to be false")
	}
	if !caps.CanPostFile {
		t.Error("Expected CanPostFile to be true")
	}
	if !caps.CanExportLogs {
		t.Error("Expected CanExportLogs to be true")
	}
}

func TestPostMessage(t *testing.T) {
	ctx := appcontext.NewContext(false, false, false, "", false, nil)
	profile := config.Profile{Channel: "#mock-channel"}
	p, _ := NewProvider(profile, ctx)

	opts := provider.PostMessageOptions{Text: "hello world"}

	output := captureStderr(func() {
		err := p.PostMessage(opts)
		if err != nil {
			t.Errorf("PostMessage() error = %v", err)
		}
	})

	if !strings.Contains(output, "[MOCK] PostMessage called") {
		t.Errorf("Expected output to contain PostMessage marker, got: %s", output)
	}
	if !strings.Contains(output, "Text: hello world") {
		t.Errorf("Expected output to contain the message text, got: %s", output)
	}
	if !strings.Contains(output, "Channel: #mock-channel") {
		t.Errorf("Expected output to contain the default channel, got: %s", output)
	}
}

func TestPostMessage_WithTargetChannel(t *testing.T) {
	ctx := appcontext.NewContext(false, false, false, "", false, nil)
	profile := config.Profile{Channel: "#default-channel"}
	p, _ := NewProvider(profile, ctx)

	opts := provider.PostMessageOptions{TargetChannel: "#override-channel", Text: "hello world"}

	output := captureStderr(func() {
		err := p.PostMessage(opts)
		if err != nil {
			t.Errorf("PostMessage() error = %v", err)
		}
	})

	if !strings.Contains(output, "Channel: #override-channel") {
		t.Errorf("Expected output to contain the override channel, got: %s", output)
	}
}

func TestPostMessage_Silent(t *testing.T) {
	ctx := appcontext.NewContext(false, false, true, "", false, nil) // Silent = true
	profile := config.Profile{Channel: "#mock-channel"}
	p, _ := NewProvider(profile, ctx)

	opts := provider.PostMessageOptions{Text: "hello world"}

	output := captureStderr(func() {
		err := p.PostMessage(opts)
		if err != nil {
			t.Errorf("PostMessage() error = %v", err)
		}
	})

	if output != "" {
		t.Errorf("Expected empty output when silent, got: %s", output)
	}
}

func TestPostMessage_Debug(t *testing.T) {
	ctx := appcontext.NewContext(true, false, false, "", false, nil) // Debug = true
	profile := config.Profile{Channel: "#mock-channel"}
	p, _ := NewProvider(profile, ctx)

	opts := provider.PostMessageOptions{Text: "debug message"}

	output := captureStderr(func() {
		err := p.PostMessage(opts)
		if err != nil {
			t.Errorf("PostMessage() error = %v", err)
		}
	})

	if !strings.Contains(output, "[DEBUG] Mock PostMessage") {
		t.Errorf("Expected output to contain debug marker, got: %s", output)
	}
}

func TestPostMessage_WithBlocks(t *testing.T) {
	ctx := appcontext.NewContext(false, false, false, "", false, nil)
	profile := config.Profile{Channel: "#mock-channel"}
	p, _ := NewProvider(profile, ctx)

	blocksJSON := []byte(`[{"type": "section", "text": {"type": "mrkdwn", "text": "Hello, Block Kit!"}}]`)
	opts := provider.PostMessageOptions{Blocks: blocksJSON}

	output := captureStderr(func() {
		err := p.PostMessage(opts)
		if err != nil {
			t.Errorf("PostMessage() error = %v", err)
		}
	})

	if !strings.Contains(output, "[MOCK] PostMessage called") {
		t.Errorf("Expected output to contain PostMessage marker, got: %s", output)
	}
	if !strings.Contains(output, fmt.Sprintf("Blocks: %s", string(blocksJSON))) {
		t.Errorf("Expected output to contain the blocks JSON, got: %s", output)
	}
	// Ensure Text field is empty if blocks are present (as per PostMessageOptions logic)
	if strings.Contains(output, "Text:") && !strings.Contains(output, "Text: "+opts.Text) {
		t.Errorf("Expected Text field to be empty or not present when blocks are used, got: %s", output)
	}
}

func TestListChannels(t *testing.T) {
	p, _ := NewProvider(config.Profile{}, appcontext.Context{})
	_, err := p.ListChannels()
	if err == nil {
		t.Fatal("Expected an error from ListChannels, got nil")
	}
}

func TestExportLog(t *testing.T) {
	ctx := appcontext.NewContext(false, false, false, "", false, nil)
	p, _ := NewProvider(config.Profile{}, ctx)

	opts := export.Options{ChannelName: "#test-channel"}

	var log *export.ExportedLog
	output := captureStderr(func() {
		var err error
		log, err = p.ExportLog(opts)
		if err != nil {
			t.Errorf("ExportLog() error = %v", err)
		}
	})

	if !strings.Contains(output, "[MOCK] ExportLog called") {
		t.Errorf("Expected output to contain ExportLog marker, got: %s", output)
	}
	if log == nil {
		t.Fatal("ExportLog() returned nil log")
	}
	if log.ChannelName != "#test-channel" {
		t.Errorf("log.ChannelName = %s, want #test-channel", log.ChannelName)
	}
	if len(log.Messages) == 0 {
		t.Error("Expected non-empty messages in exported log")
	}
}