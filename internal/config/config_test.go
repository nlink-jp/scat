
package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestNewDefaultConfig(t *testing.T) {
	cfg := NewDefaultConfig()

	if cfg.CurrentProfile != "default" {
		t.Errorf("expected current profile to be 'default', got '%s'", cfg.CurrentProfile)
	}
	if len(cfg.Profiles) != 1 {
		t.Fatalf("expected 1 profile, got %d", len(cfg.Profiles))
	}
	if _, ok := cfg.Profiles["default"]; !ok {
		t.Fatal("expected 'default' profile to exist")
	}
	defaultProfile := cfg.Profiles["default"]
	if defaultProfile.Provider != "mock" {
		t.Errorf("expected default provider to be 'mock', got '%s'", defaultProfile.Provider)
	}
}

func TestSaveLoad(t *testing.T) {
	tempDir := t.TempDir()
	testPath := filepath.Join(tempDir, "config.json")

	// 1. Create a config, save it
	cfg1 := NewDefaultConfig()
	err := cfg1.Save(testPath)
	if err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// 2. Load it back
	cfg2, err := Load(testPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// 3. Compare
	if !reflect.DeepEqual(cfg1, cfg2) {
		t.Errorf("loaded config does not match saved config")
		t.Logf("Saved: %+v", cfg1)
		t.Logf("Loaded: %+v", cfg2)
	}
}

func TestLoad_FileNotExists(t *testing.T) {
	tempDir := t.TempDir()
	testPath := filepath.Join(tempDir, "non_existent_config.json")

	_, err := Load(testPath)
	if err == nil {
		t.Fatal("expected an error when loading non-existent file, but got nil")
	}
}

func TestLoad_InvalidJson(t *testing.T) {
	tempDir := t.TempDir()
	testPath := filepath.Join(tempDir, "invalid.json")

	err := os.WriteFile(testPath, []byte("{ not a valid json }"), 0600)
	if err != nil {
		t.Fatalf("failed to write invalid json file: %v", err)
	}

	_, err = Load(testPath)
	if err == nil {
		t.Fatal("expected an error when loading invalid json, but got nil")
	}
}

func TestBuildConfigFromEnv(t *testing.T) {
	defaults := NewDefaultLimits()

	tests := []struct {
		name    string
		env     map[string]string
		wantErr bool
		check   func(*testing.T, *Config)
	}{
		{
			name:    "missing required vars",
			env:     map[string]string{},
			wantErr: true,
		},
		{
			name:    "missing SCAT_TOKEN",
			env:     map[string]string{"SCAT_PROVIDER": "slack"},
			wantErr: true,
		},
		{
			name: "minimal valid",
			env:  map[string]string{"SCAT_PROVIDER": "slack", "SCAT_TOKEN": "xoxb-test"},
			check: func(t *testing.T, cfg *Config) {
				p := cfg.Profiles["server"]
				if p.Provider != "slack" {
					t.Errorf("provider = %q, want slack", p.Provider)
				}
				if p.Token != "xoxb-test" {
					t.Errorf("token = %q, want xoxb-test", p.Token)
				}
				if p.Limits.MaxFileSizeBytes != defaults.MaxFileSizeBytes {
					t.Errorf("MaxFileSizeBytes = %d, want %d", p.Limits.MaxFileSizeBytes, defaults.MaxFileSizeBytes)
				}
				if p.Limits.MaxStdinSizeBytes != defaults.MaxStdinSizeBytes {
					t.Errorf("MaxStdinSizeBytes = %d, want %d", p.Limits.MaxStdinSizeBytes, defaults.MaxStdinSizeBytes)
				}
			},
		},
		{
			name: "custom limits",
			env: map[string]string{
				"SCAT_PROVIDER":      "slack",
				"SCAT_TOKEN":         "xoxb-test",
				"SCAT_MAX_FILE_SIZE": "5242880",
				"SCAT_MAX_STDIN_SIZE": "1048576",
			},
			check: func(t *testing.T, cfg *Config) {
				p := cfg.Profiles["server"]
				if p.Limits.MaxFileSizeBytes != 5242880 {
					t.Errorf("MaxFileSizeBytes = %d, want 5242880", p.Limits.MaxFileSizeBytes)
				}
				if p.Limits.MaxStdinSizeBytes != 1048576 {
					t.Errorf("MaxStdinSizeBytes = %d, want 1048576", p.Limits.MaxStdinSizeBytes)
				}
			},
		},
		{
			name:    "invalid SCAT_MAX_FILE_SIZE",
			env:     map[string]string{"SCAT_PROVIDER": "slack", "SCAT_TOKEN": "t", "SCAT_MAX_FILE_SIZE": "abc"},
			wantErr: true,
		},
		{
			name:    "negative SCAT_MAX_STDIN_SIZE",
			env:     map[string]string{"SCAT_PROVIDER": "slack", "SCAT_TOKEN": "t", "SCAT_MAX_STDIN_SIZE": "-1"},
			wantErr: true,
		},
		{
			name: "optional fields",
			env: map[string]string{
				"SCAT_PROVIDER": "slack",
				"SCAT_TOKEN":    "xoxb-test",
				"SCAT_CHANNEL":  "#alerts",
				"SCAT_USERNAME": "bot",
			},
			check: func(t *testing.T, cfg *Config) {
				p := cfg.Profiles["server"]
				if p.Channel != "#alerts" {
					t.Errorf("channel = %q, want #alerts", p.Channel)
				}
				if p.Username != "bot" {
					t.Errorf("username = %q, want bot", p.Username)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Clear relevant env vars, set test values
			for _, k := range []string{"SCAT_PROVIDER", "SCAT_TOKEN", "SCAT_CHANNEL", "SCAT_USERNAME", "SCAT_MAX_FILE_SIZE", "SCAT_MAX_STDIN_SIZE"} {
				t.Setenv(k, "")
			}
			for k, v := range tc.env {
				t.Setenv(k, v)
			}

			cfg, err := BuildConfigFromEnv()
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.check != nil {
				tc.check(t, cfg)
			}
		})
	}
}

func TestProfileManagement_DirectManipulation(t *testing.T) {
	cfg := NewDefaultConfig()

	// Add a new profile by direct map manipulation
	newProfile := Profile{
		Provider: "slack",
		Channel:  "#general",
		Limits:   NewDefaultLimits(),
	}
	cfg.Profiles["my-slack"] = newProfile

	if len(cfg.Profiles) != 2 {
		t.Errorf("expected 2 profiles after adding, got %d", len(cfg.Profiles))
	}
	if !reflect.DeepEqual(cfg.Profiles["my-slack"], newProfile) {
		t.Error("added profile does not match original")
	}

	// Set current profile by direct field manipulation
	cfg.CurrentProfile = "my-slack"
	if cfg.CurrentProfile != "my-slack" {
		t.Errorf("expected current profile to be 'my-slack', got '%s'", cfg.CurrentProfile)
	}

	// Remove profile by using delete
	delete(cfg.Profiles, "my-slack")
	if len(cfg.Profiles) != 1 {
		t.Errorf("expected 1 profile after removing, got %d", len(cfg.Profiles))
	}
	if _, ok := cfg.Profiles["my-slack"]; ok {
		t.Error("removed profile 'my-slack' still exists")
	}
}
