package agentic

import (
	"testing"
)

func TestNewIntentClassifier(t *testing.T) {
	ic := NewIntentClassifier()
	if ic == nil {
		t.Fatal("NewIntentClassifier returned nil")
	}
}

func TestClassify_FixCommand(t *testing.T) {
	ic := NewIntentClassifier()
	intent := ic.Classify("/fix the broken function")
	if intent.OperationType != "patch" {
		t.Errorf("expected patch, got %s", intent.OperationType)
	}
	if intent.Confidence != 1.0 {
		t.Errorf("expected confidence 1.0, got %f", intent.Confidence)
	}
	if len(intent.Keywords) == 0 || intent.Keywords[0] != "/fix" {
		t.Errorf("expected keyword /fix, got %v", intent.Keywords)
	}
}

func TestClassify_FixCommandCaseInsensitive(t *testing.T) {
	ic := NewIntentClassifier()
	intent := ic.Classify("/FIX the broken function")
	if intent.OperationType != "patch" {
		t.Errorf("expected patch, got %s", intent.OperationType)
	}
	if intent.Confidence != 1.0 {
		t.Errorf("expected confidence 1.0, got %f", intent.Confidence)
	}
}

func TestClassify_AdditiveKeywords(t *testing.T) {
	tests := []struct {
		name    string
		message string
	}{
		{"add", "add a new function to handle errors"},
		{"append", "append a footer to the file"},
		{"insert", "insert a comment at the top"},
		{"add to", "add to the existing list"},
		{"also include", "also include error handling"},
		{"include also", "include also the logging module"},
		{"update the file and include", "update the file and include a new section"},
	}

	ic := NewIntentClassifier()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			intent := ic.Classify(tt.message)
			if intent.OperationType != "append" {
				t.Errorf("expected append, got %s", intent.OperationType)
			}
			if intent.Confidence < 0.8 {
				t.Errorf("expected confidence >= 0.8, got %f", intent.Confidence)
			}
		})
	}
}

func TestClassify_UpdateAlsoPattern(t *testing.T) {
	tests := []struct {
		name    string
		message string
	}{
		{"update also adjacent", "update also the tests"},
		{"update with gap", "update the config and also add logging"},
		{"update far also", "update everything in the module, also include tests"},
	}

	ic := NewIntentClassifier()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			intent := ic.Classify(tt.message)
			if intent.OperationType != "append" {
				t.Errorf("expected append, got %s", intent.OperationType)
			}
			if intent.Confidence < 0.8 {
				t.Errorf("expected confidence >= 0.8, got %f", intent.Confidence)
			}
		})
	}
}

func TestClassify_ReplacementKeywords(t *testing.T) {
	tests := []struct {
		name    string
		message string
	}{
		{"replace", "replace the entire function"},
		{"rewrite", "rewrite this module from scratch"},
		{"redo", "redo the implementation"},
		{"start over", "start over with a clean slate"},
	}

	ic := NewIntentClassifier()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			intent := ic.Classify(tt.message)
			if intent.OperationType != "replace" {
				t.Errorf("expected replace, got %s", intent.OperationType)
			}
			if intent.Confidence < 0.8 {
				t.Errorf("expected confidence >= 0.8, got %f", intent.Confidence)
			}
		})
	}
}

func TestClassify_AmbiguousMessage(t *testing.T) {
	tests := []struct {
		name    string
		message string
	}{
		{"generic request", "make the code better"},
		{"question", "what does this function do"},
		{"empty", ""},
		{"whitespace only", "   "},
	}

	ic := NewIntentClassifier()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			intent := ic.Classify(tt.message)
			if intent.OperationType != "patch" {
				t.Errorf("expected patch, got %s", intent.OperationType)
			}
			if intent.Confidence != 0.5 {
				t.Errorf("expected confidence 0.5, got %f", intent.Confidence)
			}
		})
	}
}

func TestClassify_AdditiveTakesPriorityOverReplacement(t *testing.T) {
	ic := NewIntentClassifier()
	// Message contains both additive ("add") and replacement ("replace") keywords.
	// Additive should win because it's checked first.
	intent := ic.Classify("add a new section and replace the old header")
	if intent.OperationType != "append" {
		t.Errorf("expected append (additive priority), got %s", intent.OperationType)
	}
}

func TestClassify_ResultAlwaysValid(t *testing.T) {
	ic := NewIntentClassifier()
	messages := []string{
		"/fix something",
		"add a function",
		"replace everything",
		"just do something",
		"",
		"update the file and include tests",
		"update the config also add logging",
	}
	for _, msg := range messages {
		intent := ic.Classify(msg)
		if err := intent.Validate(); err != nil {
			t.Errorf("Classify(%q) returned invalid EditIntent: %v", msg, err)
		}
	}
}
