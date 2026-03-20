package agentic

import (
	"regexp"
	"strings"
)

// additiveKeywords are phrases that indicate the user wants to add content
// to an existing file rather than replace it. Multi-word phrases are ordered
// before single words so longer matches take priority.
var additiveKeywords = []string{
	"update the file and include",
	"include also",
	"also include",
	"add to",
	"append",
	"insert",
	"add",
}

// updateAlsoPattern matches "update" followed by "also" with anything in between,
// e.g. "update the config and also add logging".
var updateAlsoPattern = regexp.MustCompile(`(?i)update.*also`)

// replacementKeywords are phrases that indicate the user wants to fully replace
// the file content. Multi-word phrases are ordered before single words.
var replacementKeywords = []string{
	"start over",
	"replace",
	"rewrite",
	"redo",
}

// IntentClassifier analyzes user messages to determine the desired edit operation.
// It uses deterministic, rule-based keyword matching — no AI call is needed.
type IntentClassifier struct{}

// NewIntentClassifier creates a new IntentClassifier.
func NewIntentClassifier() *IntentClassifier {
	return &IntentClassifier{}
}

// Classify analyzes a user message and returns an EditIntent.
//
// Classification rules (applied in order):
//  1. "/fix" command prefix → patch at confidence 1.0
//  2. Additive keywords matched → append at confidence 0.8
//  3. Replacement keywords matched → replace at confidence 0.8
//  4. No matches (ambiguous) → patch at confidence 0.5
func (ic *IntentClassifier) Classify(message string) EditIntent {
	lower := strings.ToLower(strings.TrimSpace(message))

	// Rule 1: explicit /fix command
	if strings.HasPrefix(lower, "/fix") {
		return EditIntent{
			OperationType: "patch",
			Confidence:    1.0,
			Keywords:      []string{"/fix"},
		}
	}

	// Rule 2: check additive keywords (multi-word first)
	var matchedAdditive []string

	// Special regex pattern: "update.*also"
	if updateAlsoPattern.MatchString(lower) {
		matchedAdditive = append(matchedAdditive, "update.*also")
	}

	for _, kw := range additiveKeywords {
		if strings.Contains(lower, kw) {
			matchedAdditive = append(matchedAdditive, kw)
			break // first match is enough
		}
	}

	if len(matchedAdditive) > 0 {
		return EditIntent{
			OperationType: "append",
			Confidence:    0.8,
			Keywords:      matchedAdditive,
		}
	}

	// Rule 3: check replacement keywords (multi-word first)
	for _, kw := range replacementKeywords {
		if strings.Contains(lower, kw) {
			return EditIntent{
				OperationType: "replace",
				Confidence:    0.8,
				Keywords:      []string{kw},
			}
		}
	}

	// Rule 4: ambiguous — default to patch
	return EditIntent{
		OperationType: "patch",
		Confidence:    0.5,
		Keywords:      nil,
	}
}
