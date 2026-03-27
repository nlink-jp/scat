package cmd

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/nlink-jp/scat/internal/provider"
	_ "github.com/nlink-jp/scat/internal/provider/testprovider"
)

func TestChannelList_Default(t *testing.T) {
	configPath, cleanup := setupTest(t)
	defer cleanup()

	rootCmd := newRootCmd()
	rootCmd.AddCommand(newChannelCmd())

	stdout, stderr, err := testExecuteCommandAndCapture(rootCmd, "--config", configPath, "channel", "list")
	if err != nil {
		t.Fatalf("testExecuteCommandAndCapture returned an error: %v\nStderr: %s", err, stderr)
	}

	// Table output should include header and both channel names and IDs
	for _, expected := range []string{"NAME", "ID", "test-channel-1", "C0000000001", "test-channel-2", "C0000000002"} {
		if !strings.Contains(stdout, expected) {
			t.Errorf("Expected stdout to contain '%s', got: '%s'", expected, stdout)
		}
	}

	expectedStderr := "Channels for profile: test"
	if !strings.Contains(stderr, expectedStderr) {
		t.Errorf("Expected stderr to contain '%s', got: '%s'", expectedStderr, stderr)
	}
}

func TestChannelList_JSON(t *testing.T) {
	configPath, cleanup := setupTest(t)
	defer cleanup()

	rootCmd := newRootCmd()
	rootCmd.AddCommand(newChannelCmd())

	stdout, _, err := testExecuteCommandAndCapture(rootCmd, "--config", configPath, "channel", "list", "--json")
	if err != nil {
		t.Fatalf("testExecuteCommandAndCapture returned an error: %v", err)
	}

	var result map[string][]provider.Channel
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("Failed to unmarshal json output: %v", err)
	}

	if _, ok := result["test"]; !ok {
		t.Fatal("Expected 'test' profile in json output")
	}

	if len(result["test"]) != 2 {
		t.Errorf("Expected 2 channels for test profile, got %d", len(result["test"]))
	}

	found := false
	for _, ch := range result["test"] {
		if ch.Name == "test-channel-1" && ch.ID == "C0000000001" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected channel {test-channel-1, C0000000001} in results, got: %+v", result["test"])
	}
}