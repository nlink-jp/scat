// Package config owns bot profiles and environment-only service configuration.
package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
)

type Config struct {
	CurrentProfile string             `json:"current_profile"`
	Profiles       map[string]Profile `json:"profiles"`
}
type Profile struct {
	Channel  string `json:"channel,omitempty"`
	Token    string `json:"token,omitempty"`
	Username string `json:"username,omitempty"`
	Limits   Limits `json:"limits"`
}
type Limits struct {
	MaxFileSizeBytes  int64 `json:"max_file_size_bytes"`
	MaxStdinSizeBytes int64 `json:"max_stdin_size_bytes"`
}

func NewDefaultLimits() Limits { return Limits{1024 * 1024 * 1024, 10 * 1024 * 1024} }
func (l Limits) Validate() error {
	if l.MaxFileSizeBytes < 0 || l.MaxStdinSizeBytes < 0 {
		return errors.New("input limits must be nonnegative; zero means unlimited")
	}
	return nil
}
func (p *Profile) UnmarshalJSON(b []byte) error {
	var keys map[string]json.RawMessage
	if json.Unmarshal(b, &keys) != nil {
		return errors.New("invalid profile JSON")
	}
	for _, key := range []string{"provider", "endpoint"} {
		if _, ok := keys[key]; ok {
			return errors.New("legacy provider/endpoint configuration: back up the file and remove these fields; scat v2 accepts Slack bot profiles only")
		}
	}
	type plain Profile
	v := plain{Limits: NewDefaultLimits()}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if d.Decode(&v) != nil {
		return errors.New("invalid or unknown profile field")
	}
	*p = Profile(v)
	return p.Limits.Validate()
}
func NewDefaultConfig() *Config {
	return &Config{CurrentProfile: "default", Profiles: map[string]Profile{"default": {Limits: NewDefaultLimits()}}}
}

var profileName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{0,63}$`)

func ValidName(s string) bool { return profileName.MatchString(s) }
func (c *Config) Validate() error {
	if c.Profiles == nil {
		return errors.New("configuration requires profiles")
	}
	if _, ok := c.Profiles[c.CurrentProfile]; !ok {
		return errors.New("current_profile does not identify an existing profile")
	}
	for n, p := range c.Profiles {
		if !ValidName(n) {
			return errors.New("invalid profile name")
		}
		if err := p.Limits.Validate(); err != nil {
			return err
		}
	}
	return nil
}
func Load(path string) (*Config, error) {
	// #nosec G304 -- Path is the default config or the operator's explicit --config selection.
	b, e := os.ReadFile(path)
	if e != nil {
		return nil, e
	}
	var c Config
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if e = d.Decode(&c); e != nil {
		// Custom profile errors contain migration guidance; JSON values never enter diagnostics.
		if e.Error() == "legacy provider/endpoint configuration: back up the file and remove these fields; scat v2 accepts Slack bot profiles only" {
			return nil, e
		}
		return nil, errors.New("invalid configuration; check fields and nonnegative input limits")
	}
	if !json.Valid(b) {
		return nil, errors.New("invalid configuration JSON")
	}
	if e = c.Validate(); e != nil {
		return nil, e
	}
	return &c, nil
}
func (c *Config) Save(path string) error {
	if err := c.Validate(); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".scat-config-")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	defer f.Close()
	if _, err = f.Write(append(b, '\n')); err != nil {
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}
func GetConfigPath(override string) (string, error) {
	if override != "" {
		return filepath.Abs(override)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "scat", "config.json"), nil
}
func ServerMode(getenv func(string) string) (bool, error) {
	switch getenv("SCAT_MODE") {
	case "":
		return false, nil
	case "server":
		return true, nil
	default:
		return false, errors.New("SCAT_MODE must be server or unset")
	}
}
func DetectServerMode() (bool, error) { return ServerMode(os.Getenv) }
func FromEnv(getenv func(string) string) (*Config, error) {
	if getenv("SCAT_PROVIDER") != "" {
		return nil, errors.New("SCAT_PROVIDER was removed; unset it for Slack-only scat v2")
	}
	if getenv("SCAT_TOKEN") == "" {
		return nil, errors.New("server mode requires SCAT_TOKEN")
	}
	l := NewDefaultLimits()
	for _, p := range []struct {
		key    string
		target *int64
	}{{"SCAT_MAX_FILE_SIZE", &l.MaxFileSizeBytes}, {"SCAT_MAX_STDIN_SIZE", &l.MaxStdinSizeBytes}} {
		if s := getenv(p.key); s != "" {
			n, e := strconv.ParseInt(s, 10, 64)
			if e != nil || n < 0 {
				return nil, fmt.Errorf("%s must be a nonnegative integer", p.key)
			}
			*p.target = n
		}
	}
	return &Config{CurrentProfile: "server", Profiles: map[string]Profile{"server": {Token: getenv("SCAT_TOKEN"), Channel: getenv("SCAT_CHANNEL"), Username: getenv("SCAT_USERNAME"), Limits: l}}}, nil
}
func BuildConfigFromEnv() (*Config, error) { return FromEnv(os.Getenv) }
