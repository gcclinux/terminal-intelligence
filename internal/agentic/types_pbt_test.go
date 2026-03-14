package agentic

import (
	"testing"
	"time"

	"pgregory.net/rapid"
)

// Feature: project-wide-agentic-fixer, Property 18: Fix session data structure completeness
// **Validates: Requirements 11.1, 11.2, 11.3**
func TestProperty18_FixSessionDataStructureCompleteness(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a valid FixSession with random but valid inputs.
		originalAsk := rapid.StringMatching(`[a-zA-Z][a-zA-Z0-9 ]{0,99}`).Draw(t, "originalAsk")
		startTime := time.Unix(rapid.Int64Range(1, 4102444800).Draw(t, "startTimeUnix"), 0)

		// Generate random snapshots map (0-5 entries).
		numSnapshots := rapid.IntRange(0, 5).Draw(t, "numSnapshots")
		snapshots := make(map[string][]byte)
		for i := 0; i < numSnapshots; i++ {
			key := rapid.StringMatching(`[a-z]{1,10}/[a-z]{1,10}\.(go|py|sh)`).Draw(t, "snapshotKey")
			val := []byte(rapid.StringMatching(`[a-zA-Z0-9 \n]{0,200}`).Draw(t, "snapshotVal"))
			snapshots[key] = val
		}

		// Generate random FixAttempts (1-5).
		numAttempts := rapid.IntRange(1, 5).Draw(t, "numAttempts")
		attempts := make([]FixAttempt, numAttempts)
		for i := 0; i < numAttempts; i++ {
			stratDesc := rapid.StringMatching(`[a-zA-Z][a-zA-Z0-9 ]{0,49}`).Draw(t, "stratDesc")
			stratPrompt := rapid.StringMatching(`[a-zA-Z][a-zA-Z0-9 ]{0,49}`).Draw(t, "stratPrompt")
			attemptTS := time.Unix(rapid.Int64Range(1, 4102444800).Draw(t, "attemptTS"), 0)

			attempts[i] = FixAttempt{
				Number:    i + 1,
				Cycle:     rapid.IntRange(0, 2).Draw(t, "cycle"),
				Strategy: Strategy{
					Description: stratDesc,
					Prompt:      stratPrompt,
					AIResponse:  rapid.StringMatching(`[a-zA-Z0-9 ]{0,100}`).Draw(t, "aiResponse"),
				},
				Timestamp: attemptTS,
			}
		}

		session := FixSession{
			OriginalAsk: originalAsk,
			StartTime:   startTime,
			Attempts:    attempts,
			Snapshots:   snapshots,
		}

		// Requirement 11.1: FixSession contains original ask, start timestamp, attempts, snapshots.
		if session.OriginalAsk == "" {
			t.Fatal("OriginalAsk must not be empty")
		}
		if session.StartTime.IsZero() {
			t.Fatal("StartTime must not be zero")
		}
		if session.Snapshots == nil {
			t.Fatal("Snapshots must not be nil")
		}
		if err := session.Validate(); err != nil {
			t.Fatalf("FixSession.Validate() failed: %v", err)
		}

		// Requirement 11.2: Each FixAttempt has Number > 0, non-zero Timestamp,
		// Strategy with non-empty Description and Prompt.
		for idx, attempt := range session.Attempts {
			if attempt.Number <= 0 {
				t.Fatalf("Attempt[%d].Number must be > 0, got %d", idx, attempt.Number)
			}
			if attempt.Timestamp.IsZero() {
				t.Fatalf("Attempt[%d].Timestamp must not be zero", idx)
			}
			if attempt.Strategy.Description == "" {
				t.Fatalf("Attempt[%d].Strategy.Description must not be empty", idx)
			}
			if attempt.Strategy.Prompt == "" {
				t.Fatalf("Attempt[%d].Strategy.Prompt must not be empty", idx)
			}
		}

		// Requirement 11.3: Strategy contains description, prompt, and AI response fields.
		for idx, attempt := range session.Attempts {
			_ = attempt.Strategy.Description // already checked non-empty above
			_ = attempt.Strategy.Prompt      // already checked non-empty above
			_ = attempt.Strategy.AIResponse  // may be empty (AI hasn't responded yet), but field must exist

			// Verify the Strategy struct has all three fields populated as strings.
			s := attempt.Strategy
			if len(s.Description) == 0 {
				t.Fatalf("Attempt[%d].Strategy.Description is empty", idx)
			}
			if len(s.Prompt) == 0 {
				t.Fatalf("Attempt[%d].Strategy.Prompt is empty", idx)
			}
			// AIResponse can be empty, but the field must be a valid string (type check is compile-time).
		}
	})
}
