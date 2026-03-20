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
				Number: i + 1,
				Cycle:  rapid.IntRange(0, 2).Draw(t, "cycle"),
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

// Feature: content-preserving-file-updates, Property 4: Classifier always returns valid EditIntent
// For any EditIntent with OperationType in {"replace","append","insert","patch"} and Confidence in [0.0, 1.0],
// Validate() returns nil.
// **Validates: Requirements 1.5, 6.1, 6.2**
func TestProperty4_ClassifierAlwaysReturnsValidEditIntent(t *testing.T) {
	validOps := []string{"replace", "append", "insert", "patch"}

	rapid.Check(t, func(t *rapid.T) {
		opType := rapid.SampledFrom(validOps).Draw(t, "operationType")
		confidence := rapid.Float64Range(0.0, 1.0).Draw(t, "confidence")

		// Generate random keywords (0-5 entries).
		numKeywords := rapid.IntRange(0, 5).Draw(t, "numKeywords")
		keywords := make([]string, numKeywords)
		for i := range numKeywords {
			keywords[i] = rapid.StringMatching(`[a-z]{1,15}`).Draw(t, "keyword")
		}

		intent := EditIntent{
			OperationType: opType,
			Confidence:    confidence,
			Keywords:      keywords,
		}

		// Property: any EditIntent with valid OperationType and valid Confidence passes Validate().
		if err := intent.Validate(); err != nil {
			t.Fatalf("EditIntent.Validate() should return nil for valid inputs (op=%q, conf=%f), got: %v",
				opType, confidence, err)
		}
	})
}

// Feature: content-preserving-file-updates, Property 5: EditIntent validation rejects invalid inputs
// For any EditIntent with OperationType NOT in {"replace","append","insert","patch"} OR Confidence outside [0.0, 1.0],
// Validate() returns non-nil error.
// **Validates: Requirements 6.4, 6.5**
func TestProperty5_EditIntentValidationRejectsInvalidInputs(t *testing.T) {
	validOps := map[string]bool{
		"replace": true,
		"append":  true,
		"insert":  true,
		"patch":   true,
	}

	rapid.Check(t, func(t *rapid.T) {
		// Decide which invariant to violate: 0 = invalid OperationType, 1 = invalid Confidence, 2 = both.
		violationType := rapid.IntRange(0, 2).Draw(t, "violationType")

		var opType string
		var confidence float64

		switch violationType {
		case 0:
			// Invalid OperationType, valid Confidence.
			opType = rapid.StringMatching(`[a-z]{1,20}`).
				Filter(func(s string) bool { return !validOps[s] }).
				Draw(t, "invalidOpType")
			confidence = rapid.Float64Range(0.0, 1.0).Draw(t, "validConfidence")

		case 1:
			// Valid OperationType, invalid Confidence.
			validOpSlice := []string{"replace", "append", "insert", "patch"}
			opType = rapid.SampledFrom(validOpSlice).Draw(t, "validOpType")
			// Generate confidence outside [0.0, 1.0].
			if rapid.Bool().Draw(t, "negativeConfidence") {
				confidence = -rapid.Float64Range(0.01, 1000.0).Draw(t, "negConf")
			} else {
				confidence = 1.0 + rapid.Float64Range(0.01, 1000.0).Draw(t, "highConf")
			}

		case 2:
			// Both invalid.
			opType = rapid.StringMatching(`[a-z]{1,20}`).
				Filter(func(s string) bool { return !validOps[s] }).
				Draw(t, "invalidOpType2")
			if rapid.Bool().Draw(t, "negativeConfidence2") {
				confidence = -rapid.Float64Range(0.01, 1000.0).Draw(t, "negConf2")
			} else {
				confidence = 1.0 + rapid.Float64Range(0.01, 1000.0).Draw(t, "highConf2")
			}
		}

		intent := EditIntent{
			OperationType: opType,
			Confidence:    confidence,
		}

		// Property: Validate() must return a non-nil error for any invalid input.
		if err := intent.Validate(); err == nil {
			t.Fatalf("EditIntent.Validate() should return error for invalid inputs (op=%q, conf=%f), but got nil",
				opType, confidence)
		}
	})
}
