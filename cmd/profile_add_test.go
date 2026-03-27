package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nlink-jp/scat/internal/config"
)

func TestProfileAdd_Success(t *testing.T) {
	// Create a temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")
	configContent := `{
		"current_profile": "default",
		"profiles": {
			"default": {
				"provider": "mock"
			}
		}
	}`
	if err := os.WriteFile(configPath, []byte(configContent), 0666); err != nil {
		t.Fatal(err)
	}

	// Mock the password input
	oldReadPasswordFunc := readPasswordFunc
	readPasswordFunc = func(fd int) ([]byte, error) {
		return []byte("mock_token"), nil
	}
	defer func() { readPasswordFunc = oldReadPasswordFunc }()

	// Create a clean command tree for the test
	rootCmd := newRootCmd()
	rootCmd.AddCommand(newProfileCmd())

	// Execute the command
	stdout, stderr, err := testExecuteCommandAndCapture(rootCmd, "--config", configPath, "profile", "add", "new_profile", "--provider", "slack", "--channel", "#test")
	if err != nil {
		t.Fatalf("testExecuteCommandAndCapture returned an error: %v\nStderr: %s", err, stderr)
	}

	// Check stderr output
	if !strings.Contains(stderr, "Profile 'new_profile' added.") {
		t.Errorf("Expected stderr to contain 'Profile 'new_profile' added.', got: %s", stderr)
	}

	// Verify the config file was updated
	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load config after add command: %v", err)
	}
	if _, ok := cfg.Profiles["new_profile"]; !ok {
		t.Errorf("Expected 'new_profile' to exist in config")
	}
	if cfg.Profiles["new_profile"].Token != "mock_token" {
		t.Errorf("Expected token to be 'mock_token', got: %s", cfg.Profiles["new_profile"].Token)
	}

	if stdout != "" {
		t.Errorf("Expected no stdout, got: %s", stdout)
	}
}

func TestProfileAdd_AlreadyExists(t *testing.T) {
	// Create a temporary config file with a default profile
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")
	configContent := `{
		"current_profile": "default",
		"profiles": {
			"default": {
				"provider": "mock"
			}
		}
	}`
	if err := os.WriteFile(configPath, []byte(configContent), 0666); err != nil {
		t.Fatal(err)
	}

	// Create a clean command tree for the test
	rootCmd := newRootCmd()
	rootCmd.AddCommand(newProfileCmd())

	// Execute the command to add an existing profile
	stdout, stderr, err := testExecuteCommandAndCapture(rootCmd, "--config", configPath, "profile", "add", "default")
	if err == nil {
		t.Fatal("Expected an error for adding existing profile, but got nil")
	}

	// Check the error message
	if !strings.Contains(err.Error(), "Profile 'default' already exists") {
		t.Errorf("Expected error message to contain 'already exists', got: %v", err)
	}

	if stdout != "" {
		t.Errorf("Expected no stdout, got: %s", stdout)
	}

	// Stderr should be empty for this error type
	if stderr != "" {
		t.Errorf("Expected empty stderr, got: %s", stderr)
	}
}

func TestProfileAdd_NoConfig(t *testing.T) {
	// Point to a non-existent config file
	configPath := filepath.Join(t.TempDir(), "non-existent.json")

	// Create a clean command tree for the test
	rootCmd := newRootCmd()
	rootCmd.AddCommand(newProfileCmd())

	// Execute the command
	stdout, stderr, err := testExecuteCommandAndCapture(rootCmd, "--config", configPath, "profile", "add", "any_profile")
	if err == nil {
		t.Fatal("Expected an error for non-existent config, but got nil")
	}

	// Check the error message
	if !strings.Contains(err.Error(), "configuration file not found") {
		t.Errorf("Expected error message to contain 'configuration file not found', got: %v", err)
	}

	if stdout != "" {
		t.Errorf("Expected no stdout, got: %s", stdout)
	}

	// Stderr should be empty for this error type
	if stderr != "" {
		t.Errorf("Expected empty stderr, got: %s", stderr)
	}
}
