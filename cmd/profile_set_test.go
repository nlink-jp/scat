
package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nlink-jp/scat/internal/config"
)

func TestProfileSet_Success(t *testing.T) {
	// Create a temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")
	configContent := `{
		"current_profile": "default",
		"profiles": {
			"default": {
				"provider": "mock",
				"channel": "#general",
				"username": "old_user",
				"limits": {
					"max_file_size_bytes": 100,
					"max_stdin_size_bytes": 10
				}
			}
		}
	}`
	if err := os.WriteFile(configPath, []byte(configContent), 0666); err != nil {
		t.Fatal(err)
	}

	// Test cases for various key-value pairs
	tests := []struct {
		key      string
		value    string
		expected func(cfg *config.Config) bool
	}{
		{
			key:   "provider",
			value: "slack",
			expected: func(cfg *config.Config) bool {
				return cfg.Profiles["default"].Provider == "slack"
			},
		},
		{
			key:   "channel",
			value: "#new-channel",
			expected: func(cfg *config.Config) bool {
				return cfg.Profiles["default"].Channel == "#new-channel"
			},
		},
		{
			key:   "username",
			value: "new_user",
			expected: func(cfg *config.Config) bool {
				return cfg.Profiles["default"].Username == "new_user"
			},
		},
		{
			key:   "limits.max_file_size_bytes",
			value: "2000000",
			expected: func(cfg *config.Config) bool {
				return cfg.Profiles["default"].Limits.MaxFileSizeBytes == 2000000
			},
		},
		{
			key:   "limits.max_stdin_size_bytes",
			value: "200000",
			expected: func(cfg *config.Config) bool {
				return cfg.Profiles["default"].Limits.MaxStdinSizeBytes == 200000
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			// Create a clean command tree for the test
			rootCmd := newRootCmd()
			rootCmd.AddCommand(newProfileCmd())

			// Execute the command
			stdout, stderr, err := testExecuteCommandAndCapture(rootCmd, "--config", configPath, "profile", "set", tt.key, tt.value)
			if err != nil {
				t.Fatalf("testExecuteCommandAndCapture returned an error: %v\nStderr: %s", err, stderr)
			}

			// Verify the config file was updated
			cfg, err := config.Load(configPath)
			if err != nil {
				t.Fatalf("Failed to load config after set command: %v", err)
			}
			if !tt.expected(cfg) {
				t.Errorf("Expected %s to be updated to %s, but it was not.", tt.key, tt.value)
			}

			if stdout != "" {
				t.Errorf("Expected no stdout, got: %s", stdout)
			}
			if !strings.Contains(stderr, fmt.Sprintf("Set %s in profile default", tt.key)) {
				t.Errorf("Expected stderr to contain 'Set %s in profile default', got: %s", tt.key, stderr)
			}
		})
	}
}

func TestProfileSet_TokenSuccess(t *testing.T) {
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
		return []byte("new_mock_token"), nil
	}
	defer func() { readPasswordFunc = oldReadPasswordFunc }()

	// Create a clean command tree for the test
	rootCmd := newRootCmd()
	rootCmd.AddCommand(newProfileCmd())

	// Execute the command
	stdout, stderr, err := testExecuteCommandAndCapture(rootCmd, "--config", configPath, "profile", "set", "token")
	if err != nil {
		t.Fatalf("testExecuteCommandAndCapture returned an error: %v\nStderr: %s", err, stderr)
	}

	// Verify the config file was updated
	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load config after set command: %v", err)
	}
	if cfg.Profiles["default"].Token != "new_mock_token" {
		t.Errorf("Expected token to be 'new_mock_token', got: %s", cfg.Profiles["default"].Token)
	}

	if stdout != "" {
		t.Errorf("Expected no stdout, got: %s", stdout)
	}
	if !strings.Contains(stderr, "Set token in profile default") {
		t.Errorf("Expected stderr to contain 'Set token in profile default', got: %s", stderr)
	}
}

func TestProfileSet_InvalidKey(t *testing.T) {
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
	stdout, stderr, err := testExecuteCommandAndCapture(rootCmd, "--config", configPath, "profile", "set", "non_existent_key", "value")
	if err == nil {
		t.Fatal("Expected an error for invalid key, but got nil")
	}

	// Check the error message
	if !strings.Contains(err.Error(), "unknown configuration key 'non_existent_key'") {
		t.Errorf("Expected error message to contain 'unknown configuration key', got: %v", err)
	}

	if stdout != "" {
		t.Errorf("Expected no stdout, got: %s", stdout)
	}
	if stderr != "" {
		t.Errorf("Expected empty stderr, got: %s", stderr)
	}
}

func TestProfileSet_InvalidProviderValue(t *testing.T) {
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
	stdout, stderr, err := testExecuteCommandAndCapture(rootCmd, "--config", configPath, "profile", "set", "provider", "invalid_provider")
	if err == nil {
		t.Fatal("Expected an error for invalid provider value, but got nil")
	}

	// Check the error message
	if !strings.Contains(err.Error(), "invalid provider 'invalid_provider'. avalid values are 'mock' or 'slack'") {
		t.Errorf("Expected error message to contain 'invalid provider', got: %v", err)
	}

	if stdout != "" {
		t.Errorf("Expected no stdout, got: %s", stdout)
	}
	if stderr != "" {
		t.Errorf("Expected empty stderr, got: %s", stderr)
	}
}

func TestProfileSet_InvalidLimitsValue(t *testing.T) {
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
	stdout, stderr, err := testExecuteCommandAndCapture(rootCmd, "--config", configPath, "profile", "set", "limits.max_file_size_bytes", "abc")
	if err == nil {
		t.Fatal("Expected an error for invalid limits value, but got nil")
	}

	// Check the error message
	if !strings.Contains(err.Error(), "invalid integer value for limits.max_file_size_bytes") {
		t.Errorf("Expected error message to contain 'invalid integer value', got: %v", err)
	}

	if stdout != "" {
		t.Errorf("Expected no stdout, got: %s", stdout)
	}
	if stderr != "" {
		t.Errorf("Expected empty stderr, got: %s", stderr)
	}
}

func TestProfileSet_NoConfig(t *testing.T) {
	// Point to a non-existent config file
	configPath := filepath.Join(t.TempDir(), "non-existent.json")

	// Create a clean command tree for the test
	rootCmd := newRootCmd()
	rootCmd.AddCommand(newProfileCmd())

	// Execute the command
	stdout, stderr, err := testExecuteCommandAndCapture(rootCmd, "--config", configPath, "profile", "set", "provider", "mock")
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

	if stderr != "" {
		t.Errorf("Expected empty stderr, got: %s", stderr)
	}
}
