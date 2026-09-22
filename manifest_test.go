package main

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
)

// The installable template is a permission boundary: cover the bot CLI's
// operations without introducing user credentials, callbacks or token refresh.
func TestSlackAppManifest(t *testing.T) {
	data, err := os.ReadFile("slack-app-manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	var m struct {
		Display struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		} `json:"display_information"`
		Features struct {
			Bot struct {
				Name   string `json:"display_name"`
				Online *bool  `json:"always_online"`
			} `json:"bot_user"`
		} `json:"features"`
		OAuth struct {
			Scopes struct {
				Bot []string `json:"bot"`
			} `json:"scopes"`
		} `json:"oauth_config"`
		Settings struct {
			Org      *bool `json:"org_deploy_enabled"`
			Socket   *bool `json:"socket_mode_enabled"`
			Rotation *bool `json:"token_rotation_enabled"`
		} `json:"settings"`
	}
	strict := func(b []byte, out any) {
		t.Helper()
		d := json.NewDecoder(bytes.NewReader(b))
		d.DisallowUnknownFields()
		if err := d.Decode(out); err != nil {
			t.Fatal(err)
		}
		if !json.Valid(b) {
			t.Fatal("invalid manifest JSON")
		}
	}
	strict(data, &m)
	if m.Display.Name != "scat" || len(m.Display.Description) > 140 || m.Display.Description == "" || m.Features.Bot.Name != "scat" {
		t.Fatal("invalid app/bot identity")
	}
	for _, flag := range []*bool{m.Features.Bot.Online, m.Settings.Org, m.Settings.Socket, m.Settings.Rotation} {
		if flag == nil || *flag {
			t.Fatal("unsupported app mode enabled or default left implicit")
		}
	}
	// These groups express real CLI capability requirements. Any new permission
	// must be justified by a feature, and a missing one breaks one-click setup.
	capabilities := map[string][]string{
		"post and custom identity":                   {"chat:write", "chat:write.customize"},
		"file round trips":                           {"files:read", "files:write"},
		"public channel export and management":       {"channels:read", "channels:history", "channels:join", "channels:manage"},
		"private channel export and management":      {"groups:read", "groups:history", "groups:write"},
		"DM delivery and export":                     {"im:write", "im:read", "im:history"},
		"existing group DM export":                   {"mpim:read", "mpim:history"},
		"user/group lookup and invitation expansion": {"users:read", "usergroups:read"},
	}
	expected := map[string]bool{}
	for _, scopes := range capabilities {
		for _, s := range scopes {
			expected[s] = true
		}
	}
	seen := map[string]bool{}
	for _, s := range m.OAuth.Scopes.Bot {
		if seen[s] {
			t.Errorf("duplicate scope %s", s)
		}
		seen[s] = true
		if !expected[s] {
			t.Errorf("scope has no supported bot capability: %s", s)
		}
	}
	for capability, scopes := range capabilities {
		for _, s := range scopes {
			if !seen[s] {
				t.Errorf("%s missing scope %s", capability, s)
			}
		}
	}
}
