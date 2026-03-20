package agentic

import (
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// allAdditiveKeywords lists every additive keyword the classifier recognises.
var allAdditiveKeywords = []string{
	"update the file and include",
	"include also",
	"also include",
	"add to",
	"append",
	"insert",
	"add",
}

// allReplacementKeywords lists every replacement keyword the classifier recognises.
var allReplacementKeywords = []string{
	"start over",
	"replace",
	"rewrite",
	"redo",
}

// safePrefix generates a short prefix that cannot start with "/fix" (case-insensitive).
// It draws from digits and punctuation that won't form keywords.
func safePrefix(t *rapid.T, label string) string {
	return rapid.StringMatching(`[0-9!@#$%^&*()?]{0,10}`).Draw(t, label)
}

// safeSuffix generates a short suffix from digits and punctuation.
func safeSuffix(t *rapid.T, label string) string {
	return rapid.StringMatching(`[0-9!@#$%^&*()?]{0,10}`).Draw(t, label)
}

// containsAny returns true if s contains any of the given substrings (case-insensitive).
func containsAny(s string, subs []string) bool {
	lower := strings.ToLower(s)
	for _, sub := range subs {
		if strings.Contains(lower, strings.ToLower(sub)) {
			return true
		}
	}
	return false
}

// Feature: content-preserving-file-updates, Property 1: Additive keywords produce append or insert intent
// For any user message string that contains at least one additive keyword ("add", "append",
// "insert", "include also", "also include", "add to"), the Classify method should return an
// EditIntent with OperationType equal to "append" or "insert" and Confidence >= 0.8.
// The generated messages must NOT start with "/fix" and must NOT contain replacement keywords.
// **Validates: Requirements 1.1, 1.6**
func TestProperty1_AdditiveKeywordsProduceAppendOrInsert(t *testing.T) {
	ic := NewIntentClassifier()

	rapid.Check(t, func(t *rapid.T) {
		// Pick a random additive keyword to inject.
		keyword := rapid.SampledFrom(allAdditiveKeywords).Draw(t, "additiveKeyword")

		// Build a message: safePrefix + keyword + safeSuffix.
		// safePrefix uses digits/punctuation so it can't start with "/fix"
		// and can't accidentally form replacement keywords.
		prefix := safePrefix(t, "prefix")
		suffix := safeSuffix(t, "suffix")
		message := prefix + " " + keyword + " " + suffix

		// Verify preconditions: no /fix prefix, no replacement keywords.
		lower := strings.ToLower(strings.TrimSpace(message))
		if strings.HasPrefix(lower, "/fix") {
			t.Skip("generated message starts with /fix — skipping")
		}
		if containsAny(message, allReplacementKeywords) {
			t.Skip("generated message contains replacement keyword — skipping")
		}

		intent := ic.Classify(message)

		if intent.OperationType != "append" && intent.OperationType != "insert" {
			t.Fatalf("expected OperationType append or insert for message %q, got %q",
				message, intent.OperationType)
		}
		if intent.Confidence < 0.8 {
			t.Fatalf("expected Confidence >= 0.8 for message %q, got %f",
				message, intent.Confidence)
		}
	})
}

// Feature: content-preserving-file-updates, Property 2: Replacement keywords produce replace intent
// For any user message string that contains at least one replacement keyword ("replace", "rewrite",
// "redo", "start over") and NO additive keywords, the Classify method should return an EditIntent
// with OperationType equal to "replace" and Confidence >= 0.8.
// The generated messages must NOT start with "/fix".
// **Validates: Requirements 1.2**
func TestProperty2_ReplacementKeywordsProduceReplace(t *testing.T) {
	ic := NewIntentClassifier()

	rapid.Check(t, func(t *rapid.T) {
		// Pick a random replacement keyword to inject.
		keyword := rapid.SampledFrom(allReplacementKeywords).Draw(t, "replacementKeyword")

		// Build a message: safePrefix + keyword + safeSuffix.
		// safePrefix/safeSuffix use digits/punctuation so they can't form additive keywords.
		prefix := safePrefix(t, "prefix")
		suffix := safeSuffix(t, "suffix")
		message := prefix + " " + keyword + " " + suffix

		// Verify preconditions: no /fix prefix, no additive keywords.
		lower := strings.ToLower(strings.TrimSpace(message))
		if strings.HasPrefix(lower, "/fix") {
			t.Skip("generated message starts with /fix — skipping")
		}
		if containsAny(message, allAdditiveKeywords) {
			t.Skip("generated message contains additive keyword — skipping")
		}

		intent := ic.Classify(message)

		if intent.OperationType != "replace" {
			t.Fatalf("expected OperationType replace for message %q, got %q",
				message, intent.OperationType)
		}
		if intent.Confidence < 0.8 {
			t.Fatalf("expected Confidence >= 0.8 for message %q, got %f",
				message, intent.Confidence)
		}
	})
}

// Feature: content-preserving-file-updates, Property 3: Ambiguous messages default to patch
// For any user message string that contains none of the additive keywords, none of the replacement
// keywords, and does not start with "/fix", the Classify method should return an EditIntent with
// OperationType equal to "patch" and Confidence equal to 0.5.
// Uses a character set (digits and special chars) that cannot accidentally form keywords.
// **Validates: Requirements 1.4**
func TestProperty3_AmbiguousMessagesDefaultToPatch(t *testing.T) {
	ic := NewIntentClassifier()

	rapid.Check(t, func(t *rapid.T) {
		// Generate a string from digits and safe punctuation only.
		// This character set cannot form any of the additive/replacement keywords
		// (which are all alphabetic) or "/fix".
		message := rapid.StringMatching(`[0-9!@#$%^&*()_=+\[\]{}<>,.?~]{0,50}`).Draw(t, "ambiguousMessage")

		// Double-check preconditions (should always hold given the character set).
		lower := strings.ToLower(strings.TrimSpace(message))
		if strings.HasPrefix(lower, "/fix") {
			t.Skip("generated message starts with /fix — skipping")
		}
		if containsAny(message, allAdditiveKeywords) {
			t.Skip("generated message contains additive keyword — skipping")
		}
		if containsAny(message, allReplacementKeywords) {
			t.Skip("generated message contains replacement keyword — skipping")
		}

		intent := ic.Classify(message)

		if intent.OperationType != "patch" {
			t.Fatalf("expected OperationType patch for message %q, got %q",
				message, intent.OperationType)
		}
		if intent.Confidence != 0.5 {
			t.Fatalf("expected Confidence 0.5 for message %q, got %f",
				message, intent.Confidence)
		}
	})
}
