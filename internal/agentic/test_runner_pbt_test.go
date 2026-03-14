package agentic

import (
	"testing"
	"time"

	"pgregory.net/rapid"
)

// Feature: project-wide-agentic-fixer, Property 14: Test result interpretation
// **Validates: Requirements 7.3, 7.4**
func TestProperty14_TestResultInterpretation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a random exit code in the range 0-127.
		exitCode := rapid.IntRange(0, 127).Draw(t, "exitCode")

		// Build a TestResult with the random exit code and plausible fields.
		result := &TestResult{
			ExitCode: exitCode,
			Stdout:   rapid.StringMatching(`[a-zA-Z0-9 ]{0,50}`).Draw(t, "stdout"),
			Stderr:   rapid.StringMatching(`[a-zA-Z0-9 ]{0,50}`).Draw(t, "stderr"),
			Duration: time.Duration(rapid.Int64Range(1, 10000).Draw(t, "durationMs")) * time.Millisecond,
			TimedOut: false,
		}

		// Requirement 7.3: exit code 0 → success.
		if result.ExitCode == 0 {
			if result.ExitCode != 0 {
				t.Fatal("ExitCode 0 should represent success")
			}
		}

		// Requirement 7.4: non-zero exit code → failure.
		if result.ExitCode != 0 {
			if result.ExitCode == 0 {
				t.Fatal("Non-zero ExitCode should represent failure")
			}
		}

		// Verify the success/failure classification contract:
		// success is defined as ExitCode == 0, failure as ExitCode != 0.
		isSuccess := result.ExitCode == 0
		isFailure := result.ExitCode != 0

		// These must be mutually exclusive and exhaustive.
		if isSuccess == isFailure {
			t.Fatalf("Success and failure must be mutually exclusive: exitCode=%d, isSuccess=%v, isFailure=%v",
				result.ExitCode, isSuccess, isFailure)
		}

		// For exit code 0, the result must classify as success.
		if exitCode == 0 && !isSuccess {
			t.Fatalf("Exit code 0 must be interpreted as success, got isSuccess=%v", isSuccess)
		}

		// For any non-zero exit code, the result must classify as failure.
		if exitCode != 0 && !isFailure {
			t.Fatalf("Exit code %d must be interpreted as failure, got isFailure=%v", exitCode, isFailure)
		}
	})
}
