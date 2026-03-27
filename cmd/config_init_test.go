package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nlink-jp/scat/internal/config"
	_ "github.com/nlink-jp/scat/internal/provider/testprovider"
)

func TestConfigInit_Success(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")

	rootCmd := newRootCmd()
	rootCmd.AddCommand(newConfigCmd())

	// Execute the command
	stdout, _, err := testExecuteCommandAndCapture(rootCmd, "--config", configPath, "config", "init")
	if err != nil {
		t.Fatalf("testExecuteCommandAndCapture returned an error: %v", err)
	}

	// Check the output message
	if !strings.Contains(stdout, "Successfully created a new configuration file") {
		t.Errorf("Expected success message, got: %s", stdout)
	}

	// Verify the file was created
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read created config file: %v", err)
	}

	// Verify the content is the default config
	var cfgFromFile config.Config
	if err := json.Unmarshal(content, &cfgFromFile); err != nil {
		t.Fatalf("Failed to unmarshal created config: %v", err)
	}

	defaultCfg := config.NewDefaultConfig()
	if cfgFromFile.CurrentProfile != defaultCfg.CurrentProfile {
		t.Errorf("Expected current profile to be '%s', got '%s'", defaultCfg.CurrentProfile, cfgFromFile.CurrentProfile)
	}
	if len(cfgFromFile.Profiles) != 1 {
		t.Errorf("Expected 1 default profile, got %d", len(cfgFromFile.Profiles))
	}
}

func TestConfigInit_AlreadyExists(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")

	// Create a dummy config file first
	initialContent := "{\"current_profile\": \"custom\"}"
	if err := os.WriteFile(configPath, []byte(initialContent), 0666); err != nil {
		t.Fatal(err)
	}

	rootCmd := newRootCmd()
	rootCmd.AddCommand(newConfigCmd())

	// Execute the command
	stdout, _, err := testExecuteCommandAndCapture(rootCmd, "--config", configPath, "config", "init")
	if err != nil {
		t.Fatalf("testExecuteCommandAndCapture returned an error: %v", err)
	}

	// Check the output message
	if !strings.Contains(stdout, "Configuration file already exists") {
		t.Errorf("Expected already exists message, got: %s", stdout)
	}

	// Verify the file content has not changed
	finalContent, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config file after command: %v", err)
	}
	if string(finalContent) != initialContent {
		t.Error("Expected file content to remain unchanged, but it was modified")
	}
}
