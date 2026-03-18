package testprovider

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/magifd2/scat/internal/appcontext"
	"github.com/magifd2/scat/internal/config"
	"github.com/magifd2/scat/internal/export"
	"github.com/magifd2/scat/internal/provider"
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
	var p provider.Interface
	var err error

	output := captureStderr(func() {
		p, err = NewProvider(config.Profile{}, appcontext.Context{})
	})

	if err != nil {
		t.Fatalf("NewProvider() error = %v", err)
	}
	if p == nil {
		t.Fatal("NewProvider() returned nil provider")
	}
	if !strings.Contains(output, "[TESTPROVIDER] NewProvider called") {
		t.Errorf("Expected output to contain NewProvider marker, got: %s", output)
	}
}

func TestCapabilities(t *testing.T) {
	p, _ := NewProvider(config.Profile{}, appcontext.Context{})
	caps := p.Capabilities()

	if !caps.CanListChannels {
		t.Error("Expected CanListChannels to be true")
	}
	if !caps.CanPostFile {
		t.Error("Expected CanPostFile to be true")
	}
	if !caps.CanUseIconEmoji {
		t.Error("Expected CanUseIconEmoji to be true")
	}
	if !caps.CanExportLogs {
		t.Error("Expected CanExportLogs to be true")
	}
}

func TestPostMessage(t *testing.T) {
	p, _ := NewProvider(config.Profile{}, appcontext.Context{})
	opts := provider.PostMessageOptions{Text: "hello test"}

	output := captureStderr(func() {
		err := p.PostMessage(opts)
		if err != nil {
			t.Errorf("PostMessage() error = %v", err)
		}
	})

	if !strings.Contains(output, "[TESTPROVIDER] PostMessage called") {
		t.Errorf("Expected output to contain PostMessage marker, got: %s", output)
	}
	if !strings.Contains(output, "Text:hello test") {
		t.Errorf("Expected output to contain the message text, got: %s", output)
	}
}

func TestPostMessage_WithBlocks(t *testing.T) {
	ctx := appcontext.NewContext(false, false, false, "", false, nil)
	p, _ := NewProvider(config.Profile{}, ctx)

	blocksJSON := []byte(`[{"type": "section", "text": {"type": "mrkdwn", "text": "Hello, Block Kit!"}}]`)
	opts := provider.PostMessageOptions{Blocks: blocksJSON}

	output := captureStderr(func() {
		err := p.PostMessage(opts)
		if err != nil {
			t.Errorf("PostMessage() error = %v", err)
		}
	})

	if !strings.Contains(output, "[TESTPROVIDER] PostMessage called") {
		t.Errorf("Expected output to contain PostMessage marker, got: %s", output)
	}
	if !strings.Contains(output, fmt.Sprintf("Blocks:%s", string(blocksJSON))) {
		t.Errorf("Expected output to contain the blocks JSON, got: %s", output)
	}
	// Ensure Text field is empty when blocks are used
	if strings.Contains(output, "Text:") && !strings.Contains(output, "Text: ") {
		t.Errorf("Expected Text field to be empty or not present when blocks are used, got: %s", output)
	}
}

func TestListChannels(t *testing.T) {
	p, _ := NewProvider(config.Profile{}, appcontext.Context{})
	var err error

	output := captureStderr(func() {
		var channels []provider.Channel
		channels, err = p.ListChannels()
		if err != nil {
			return
		}
		if len(channels) == 0 {
			t.Error("Expected non-empty channel list")
		}
		if channels[0].ID == "" || channels[0].Name == "" {
			t.Errorf("Expected channel with ID and Name, got: %+v", channels[0])
		}
	})

	if err != nil {
		t.Fatalf("ListChannels() error = %v", err)
	}
	if !strings.Contains(output, "[TESTPROVIDER] ListChannels called") {
		t.Errorf("Expected output to contain ListChannels marker, got: %s", output)
	}
}

func TestExportLog(t *testing.T) {
	p, _ := NewProvider(config.Profile{}, appcontext.Context{})
	opts := export.Options{ChannelName: "#export-test", IncludeFiles: true, OutputDir: t.TempDir()}

	var logData *export.ExportedLog
	var err error

	output := captureStderr(func() {
		logData, err = p.ExportLog(opts)
	})

	if err != nil {
		t.Fatalf("ExportLog() error = %v", err)
	}
	if logData == nil {
		t.Fatal("ExportLog() returned nil data")
	}
	if !strings.Contains(output, "[TESTPROVIDER] ExportLog called") {
		t.Errorf("Expected output to contain ExportLog marker, got: %s", output)
	}
	if !strings.Contains(output, "IncludeFiles:true") {
		t.Errorf("Expected output to reflect IncludeFiles option, got: %s", output)
	}
	if logData.ChannelName != opts.ChannelName {
		t.Errorf("logData.ChannelName = %s, want %s", logData.ChannelName, opts.ChannelName)
	}
	if len(logData.Messages) == 0 {
		t.Error("Expected non-empty messages")
	}
	if len(logData.Messages[0].Files) == 0 {
		t.Error("Expected non-empty files list when IncludeFiles is true")
	}
}
