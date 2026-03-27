package cmd

import (
	"strings"
	"testing"

	_ "github.com/nlink-jp/scat/internal/provider/testprovider"
)

func TestChannelCreateCmd(t *testing.T) {
	configPath, cleanup := setupTest(t)
	defer cleanup()

	tests := []struct {
		name       string
		args       []string
		wantErr    bool
		wantOutput string
	}{
		{
			name:       "success",
			args:       []string{"--config", configPath, "channel", "create", "new-channel"},
			wantErr:    false,
			wantOutput: "info: Successfully created channel with ID: C1234567890",
		},
		{
			name:       "no channel name",
			args:       []string{"--config", configPath, "channel", "create"},
			wantErr:    true,
			wantOutput: "accepts 1 arg(s), received 0",
		},
		{
			name: "success with all options",
			args: []string{
				"--config", configPath,
				"channel", "create", "another-channel",
				"--description", "A new channel",
				"--topic", "Channel topic",
				"--private",
				"--invite", "user1,user2",
			},
			wantErr:    false,
			wantOutput: "info: Successfully created channel with ID: C1234567890",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootCmd := newRootCmd()
			rootCmd.AddCommand(newChannelCmd())

			stdout, _, err := testExecuteCommandAndCapture(rootCmd, tt.args...)

			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				if !strings.Contains(err.Error(), tt.wantOutput) {
					t.Errorf("Execute() error = %v, wantOutput %v", err.Error(), tt.wantOutput)
				}
			} else {
				if !strings.Contains(stdout, tt.wantOutput) {
					t.Errorf("Execute() stdout = %v, wantOutput %v", stdout, tt.wantOutput)
				}
			}
		})
	}
}
