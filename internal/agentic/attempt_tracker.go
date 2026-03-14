package agentic

import (
	"fmt"
	"strings"
)

// AttemptTracker tracks what has been tried during a fix session
// to prevent repetition of failed approaches.
type AttemptTracker struct {
	attempts []FixAttempt
}

// NewAttemptTracker creates an empty AttemptTracker.
func NewAttemptTracker() *AttemptTracker {
	return &AttemptTracker{
		attempts: []FixAttempt{},
	}
}

// Record adds a completed attempt to the tracker.
func (at *AttemptTracker) Record(attempt FixAttempt) {
	at.attempts = append(at.attempts, attempt)
}

// AttemptCount returns the total number of recorded attempts.
func (at *AttemptTracker) AttemptCount() int {
	return len(at.attempts)
}

// GenerateSummary produces a text summary of all prior attempts
// suitable for inclusion in an AI prompt.
func (at *AttemptTracker) GenerateSummary() string {
	if len(at.attempts) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("Prior attempts:\n")

	for _, a := range at.attempts {
		sb.WriteString(fmt.Sprintf("\nAttempt %d (cycle %d):\n", a.Number, a.Cycle))
		sb.WriteString(fmt.Sprintf("  Strategy: %s\n", a.Strategy.Description))

		if len(a.FilesModified) > 0 {
			sb.WriteString("  Files modified:\n")
			for _, f := range a.FilesModified {
				sb.WriteString(fmt.Sprintf("    - %s\n", f.Path))
			}
		}

		if a.TestResult != nil {
			if a.TestResult.ExitCode == 0 {
				sb.WriteString("  Test result: PASS (exit code 0)\n")
			} else {
				sb.WriteString(fmt.Sprintf("  Test result: FAIL (exit code %d)\n", a.TestResult.ExitCode))
			}
		}

		sb.WriteString(fmt.Sprintf("  Outcome: %s\n", attemptOutcome(a)))
	}

	return sb.String()
}

// attemptOutcome returns a short description of the attempt's outcome.
func attemptOutcome(a FixAttempt) string {
	if a.TestResult == nil {
		return "no tests run"
	}
	if a.TestResult.TimedOut {
		return "test timed out"
	}
	if a.TestResult.ExitCode == 0 {
		return "tests passed"
	}
	return "tests failed"
}
