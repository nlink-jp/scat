package appcontext

import (
	"reflect"
	"testing"
)

func TestNewContext(t *testing.T) {
	tests := []struct {
		name       string
		debug      bool
		noOp       bool
		silent     bool
		configPath string
		want       Context
	}{
		{
			name:       "All false, empty path",
			debug:      false,
			noOp:       false,
			silent:     false,
			configPath: "",
			want: Context{
				Debug:      false,
				NoOp:       false,
				Silent:     false,
				ConfigPath: "",
				ServerMode: false,
				Config:     nil,
			},
		},
		{
			name:       "All true, with path",
			debug:      true,
			noOp:       true,
			silent:     true,
			configPath: "/tmp/config.json",
			want: Context{
				Debug:      true,
				NoOp:       true,
				Silent:     true,
				ConfigPath: "/tmp/config.json",
				ServerMode: false,
				Config:     nil,
			},
		},
		{
			name:       "Mixed values",
			debug:      true,
			noOp:       false,
			silent:     true,
			configPath: "/path/to/config",
			want: Context{
				Debug:      true,
				NoOp:       false,
				Silent:     true,
				ConfigPath: "/path/to/config",
				ServerMode: false,
				Config:     nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewContext(tt.debug, tt.noOp, tt.silent, tt.configPath, false, nil)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewContext() = %v, want %v", got, tt.want)
			}
		})
	}
}
