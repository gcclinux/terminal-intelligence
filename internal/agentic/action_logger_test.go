package agentic

import (
	"regexp"
	"testing"
)

func TestNewActionLogger(t *testing.T) {
	called := false
	logger := NewActionLogger(func(msg string) {
		called = true
	})
	if logger == nil {
		t.Fatal("NewActionLogger returned nil")
	}
	logger.Log("hello")
	if !called {
		t.Fatal("notify was not called")
	}
}

func TestActionLoggerTimestampFormat(t *testing.T) {
	var captured string
	logger := NewActionLogger(func(msg string) {
		captured = msg
	})

	logger.Log("test message")

	// Expect "[HH:MM:SS] test message"
	pattern := `^\[\d{2}:\d{2}:\d{2}\] test message$`
	matched, err := regexp.MatchString(pattern, captured)
	if err != nil {
		t.Fatalf("regex error: %v", err)
	}
	if !matched {
		t.Fatalf("unexpected format: %q", captured)
	}
}

func TestActionLoggerFormatArgs(t *testing.T) {
	var captured string
	logger := NewActionLogger(func(msg string) {
		captured = msg
	})

	logger.Log("file %s: %d lines added", "main.go", 42)

	pattern := `^\[\d{2}:\d{2}:\d{2}\] file main\.go: 42 lines added$`
	matched, err := regexp.MatchString(pattern, captured)
	if err != nil {
		t.Fatalf("regex error: %v", err)
	}
	if !matched {
		t.Fatalf("unexpected format: %q", captured)
	}
}

func TestActionLoggerNotifyCalledOnce(t *testing.T) {
	callCount := 0
	logger := NewActionLogger(func(msg string) {
		callCount++
	})

	logger.Log("single call")
	if callCount != 1 {
		t.Fatalf("expected notify called once, got %d", callCount)
	}
}
