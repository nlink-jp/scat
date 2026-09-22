package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestV2ConfigDefaultsZeroAndMigration(t *testing.T) {
	for _, limits := range []Limits{NewDefaultLimits(), {0, 0}, {100, 20}} {
		c := NewDefaultConfig()
		p := c.Profiles["default"]
		p.Limits = limits
		c.Profiles["default"] = p
		path := filepath.Join(t.TempDir(), "sub", "config.json")
		if e := c.Save(path); e != nil {
			t.Fatal(e)
		}
		got, e := Load(path)
		if e != nil || !reflect.DeepEqual(got, c) {
			t.Fatal(got, e)
		}
		info, _ := os.Stat(path)
		if info.Mode().Perm() != 0600 {
			t.Fatal(info.Mode())
		}
	}
	for _, body := range []string{`{"provider":"slack"}`, `{"provider":"mock"}`, `{"endpoint":null}`, `{"limits":{"max_file_size_bytes":-1}}`, `{"unknown":"value"}`} {
		path := filepath.Join(t.TempDir(), "c.json")
		os.WriteFile(path, []byte(`{"current_profile":"default","profiles":{"default":`+body+`}}`), 0600)
		if _, e := Load(path); e == nil {
			t.Fatal(body)
		}
	}
	path := filepath.Join(t.TempDir(), "c.json")
	os.WriteFile(path, []byte(`{"current_profile":"default","profiles":{"default":{"limits":{"max_file_size_bytes":0}}}}`), 0600)
	c, e := Load(path)
	if e != nil || c.Profiles["default"].Limits.MaxFileSizeBytes != 0 || c.Profiles["default"].Limits.MaxStdinSizeBytes != 10*1024*1024 {
		t.Fatal(c, e)
	}
}
func TestV2Environment(t *testing.T) {
	for _, env := range []map[string]string{{}, {"SCAT_TOKEN": "fixture", "SCAT_PROVIDER": "slack"}, {"SCAT_TOKEN": "fixture", "SCAT_MAX_FILE_SIZE": "-1"}, {"SCAT_TOKEN": "fixture", "SCAT_MAX_STDIN_SIZE": "bad"}} {
		if _, e := FromEnv(func(k string) string { return env[k] }); e == nil {
			t.Fatal(env)
		}
	}
	env := map[string]string{"SCAT_TOKEN": "fixture", "SCAT_CHANNEL": "C1", "SCAT_MAX_FILE_SIZE": "0"}
	c, e := FromEnv(func(k string) string { return env[k] })
	if e != nil || c.Profiles["server"].Limits.MaxFileSizeBytes != 0 {
		t.Fatal(c, e)
	}
}
func TestV2MissingAndInvalidConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if _, e := Load(path); !os.IsNotExist(e) {
		t.Fatal(e)
	}
	for _, s := range []string{"{bad", `{"profiles":{}}`, `{"current_profile":"missing","profiles":{"default":{}}}`} {
		os.WriteFile(path, []byte(s), 0600)
		if _, e := Load(path); e == nil {
			t.Fatal(s)
		}
	}
}
