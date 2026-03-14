package agentic

import (
	"regexp"
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// Feature: project-wide-agentic-fixer, Property 10: All actions logged via DisplayNotification
// **Validates: Requirements 6.7**
func TestProperty10_AllActionsLoggedViaDisplayNotification(t *testing.T) {
	timestampPattern := regexp.MustCompile(`^\[\d{2}:\d{2}:\d{2}\]`)

	rapid.Check(t, func(t *rapid.T) {
		// Generate a random log message.
		msg := rapid.StringMatching(`[a-zA-Z0-9 _.,:;!?/\\-]{1,200}`).Draw(t, "logMessage")

		// Track how many times notify is called and what it receives.
		callCount := 0
		var captured string
		logger := NewActionLogger(func(s string) {
			callCount++
			captured = s
		})

		logger.Log("%s", msg)

		// Verify: notify is invoked exactly once per Log() call.
		if callCount != 1 {
			t.Fatalf("expected notify called exactly once, got %d", callCount)
		}

		// Verify: the output starts with a timestamp in "[HH:MM:SS]" format.
		if !timestampPattern.MatchString(captured) {
			t.Fatalf("output does not start with [HH:MM:SS] timestamp: %q", captured)
		}

		// Verify: the output contains the original message text.
		if !strings.Contains(captured, msg) {
			t.Fatalf("output does not contain original message %q: got %q", msg, captured)
		}
	})
}
