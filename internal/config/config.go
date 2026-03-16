package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const modeEnvVar = "SCAT_MODE"

// DetectServerMode returns true when SCAT_MODE=server is set.
// Returns an error if SCAT_MODE is set to an unrecognized value.
func DetectServerMode() (bool, error) {
	val := os.Getenv(modeEnvVar)
	switch val {
	case "":
		return false, nil
	case "server":
		return true, nil
	default:
		return false, fmt.Errorf("invalid SCAT_MODE value %q: must be \"server\" or unset", val)
	}
}

// BuildConfigFromEnv constructs a virtual Config from environment variables for server mode.
// SCAT_PROVIDER and SCAT_TOKEN are required; SCAT_CHANNEL, SCAT_USERNAME,
// SCAT_MAX_FILE_SIZE, and SCAT_MAX_STDIN_SIZE are optional.
// The resulting Config has a single profile named "server".
func BuildConfigFromEnv() (*Config, error) {
	p := os.Getenv("SCAT_PROVIDER")
	t := os.Getenv("SCAT_TOKEN")

	var missing []string
	if p == "" {
		missing = append(missing, "SCAT_PROVIDER")
	}
	if t == "" {
		missing = append(missing, "SCAT_TOKEN")
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("server mode requires environment variables: %s", strings.Join(missing, ", "))
	}

	limits, err := limitsFromEnv()
	if err != nil {
		return nil, err
	}

	return &Config{
		CurrentProfile: "server",
		Profiles: map[string]Profile{
			"server": {
				Provider: p,
				Token:    t,
				Channel:  os.Getenv("SCAT_CHANNEL"),
				Username: os.Getenv("SCAT_USERNAME"),
				Limits:   limits,
			},
		},
	}, nil
}

// limitsFromEnv reads SCAT_MAX_FILE_SIZE and SCAT_MAX_STDIN_SIZE, falling back to defaults.
func limitsFromEnv() (Limits, error) {
	defaults := NewDefaultLimits()
	limits := defaults

	if v := os.Getenv("SCAT_MAX_FILE_SIZE"); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil || n < 0 {
			return Limits{}, fmt.Errorf("invalid SCAT_MAX_FILE_SIZE value %q: must be a non-negative integer (bytes)", v)
		}
		limits.MaxFileSizeBytes = n
	}
	if v := os.Getenv("SCAT_MAX_STDIN_SIZE"); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil || n < 0 {
			return Limits{}, fmt.Errorf("invalid SCAT_MAX_STDIN_SIZE value %q: must be a non-negative integer (bytes)", v)
		}
		limits.MaxStdinSizeBytes = n
	}
	return limits, nil
}

const (
	configDir  = ".config"
	configFile = "scat/config.json"
)

// Config represents the overall structure of the application's configuration file.
type Config struct {
	CurrentProfile string             `json:"current_profile"`
	Profiles       map[string]Profile `json:"profiles"`
}

// Profile defines the settings for a specific destination endpoint.
type Profile struct {
	Provider string `json:"provider,omitempty"` // "mock" or "slack"
	Endpoint string `json:"endpoint,omitempty"` // Used by "generic" provider (deprecated)
	Channel  string `json:"channel,omitempty"`  // Used by "slack" provider
	Token    string `json:"token,omitempty"`
	Username string `json:"username,omitempty"`
	Limits   Limits `json:"limits"`
}

// Limits defines the size limits for inputs.
type Limits struct {
	MaxFileSizeBytes int64 `json:"max_file_size_bytes,omitempty"`
	MaxStdinSizeBytes int64 `json:"max_stdin_size_bytes,omitempty"`
}

// NewDefaultLimits returns a Limits struct with default values.
func NewDefaultLimits() Limits {
	return Limits{
		MaxFileSizeBytes: 1024 * 1024 * 1024, // 1 GB
		MaxStdinSizeBytes: 10 * 1024 * 1024,  // 10 MB
	}
}

// NewDefaultConfig creates a new Config object with default settings.
func NewDefaultConfig() *Config {
	return &Config{
		CurrentProfile: "default",
		Profiles: map[string]Profile{
			"default": {
				Provider: "mock",
				Channel:  "#mock-channel",
				Limits:   NewDefaultLimits(),
			},
		},
	}
}

// Load reads the configuration file from the user's config directory.
func Load(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err // Return error if file doesn't exist or other read error
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	// For backward compatibility, populate limits if they are not set.
	for name, profile := range cfg.Profiles {
		if profile.Limits.MaxFileSizeBytes == 0 && profile.Limits.MaxStdinSizeBytes == 0 {
			profile.Limits = NewDefaultLimits()
			cfg.Profiles[name] = profile
		}
	}

	return &cfg, nil
}

// Save writes the current configuration to the user's config directory.
func (c *Config) Save(configPath string) error {
	if err := os.MkdirAll(filepath.Dir(configPath), 0700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0600)
}

// GetConfigPath returns the absolute path to the configuration file.
func GetConfigPath(overridePath string) (string, error) {
	if overridePath != "" {
		return overridePath, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, configDir, configFile), nil
}
