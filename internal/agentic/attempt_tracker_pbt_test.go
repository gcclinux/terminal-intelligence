package agentic

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"pgregory.net/rapid"
)

// Feature: project-wide-agentic-fixer, Property 5: Attempt tracker records and retains all attempt data
// **Validates: Requirements 4.1, 4.4**
func TestProperty5_AttemptTrackerRecordsAndRetainsAllAttemptData(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		tracker := NewAttemptTracker()

		// Generate a random number of attempts (1-20).
		numAttempts := rapid.IntRange(1, 20).Draw(t, "numAttempts")
		recorded := make([]FixAttempt, 0, numAttempts)

		for i := 0; i < numAttempts; i++ {
			// Generate random file results.
			numFiles := rapid.IntRange(0, 5).Draw(t, fmt.Sprintf("numFiles_%d", i))
			files := make([]FileResult, numFiles)
			for j := 0; j < numFiles; j++ {
				files[j] = FileResult{
					Path:         rapid.StringMatching(`[a-z]{1,8}\.(go|py|sh)`).Draw(t, fmt.Sprintf("filePath_%d_%d", i, j)),
					LinesAdded:   rapid.IntRange(0, 100).Draw(t, fmt.Sprintf("linesAdded_%d_%d", i, j)),
					LinesRemoved: rapid.IntRange(0, 100).Draw(t, fmt.Sprintf("linesRemoved_%d_%d", i, j)),
				}
			}

			// Generate optional test result.
			var testResult *TestResult
			hasTest := rapid.Bool().Draw(t, fmt.Sprintf("hasTest_%d", i))
			if hasTest {
				testResult = &TestResult{
					ExitCode: rapid.IntRange(0, 127).Draw(t, fmt.Sprintf("exitCode_%d", i)),
				}
			}

			attempt := FixAttempt{
				Number:        i + 1,
				Cycle:         rapid.IntRange(0, 2).Draw(t, fmt.Sprintf("cycle_%d", i)),
				Strategy:      Strategy{Description: rapid.StringMatching(`[a-zA-Z][a-zA-Z0-9 ]{0,49}`).Draw(t, fmt.Sprintf("stratDesc_%d", i))},
				FilesModified: files,
				TestResult:    testResult,
				Timestamp:     time.Unix(rapid.Int64Range(1, 4102444800).Draw(t, fmt.Sprintf("ts_%d", i)), 0),
			}

			tracker.Record(attempt)
			recorded = append(recorded, attempt)
		}

		// Verify AttemptCount matches the number of recorded attempts.
		if tracker.AttemptCount() != numAttempts {
			t.Fatalf("expected AttemptCount()=%d, got %d", numAttempts, tracker.AttemptCount())
		}

		// Verify data integrity: each recorded attempt's data is retained.
		for i, expected := range recorded {
			actual := tracker.attempts[i]
			if actual.Number != expected.Number {
				t.Fatalf("attempt[%d].Number: expected %d, got %d", i, expected.Number, actual.Number)
			}
			if actual.Strategy.Description != expected.Strategy.Description {
				t.Fatalf("attempt[%d].Strategy.Description mismatch", i)
			}
			if len(actual.FilesModified) != len(expected.FilesModified) {
				t.Fatalf("attempt[%d].FilesModified length: expected %d, got %d", i, len(expected.FilesModified), len(actual.FilesModified))
			}
			for j, ef := range expected.FilesModified {
				if actual.FilesModified[j].Path != ef.Path {
					t.Fatalf("attempt[%d].FilesModified[%d].Path mismatch", i, j)
				}
			}
			if (expected.TestResult == nil) != (actual.TestResult == nil) {
				t.Fatalf("attempt[%d].TestResult nil mismatch", i)
			}
			if expected.TestResult != nil && actual.TestResult.ExitCode != expected.TestResult.ExitCode {
				t.Fatalf("attempt[%d].TestResult.ExitCode: expected %d, got %d", i, expected.TestResult.ExitCode, actual.TestResult.ExitCode)
			}
		}
	})
}

// Feature: project-wide-agentic-fixer, Property 6: Attempt tracker summary generation
// **Validates: Requirements 11.4**
func TestProperty6_AttemptTrackerSummaryGeneration(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		tracker := NewAttemptTracker()

		// Generate 1-10 attempts.
		numAttempts := rapid.IntRange(1, 10).Draw(t, "numAttempts")
		type attemptInfo struct {
			number      int
			description string
			exitCode    int
			hasTest     bool
			timedOut    bool
		}
		infos := make([]attemptInfo, numAttempts)

		for i := 0; i < numAttempts; i++ {
			desc := rapid.StringMatching(`[a-zA-Z][a-zA-Z0-9 ]{0,49}`).Draw(t, fmt.Sprintf("desc_%d", i))
			hasTest := rapid.Bool().Draw(t, fmt.Sprintf("hasTest_%d", i))
			exitCode := 0
			timedOut := false
			var testResult *TestResult
			if hasTest {
				exitCode = rapid.IntRange(0, 127).Draw(t, fmt.Sprintf("exitCode_%d", i))
				timedOut = rapid.Bool().Draw(t, fmt.Sprintf("timedOut_%d", i))
				testResult = &TestResult{ExitCode: exitCode, TimedOut: timedOut}
			}

			infos[i] = attemptInfo{
				number:      i + 1,
				description: desc,
				exitCode:    exitCode,
				hasTest:     hasTest,
				timedOut:    timedOut,
			}

			tracker.Record(FixAttempt{
				Number:    i + 1,
				Cycle:     rapid.IntRange(0, 2).Draw(t, fmt.Sprintf("cycle_%d", i)),
				Strategy:  Strategy{Description: desc},
				TestResult: testResult,
				Timestamp: time.Unix(rapid.Int64Range(1, 4102444800).Draw(t, fmt.Sprintf("ts_%d", i)), 0),
			})
		}

		summary := tracker.GenerateSummary()

		// Summary must be non-empty.
		if summary == "" {
			t.Fatal("GenerateSummary() returned empty string for non-empty tracker")
		}

		// Verify each attempt's info appears in the summary.
		for _, info := range infos {
			// Check attempt number.
			attemptLabel := fmt.Sprintf("Attempt %d", info.number)
			if !strings.Contains(summary, attemptLabel) {
				t.Fatalf("summary missing %q\nsummary:\n%s", attemptLabel, summary)
			}

			// Check strategy description.
			if !strings.Contains(summary, info.description) {
				t.Fatalf("summary missing strategy description %q\nsummary:\n%s", info.description, summary)
			}

			// Check outcome.
			if !info.hasTest {
				if !strings.Contains(summary, "no tests run") {
					t.Fatalf("summary missing 'no tests run' for attempt %d\nsummary:\n%s", info.number, summary)
				}
			} else if info.timedOut {
				if !strings.Contains(summary, "test timed out") {
					t.Fatalf("summary missing 'test timed out' for attempt %d\nsummary:\n%s", info.number, summary)
				}
			} else if info.exitCode == 0 {
				if !strings.Contains(summary, "tests passed") {
					t.Fatalf("summary missing 'tests passed' for attempt %d\nsummary:\n%s", info.number, summary)
				}
			} else {
				if !strings.Contains(summary, "tests failed") {
					t.Fatalf("summary missing 'tests failed' for attempt %d\nsummary:\n%s", info.number, summary)
				}
			}
		}
	})
}

// Feature: project-wide-agentic-fixer, Property 9: Knowledge accumulation across reset cycles
// **Validates: Requirements 5.5**
func TestProperty9_KnowledgeAccumulationAcrossResetCycles(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		tracker := NewAttemptTracker()

		// Simulate 1-5 reset cycles, each with exactly 3 attempts.
		numCycles := rapid.IntRange(1, 5).Draw(t, "numCycles")
		attemptNum := 0

		for cycle := 0; cycle < numCycles; cycle++ {
			for a := 0; a < 3; a++ {
				attemptNum++
				tracker.Record(FixAttempt{
					Number:   attemptNum,
					Cycle:    cycle,
					Strategy: Strategy{Description: rapid.StringMatching(`[a-zA-Z][a-zA-Z0-9 ]{0,29}`).Draw(t, fmt.Sprintf("desc_%d_%d", cycle, a))},
					TestResult: &TestResult{
						ExitCode: rapid.IntRange(1, 127).Draw(t, fmt.Sprintf("exitCode_%d_%d", cycle, a)),
					},
					Timestamp: time.Unix(rapid.Int64Range(1, 4102444800).Draw(t, fmt.Sprintf("ts_%d_%d", cycle, a)), 0),
				})
			}
		}

		// After C cycles of 3 attempts each, AttemptCount must equal 3*C.
		expected := 3 * numCycles
		if tracker.AttemptCount() != expected {
			t.Fatalf("after %d cycles, expected AttemptCount()=%d, got %d", numCycles, expected, tracker.AttemptCount())
		}

		// Verify all attempts from all cycles are retained (knowledge accumulation).
		for i := 0; i < expected; i++ {
			if tracker.attempts[i].Number != i+1 {
				t.Fatalf("attempt[%d].Number: expected %d, got %d", i, i+1, tracker.attempts[i].Number)
			}
			expectedCycle := i / 3
			if tracker.attempts[i].Cycle != expectedCycle {
				t.Fatalf("attempt[%d].Cycle: expected %d, got %d", i, expectedCycle, tracker.attempts[i].Cycle)
			}
		}
	})
}
