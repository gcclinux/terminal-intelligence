package agentic

import (
	"bytes"
	"context"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

const defaultTestTimeout = 120 * time.Second

// TestRunner executes test commands and captures results.
type TestRunner struct {
	timeout time.Duration
}

// NewTestRunner creates a TestRunner with the default 120s timeout.
func NewTestRunner() *TestRunner {
	return &TestRunner{
		timeout: defaultTestTimeout,
	}
}

// NewTestRunnerWithTimeout creates a TestRunner with a custom timeout.
func NewTestRunnerWithTimeout(timeout time.Duration) *TestRunner {
	if timeout <= 0 {
		timeout = defaultTestTimeout
	}
	return &TestRunner{
		timeout: timeout,
	}
}

// Run executes a test command in the given working directory.
// It enforces a timeout and captures stdout, stderr, exit code, and duration.
func (tr *TestRunner) Run(command string, workDir string) *TestResult {
	command = strings.TrimSpace(command)
	if command == "" {
		return &TestResult{
			ExitCode: 1,
			Stderr:   "empty command",
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), tr.timeout)
	defer cancel()

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "powershell", "-NoProfile", "-Command", command)
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", command)
	}

	if workDir != "" {
		cmd.Dir = workDir
	}

	// Set process group so we can kill the entire tree on timeout.
	setProcGroupAttr(cmd)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now()
	err := cmd.Start()
	if err != nil {
		return &TestResult{
			ExitCode: 1,
			Stderr:   err.Error(),
			Duration: time.Since(start),
		}
	}

	// Wait for completion or timeout in a goroutine.
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	var timedOut bool
	select {
	case <-ctx.Done():
		// Timeout: kill the entire process group.
		timedOut = true
		if cmd.Process != nil {
			killProcessGroup(cmd.Process.Pid)
		}
		<-done // wait for Wait to return after kill
	case err = <-done:
		// Completed normally (success or failure).
	}

	duration := time.Since(start)

	result := &TestResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		Duration: duration,
		TimedOut: timedOut,
	}

	if timedOut {
		result.ExitCode = -1
		return result
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = 1
			if result.Stderr == "" {
				result.Stderr = err.Error()
			}
		}
	}

	return result
}
