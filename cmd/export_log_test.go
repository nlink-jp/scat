package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "github.com/nlink-jp/scat/internal/provider/testprovider"
)

func TestExportLog_Default(t *testing.T) {
	configPath, cleanup := setupTest(t)
	defer cleanup()

	rootCmd := newRootCmd()
	rootCmd.AddCommand(newExportCmd())

	channel := "#test-channel"

	// Execute the command
	stdout, stderr, err := testExecuteCommandAndCapture(rootCmd, "--config", configPath, "export", "log", "--channel", channel)
	if err != nil {
		t.Fatalf("testExecuteCommandAndCapture returned an error: %v\nStderr: %s", err, stderr)
	}

	// Check if the test provider's ExportLog was called with the correct options
	expectedLog := fmt.Sprintf("ExportLog called with opts: {ChannelName:%s StartTime: EndTime: IncludeFiles:false OutputDir:}", channel)
	if !strings.Contains(stderr, expectedLog) {
		t.Errorf("Expected stderr to contain '%s', got: '%s'", expectedLog, stderr)
	}

	// Check if the output contains the dummy data from the test provider
	if !strings.Contains(stdout, "Test message from ExportLog") {
		t.Errorf("Expected stdout to contain the exported log message, got: %s", stdout)
	}
}

func TestExportLog_WithTimeRange(t *testing.T) {
	configPath, cleanup := setupTest(t)
	defer cleanup()

	rootCmd := newRootCmd()
	rootCmd.AddCommand(newExportCmd())

	channel := "#test-channel"
	startTimeStr := "2023-01-01T10:00:00Z"
	endTimeStr := "2023-01-31T23:59:59Z"

	// Parse times to get expected Unix timestamps
	startTime, _ := time.Parse(time.RFC3339, startTimeStr)
	endTime, _ := time.Parse(time.RFC3339, endTimeStr)
	expectedStartUnix := fmt.Sprintf("%d.000000", startTime.Unix())
	expectedEndUnix := fmt.Sprintf("%d.000000", endTime.Unix())

	// Execute the command
	_, stderr, err := testExecuteCommandAndCapture(rootCmd, "--config", configPath, "export", "log", "--channel", channel, "--start-time", startTimeStr, "--end-time", endTimeStr)
	if err != nil {
		t.Fatalf("testExecuteCommandAndCapture returned an error: %v\nStderr: %s", err, stderr)
	}

	// Check if the test provider's ExportLog was called with the correct options
	expectedLog := fmt.Sprintf("ExportLog called with opts: {ChannelName:%s StartTime:%s EndTime:%s IncludeFiles:false OutputDir:}", channel, expectedStartUnix, expectedEndUnix)
	if !strings.Contains(stderr, expectedLog) {
		t.Errorf("Expected stderr to contain '%s', got: '%s'", expectedLog, stderr)
	}
}

func TestExportLog_WithOutputFile(t *testing.T) {
	configPath, cleanup := setupTest(t)
	defer cleanup()

	tempDir := t.TempDir()
	outputFile := filepath.Join(tempDir, "output.json")

	rootCmd := newRootCmd()
	rootCmd.AddCommand(newExportCmd())

	channel := "#test-channel"

	// Execute the command
	stdout, stderr, err := testExecuteCommandAndCapture(rootCmd, "--config", configPath, "export", "log", "--channel", channel, "--output", outputFile)
	if err != nil {
		t.Fatalf("testExecuteCommandAndCapture returned an error: %v\nStderr: %s", err, stderr)
	}

	// Check if the command reported success to stderr
	if !strings.Contains(stderr, "Log export completed successfully") {
		t.Errorf("Expected stderr to report success, got: %s", stderr)
	}

	// Check that nothing was written to stdout
	if stdout != "" {
		t.Errorf("Expected stdout to be empty, got: %s", stdout)
	}

	// Check the content of the output file
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}
	if !strings.Contains(string(content), "Test message from ExportLog") {
		t.Errorf("Output file does not contain the expected log message, got: %s", string(content))
	}
}

func TestExportLog_WithFileOutput(t *testing.T) {
	configPath, cleanup := setupTest(t)
	defer cleanup()

	tempDir := t.TempDir()
	outputFilesDir := filepath.Join(tempDir, "exported_files")

	rootCmd := newRootCmd()
	rootCmd.AddCommand(newExportCmd())

	channel := "#test-channel"

	// Execute the command
	_, stderr, err := testExecuteCommandAndCapture(rootCmd, "--config", configPath, "export", "log", "--channel", channel, "--output-files", outputFilesDir)
	if err != nil {
		t.Fatalf("testExecuteCommandAndCapture returned an error: %v\nStderr: %s", err, stderr)
	}

	// Check if the test provider's ExportLog was called with the correct options
	expectedLog := fmt.Sprintf("ExportLog called with opts: {ChannelName:%s StartTime: EndTime: IncludeFiles:true OutputDir:%s}", channel, outputFilesDir)
	if !strings.Contains(stderr, expectedLog) {
		t.Errorf("Expected stderr to contain '%s', got: '%s'", expectedLog, stderr)
	}

	// Check if the output directory was created
	if _, err := os.Stat(outputFilesDir); os.IsNotExist(err) {
		t.Errorf("Expected output directory '%s' to be created, but it was not", outputFilesDir)
	}
}