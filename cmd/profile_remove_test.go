
package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nlink-jp/scat/internal/config"
)

func TestProfileRemove_Success(t *testing.T) {
	// Create a temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")
	configContent := `{
		"current_profile": "default",
		"profiles": {
			"default": {
				"provider": "mock"
			},
			"removable_profile": {
				"provider": "slack"
			}
		}
	}`
	if err := os.WriteFile(configPath, []byte(configContent), 0666); err != nil {
		t.Fatal(err)
	}

	// Create a clean command tree for the test
	rootCmd := newRootCmd()
	rootCmd.AddCommand(newProfileCmd())

	// Execute the command
	stdout, stderr, err := testExecuteCommandAndCapture(rootCmd, "--config", configPath, "profile", "remove", "removable_profile")
	if err != nil {
		t.Fatalf("testExecuteCommandAndCapture returned an error: %v\nStderr: %s", err, stderr)
	}

	// Check stderr output
	if !strings.Contains(stderr, "Profile 'removable_profile' removed.") {
		t.Errorf("Expected stderr to contain 'Profile 'removable_profile' removed.', got: %s", stderr)
	}

	// Verify the config file was updated
	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load config after remove command: %v", err)
	}
	if _, ok := cfg.Profiles["removable_profile"]; ok {
		t.Errorf("Expected 'removable_profile' to be removed from config")
	}

	if stdout != "" {
		t.Errorf("Expected no stdout, got: %s", stdout)
	}
}

func TestProfileRemove_DefaultProfile(t *testing.T) {
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

	// Create a clean command tree for the test
	rootCmd := newRootCmd()
	rootCmd.AddCommand(newProfileCmd())

	// Execute the command
	stdout, stderr, err := testExecuteCommandAndCapture(rootCmd, "--config", configPath, "profile", "remove", "default")
	if err == nil {
		t.Fatal("Expected an error for removing default profile, but got nil")
	}

	// Check the error message
	if !strings.Contains(err.Error(), "the 'default' profile cannot be removed") {
		t.Errorf("Expected error message to contain 'default profile cannot be removed', got: %v", err)
	}

	// Verify the config file was NOT updated
	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load config after remove command: %v", err)
	}
	if _, ok := cfg.Profiles["default"]; !ok {
		t.Errorf("Expected 'default' profile to still exist in config")
	}

	if stdout != "" {
		t.Errorf("Expected no stdout, got: %s", stdout)
	}

	// Stderr should be empty for this error type
	if stderr != "" {
		t.Errorf("Expected empty stderr, got: %s", stderr)
	}
}

func TestProfileRemove_ActiveProfile(t *testing.T) {
	// Create a temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")
	configContent := `{
		"current_profile": "active_one",
		"profiles": {
			"default": {
				"provider": "mock"
			},
			"active_one": {
				"provider": "slack"
			}
		}
	}`
	if err := os.WriteFile(configPath, []byte(configContent), 0666); err != nil {
		t.Fatal(err)
	}

	// Create a clean command tree for the test
	rootCmd := newRootCmd()
	rootCmd.AddCommand(newProfileCmd())

	// Execute the command
	stdout, stderr, err := testExecuteCommandAndCapture(rootCmd, "--config", configPath, "profile", "remove", "active_one")
	if err == nil {
		t.Fatal("Expected an error for removing active profile, but got nil")
	}

	// Check the error message
	if !strings.Contains(err.Error(), "cannot remove the currently active profile") {
		t.Errorf("Expected error message to contain 'cannot remove the currently active profile', got: %v", err)
	}

	// Verify the config file was NOT updated
	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load config after remove command: %v", err)
	}
	if cfg.CurrentProfile != "active_one" {
		t.Errorf("Expected current profile to still be 'active_one', got: %s", cfg.CurrentProfile)
	}

	if stdout != "" {
		t.Errorf("Expected no stdout, got: %s", stdout)
	}

	// Stderr should be empty for this error type
	if stderr != "" {
		t.Errorf("Expected empty stderr, got: %s", stderr)
	}
}

func TestProfileRemove_ProfileNotFound(t *testing.T) {
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

	// Create a clean command tree for the test
	rootCmd := newRootCmd()
	rootCmd.AddCommand(newProfileCmd())

	// Execute the command
	stdout, stderr, err := testExecuteCommandAndCapture(rootCmd, "--config", configPath, "profile", "remove", "non_existent_profile")
	if err == nil {
		t.Fatal("Expected an error for non-existent profile, but got nil")
	}

	// Check the error message
	if !strings.Contains(err.Error(), "profile 'non_existent_profile' not found") {
		t.Errorf("Expected error message to contain 'profile not found', got: %v", err)
	}

	// Verify the config file was NOT updated
	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load config after remove command: %v", err)
	}
	if _, ok := cfg.Profiles["default"]; !ok {
		t.Errorf("Expected 'default' profile to still exist in config")
	}

	if stdout != "" {
		t.Errorf("Expected no stdout, got: %s", stdout)
	}

	// Stderr should be empty for this error type
	if stderr != "" {
		t.Errorf("Expected empty stderr, got: %s", stderr)
	}
}

func TestProfileRemove_NoConfig(t *testing.T) {
	// Point to a non-existent config file
	configPath := filepath.Join(t.TempDir(), "non-existent.json")

	// Create a clean command tree for the test
	rootCmd := newRootCmd()
	rootCmd.AddCommand(newProfileCmd())

	// Execute the command
	stdout, stderr, err := testExecuteCommandAndCapture(rootCmd, "--config", configPath, "profile", "remove", "any_profile")
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
