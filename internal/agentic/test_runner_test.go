package agentic

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestNewTestRunner(t *testing.T) {
	tr := NewTestRunner()
	if tr == nil {
		t.Fatal("NewTestRunner returned nil")
	}
	if tr.timeout != defaultTestTimeout {
		t.Fatalf("expected timeout %v, got %v", defaultTestTimeout, tr.timeout)
	}
}

func TestNewTestRunnerWithTimeout(t *testing.T) {
	tr := NewTestRunnerWithTimeout(30 * time.Second)
	if tr.timeout != 30*time.Second {
		t.Fatalf("expected 30s timeout, got %v", tr.timeout)
	}
}

func TestNewTestRunnerWithTimeout_ZeroFallsBackToDefault(t *testing.T) {
	tr := NewTestRunnerWithTimeout(0)
	if tr.timeout != defaultTestTimeout {
		t.Fatalf("expected default timeout for zero, got %v", tr.timeout)
	}
}

func TestNewTestRunnerWithTimeout_NegativeFallsBackToDefault(t *testing.T) {
	tr := NewTestRunnerWithTimeout(-5 * time.Second)
	if tr.timeout != defaultTestTimeout {
		t.Fatalf("expected default timeout for negative, got %v", tr.timeout)
	}
}

func TestRunEmptyCommand(t *testing.T) {
	tr := NewTestRunner()
	result := tr.Run("", "")
	if result.ExitCode != 1 {
		t.Fatalf("expected exit code 1 for empty command, got %d", result.ExitCode)
	}
	if result.Stderr != "empty command" {
		t.Fatalf("expected 'empty command' stderr, got %q", result.Stderr)
	}
}

func TestRunSuccessfulCommand(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}
	tr := NewTestRunner()
	result := tr.Run("echo hello", "")
	if result.ExitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr: %s", result.ExitCode, result.Stderr)
	}
	if result.TimedOut {
		t.Fatal("should not have timed out")
	}
	if result.Stdout == "" {
		t.Fatal("expected stdout output")
	}
	if result.Duration <= 0 {
		t.Fatal("expected positive duration")
	}
}

func TestRunFailingCommand(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}
	tr := NewTestRunner()
	result := tr.Run("exit 2", "")
	if result.ExitCode != 2 {
		t.Fatalf("expected exit code 2, got %d", result.ExitCode)
	}
	if result.TimedOut {
		t.Fatal("should not have timed out")
	}
}

func TestRunCapturesStdoutAndStderr(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}
	tr := NewTestRunner()
	result := tr.Run("echo out && echo err >&2", "")
	if result.ExitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", result.ExitCode)
	}
	if result.Stdout == "" {
		t.Fatal("expected stdout")
	}
	if result.Stderr == "" {
		t.Fatal("expected stderr")
	}
}

func TestRunWithWorkDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}
	tmpDir := t.TempDir()
	// Create a file in the temp dir to verify we're running there
	testFile := filepath.Join(tmpDir, "marker.txt")
	if err := os.WriteFile(testFile, []byte("ok"), 0644); err != nil {
		t.Fatal(err)
	}

	tr := NewTestRunner()
	result := tr.Run("cat marker.txt", tmpDir)
	if result.ExitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr: %s", result.ExitCode, result.Stderr)
	}
	if result.Stdout != "ok" {
		t.Fatalf("expected 'ok', got %q", result.Stdout)
	}
}

func TestRunTimeout(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}
	tr := NewTestRunnerWithTimeout(1 * time.Second)
	result := tr.Run("sleep 10", "")
	if !result.TimedOut {
		t.Fatal("expected timeout")
	}
	if result.ExitCode != -1 {
		t.Fatalf("expected exit code -1 for timeout, got %d", result.ExitCode)
	}
}

func TestRunRecordsDuration(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}
	tr := NewTestRunner()
	result := tr.Run("echo fast", "")
	if result.Duration <= 0 {
		t.Fatal("expected positive duration")
	}
}

func TestRunExitCodeZeroIsSuccess(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}
	tr := NewTestRunner()
	result := tr.Run("true", "")
	if result.ExitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", result.ExitCode)
	}
}

func TestRunExitCodeNonZeroIsFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}
	tr := NewTestRunner()
	result := tr.Run("false", "")
	if result.ExitCode == 0 {
		t.Fatal("expected non-zero exit code")
	}
}
