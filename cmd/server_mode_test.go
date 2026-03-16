package cmd

import (
	"strings"
	"testing"
)

// setServerModeEnv sets the minimum env vars required for server mode and registers
// cleanup via t.Cleanup so they are unset after the test.
func setServerModeEnv(t *testing.T, extra map[string]string) {
	t.Helper()
	t.Setenv("SCAT_MODE", "server")
	t.Setenv("SCAT_PROVIDER", "test")
	t.Setenv("SCAT_TOKEN", "test-token")
	// Clear optional vars so previous test runs don't bleed through.
	for _, k := range []string{"SCAT_CHANNEL", "SCAT_USERNAME", "SCAT_MAX_FILE_SIZE", "SCAT_MAX_STDIN_SIZE"} {
		t.Setenv(k, "")
	}
	for k, v := range extra {
		t.Setenv(k, v)
	}
}

// --- SCAT_MODE validation ---

func TestServerMode_InvalidMode(t *testing.T) {
	t.Setenv("SCAT_MODE", "invalid")

	rootCmd := newRootCmd()
	rootCmd.AddCommand(newPostCmd())

	_, _, err := testExecuteCommandAndCapture(rootCmd, "post", "hello")
	if err == nil {
		t.Fatal("expected error for invalid SCAT_MODE, got nil")
	}
	if !strings.Contains(err.Error(), "invalid SCAT_MODE") {
		t.Errorf("unexpected error: %v", err)
	}
}

// --- post in server mode ---

func TestServerMode_Post_Success(t *testing.T) {
	setServerModeEnv(t, nil)

	rootCmd := newRootCmd()
	rootCmd.AddCommand(newPostCmd())

	_, stderr, err := testExecuteCommandAndCapture(rootCmd, "post", "hello from server mode")
	if err != nil {
		t.Fatalf("unexpected error: %v\nstderr: %s", err, stderr)
	}
	if !strings.Contains(stderr, "PostMessage called") {
		t.Errorf("expected PostMessage to be called, stderr: %s", stderr)
	}
}

func TestServerMode_Post_MissingRequiredEnvVars(t *testing.T) {
	t.Setenv("SCAT_MODE", "server")
	t.Setenv("SCAT_PROVIDER", "")
	t.Setenv("SCAT_TOKEN", "")

	rootCmd := newRootCmd()
	rootCmd.AddCommand(newPostCmd())

	_, _, err := testExecuteCommandAndCapture(rootCmd, "post", "hello")
	if err == nil {
		t.Fatal("expected error when required env vars are missing, got nil")
	}
	if !strings.Contains(err.Error(), "server mode requires environment variables") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestServerMode_Post_ConfigFlagIsError(t *testing.T) {
	setServerModeEnv(t, nil)

	rootCmd := newRootCmd()
	rootCmd.AddCommand(newPostCmd())

	_, _, err := testExecuteCommandAndCapture(rootCmd, "--config", "/some/path.json", "post", "hello")
	if err == nil {
		t.Fatal("expected error when --config is used in server mode, got nil")
	}
	if !strings.Contains(err.Error(), "--config flag cannot be used in server mode") {
		t.Errorf("unexpected error: %v", err)
	}
}

// --- profile/config commands blocked in server mode ---

func TestServerMode_ProfileList_IsError(t *testing.T) {
	setServerModeEnv(t, nil)

	rootCmd := newRootCmd()
	rootCmd.AddCommand(newProfileCmd())

	_, _, err := testExecuteCommandAndCapture(rootCmd, "profile", "list")
	if err == nil {
		t.Fatal("expected error for profile list in server mode, got nil")
	}
	if !strings.Contains(err.Error(), "not available in server mode") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestServerMode_ProfileAdd_IsError(t *testing.T) {
	setServerModeEnv(t, nil)

	rootCmd := newRootCmd()
	rootCmd.AddCommand(newProfileCmd())

	_, _, err := testExecuteCommandAndCapture(rootCmd, "profile", "add", "new-profile", "--provider", "test")
	if err == nil {
		t.Fatal("expected error for profile add in server mode, got nil")
	}
	if !strings.Contains(err.Error(), "not available in server mode") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestServerMode_ConfigInit_IsError(t *testing.T) {
	setServerModeEnv(t, nil)

	rootCmd := newRootCmd()
	rootCmd.AddCommand(newConfigCmd())

	_, _, err := testExecuteCommandAndCapture(rootCmd, "config", "init")
	if err == nil {
		t.Fatal("expected error for config init in server mode, got nil")
	}
	if !strings.Contains(err.Error(), "not available in server mode") {
		t.Errorf("unexpected error: %v", err)
	}
}

// --- limits via env vars ---

func TestServerMode_CustomLimits_Applied(t *testing.T) {
	setServerModeEnv(t, map[string]string{
		"SCAT_MAX_FILE_SIZE":  "1024",
		"SCAT_MAX_STDIN_SIZE": "512",
	})

	rootCmd := newRootCmd()
	rootCmd.AddCommand(newPostCmd())

	// A short message is within limits — should succeed.
	_, _, err := testExecuteCommandAndCapture(rootCmd, "post", "short message")
	if err != nil {
		t.Fatalf("unexpected error with custom limits: %v", err)
	}
}

func TestServerMode_InvalidMaxFileSize_IsError(t *testing.T) {
	setServerModeEnv(t, map[string]string{
		"SCAT_MAX_FILE_SIZE": "not-a-number",
	})

	rootCmd := newRootCmd()
	rootCmd.AddCommand(newPostCmd())

	_, _, err := testExecuteCommandAndCapture(rootCmd, "post", "hello")
	if err == nil {
		t.Fatal("expected error for invalid SCAT_MAX_FILE_SIZE, got nil")
	}
	if !strings.Contains(err.Error(), "invalid SCAT_MAX_FILE_SIZE") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestServerMode_InvalidMaxStdinSize_IsError(t *testing.T) {
	setServerModeEnv(t, map[string]string{
		"SCAT_MAX_STDIN_SIZE": "-100",
	})

	rootCmd := newRootCmd()
	rootCmd.AddCommand(newPostCmd())

	_, _, err := testExecuteCommandAndCapture(rootCmd, "post", "hello")
	if err == nil {
		t.Fatal("expected error for invalid SCAT_MAX_STDIN_SIZE, got nil")
	}
	if !strings.Contains(err.Error(), "invalid SCAT_MAX_STDIN_SIZE") {
		t.Errorf("unexpected error: %v", err)
	}
}
