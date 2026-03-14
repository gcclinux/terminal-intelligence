package agentic

import (
	"strings"
	"testing"
	"time"
)

func TestNewAttemptTracker(t *testing.T) {
	tracker := NewAttemptTracker()
	if tracker == nil {
		t.Fatal("NewAttemptTracker returned nil")
	}
	if tracker.AttemptCount() != 0 {
		t.Fatalf("expected 0 attempts, got %d", tracker.AttemptCount())
	}
}

func TestAttemptTrackerRecord(t *testing.T) {
	tracker := NewAttemptTracker()

	attempt := FixAttempt{
		Number:    1,
		Cycle:     0,
		Strategy:  Strategy{Description: "fix imports"},
		Timestamp: time.Now(),
	}
	tracker.Record(attempt)

	if tracker.AttemptCount() != 1 {
		t.Fatalf("expected 1 attempt, got %d", tracker.AttemptCount())
	}
}

func TestAttemptTrackerRecordMultiple(t *testing.T) {
	tracker := NewAttemptTracker()

	for i := 1; i <= 5; i++ {
		tracker.Record(FixAttempt{
			Number:    i,
			Cycle:     (i - 1) / 3,
			Strategy:  Strategy{Description: "strategy"},
			Timestamp: time.Now(),
		})
	}

	if tracker.AttemptCount() != 5 {
		t.Fatalf("expected 5 attempts, got %d", tracker.AttemptCount())
	}
}

func TestGenerateSummaryEmpty(t *testing.T) {
	tracker := NewAttemptTracker()
	summary := tracker.GenerateSummary()
	if summary != "" {
		t.Fatalf("expected empty summary for no attempts, got %q", summary)
	}
}

func TestGenerateSummaryContainsAttemptInfo(t *testing.T) {
	tracker := NewAttemptTracker()

	tracker.Record(FixAttempt{
		Number:   1,
		Cycle:    0,
		Strategy: Strategy{Description: "refactor error handling"},
		FilesModified: []FileResult{
			{Path: "main.go"},
			{Path: "util.go"},
		},
		TestResult: &TestResult{ExitCode: 1},
		Timestamp:  time.Now(),
	})

	tracker.Record(FixAttempt{
		Number:   2,
		Cycle:    0,
		Strategy: Strategy{Description: "add nil check"},
		FilesModified: []FileResult{
			{Path: "handler.go"},
		},
		TestResult: &TestResult{ExitCode: 0},
		Timestamp:  time.Now(),
	})

	summary := tracker.GenerateSummary()

	checks := []string{
		"Attempt 1",
		"Attempt 2",
		"refactor error handling",
		"add nil check",
		"main.go",
		"util.go",
		"handler.go",
		"FAIL (exit code 1)",
		"PASS (exit code 0)",
		"tests failed",
		"tests passed",
	}

	for _, check := range checks {
		if !strings.Contains(summary, check) {
			t.Errorf("summary missing %q\nsummary:\n%s", check, summary)
		}
	}
}

func TestGenerateSummaryNoTestResult(t *testing.T) {
	tracker := NewAttemptTracker()

	tracker.Record(FixAttempt{
		Number:    1,
		Cycle:     0,
		Strategy:  Strategy{Description: "initial attempt"},
		Timestamp: time.Now(),
	})

	summary := tracker.GenerateSummary()
	if !strings.Contains(summary, "no tests run") {
		t.Errorf("expected 'no tests run' in summary, got:\n%s", summary)
	}
}

func TestGenerateSummaryTimedOut(t *testing.T) {
	tracker := NewAttemptTracker()

	tracker.Record(FixAttempt{
		Number:     1,
		Cycle:      0,
		Strategy:   Strategy{Description: "slow fix"},
		TestResult: &TestResult{ExitCode: 1, TimedOut: true},
		Timestamp:  time.Now(),
	})

	summary := tracker.GenerateSummary()
	if !strings.Contains(summary, "test timed out") {
		t.Errorf("expected 'test timed out' in summary, got:\n%s", summary)
	}
}
