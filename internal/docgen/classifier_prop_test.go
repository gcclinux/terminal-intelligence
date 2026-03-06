package docgen

import (
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// docKeywords returns the keywords for a specific documentation type
func docKeywords(docType DocumentationType) []string {
	switch docType {
	case DocTypeUserManual:
		return []string{"user manual", "manual", "how to use", "user guide"}
	case DocTypeInstallation:
		return []string{"installation", "setup", "install", "getting started"}
	case DocTypeAPI:
		return []string{"api", "reference", "function", "class", "method"}
	case DocTypeTutorial:
		return []string{"tutorial", "guide", "walkthrough", "example"}
	default:
		return []string{}
	}
}

// textWithKeyword generates text containing a specific keyword for a doc type
func textWithKeyword(docType DocumentationType) *rapid.Generator[string] {
	keywords := docKeywords(docType)
	if len(keywords) == 0 {
		return rapid.Just("")
	}

	return rapid.Custom(func(t *rapid.T) string {
		// Pick a random keyword
		keyword := rapid.SampledFrom(keywords).Draw(t, "keyword")

		// Generate optional prefix and suffix text
		prefix := rapid.StringMatching("[a-zA-Z0-9 ]{0,20}").Draw(t, "prefix")
		suffix := rapid.StringMatching("[a-zA-Z0-9 ]{0,20}").Draw(t, "suffix")

		// Randomly vary the case of the keyword
		caseVariant := rapid.IntRange(0, 2).Draw(t, "caseVariant")
		switch caseVariant {
		case 0:
			keyword = strings.ToLower(keyword)
		case 1:
			keyword = strings.ToUpper(keyword)
		case 2:
			keyword = strings.Title(strings.ToLower(keyword))
		}

		// Combine with optional prefix/suffix
		parts := []string{}
		if strings.TrimSpace(prefix) != "" {
			parts = append(parts, strings.TrimSpace(prefix))
		}
		parts = append(parts, keyword)
		if strings.TrimSpace(suffix) != "" {
			parts = append(parts, strings.TrimSpace(suffix))
		}

		return strings.Join(parts, " ")
	})
}

// textWithoutKeywords generates text that doesn't contain any doc keywords
func textWithoutKeywords() *rapid.Generator[string] {
	// Generate text from safe words that aren't doc keywords
	safeWords := []string{
		"create", "generate", "make", "build", "write", "produce",
		"documentation", "docs", "content", "text", "file", "output",
		"project", "code", "software", "application", "program",
		"help", "information", "details", "description",
	}

	return rapid.Custom(func(t *rapid.T) string {
		// Pick 1-5 random words
		numWords := rapid.IntRange(1, 5).Draw(t, "numWords")
		words := make([]string, numWords)
		for i := 0; i < numWords; i++ {
			words[i] = rapid.SampledFrom(safeWords).Draw(t, "word")
		}
		return strings.Join(words, " ")
	})
}

// TestProperty4_DocumentationTypeClassification verifies that for any input
// with doc keywords, classifier returns matching type.
// **Validates: Requirements 2.1, 2.2, 2.3, 2.4, 2.5, 9.1**
func TestProperty4_DocumentationTypeClassification(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Pick a random documentation type (excluding General)
		docTypes := []DocumentationType{
			DocTypeUserManual,
			DocTypeInstallation,
			DocTypeAPI,
			DocTypeTutorial,
		}
		expectedType := rapid.SampledFrom(docTypes).Draw(t, "docType")

		// Generate text containing a keyword for this type
		input := textWithKeyword(expectedType).Draw(t, "input")

		classifier := NewRequestClassifier()
		result := classifier.Classify(input)

		// Verify the expected type is in the results
		found := false
		for _, resultType := range result.Types {
			if resultType == expectedType {
				found = true
				break
			}
		}

		if !found {
			t.Fatalf("Expected type %v in classification results for input %q, got %v",
				expectedType, input, result.Types)
		}

		// Verify confidence is in valid range
		if result.Confidence <= 0 || result.Confidence > 1.0 {
			t.Fatalf("Expected confidence between 0 and 1.0, got %f", result.Confidence)
		}
	})
}

// TestProperty5_FallbackClassification verifies that for any input without
// keywords, returns DocTypeGeneral.
// **Validates: Requirements 2.5**
func TestProperty5_FallbackClassification(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate text without any doc keywords
		input := textWithoutKeywords().Draw(t, "input")

		classifier := NewRequestClassifier()
		result := classifier.Classify(input)

		// Verify DocTypeGeneral is returned
		if len(result.Types) == 0 {
			t.Fatalf("Expected at least one type in result, got none")
		}

		if result.Types[0] != DocTypeGeneral {
			t.Fatalf("Expected DocTypeGeneral for input without keywords %q, got %v",
				input, result.Types[0])
		}

		// Verify confidence is 1.0 for General type
		if result.Confidence != 1.0 {
			t.Fatalf("Expected confidence 1.0 for General type, got %f", result.Confidence)
		}
	})
}

// TestProperty24_MultiTypeClassification verifies that for any input with
// multiple keywords, all types identified.
// **Validates: Requirements 9.1**
func TestProperty24_MultiTypeClassification(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Pick 2-4 random documentation types (excluding General)
		allTypes := []DocumentationType{
			DocTypeUserManual,
			DocTypeInstallation,
			DocTypeAPI,
			DocTypeTutorial,
		}

		// Shuffle and pick a subset
		numTypes := rapid.IntRange(2, 4).Draw(t, "numTypes")
		selectedTypes := make([]DocumentationType, 0, numTypes)
		availableTypes := make([]DocumentationType, len(allTypes))
		copy(availableTypes, allTypes)

		for i := 0; i < numTypes && len(availableTypes) > 0; i++ {
			idx := rapid.IntRange(0, len(availableTypes)-1).Draw(t, "typeIdx")
			selectedTypes = append(selectedTypes, availableTypes[idx])
			// Remove selected type
			availableTypes = append(availableTypes[:idx], availableTypes[idx+1:]...)
		}

		// Generate text with keywords from all selected types
		textParts := make([]string, len(selectedTypes))
		for i, docType := range selectedTypes {
			textParts[i] = textWithKeyword(docType).Draw(t, "text")
		}

		// Shuffle the parts to randomize order
		for i := len(textParts) - 1; i > 0; i-- {
			j := rapid.IntRange(0, i).Draw(t, "shuffleIdx")
			textParts[i], textParts[j] = textParts[j], textParts[i]
		}

		input := strings.Join(textParts, " and ")

		classifier := NewRequestClassifier()
		result := classifier.Classify(input)

		// Verify all expected types are in the results
		for _, expectedType := range selectedTypes {
			found := false
			for _, resultType := range result.Types {
				if resultType == expectedType {
					found = true
					break
				}
			}

			if !found {
				t.Fatalf("Expected type %v in classification results for input %q, got %v",
					expectedType, input, result.Types)
			}
		}

		// Verify we got at least as many types as we expected
		if len(result.Types) < len(selectedTypes) {
			t.Fatalf("Expected at least %d types for input %q, got %d types: %v",
				len(selectedTypes), input, len(result.Types), result.Types)
		}

		// Verify confidence is in valid range
		if result.Confidence <= 0 || result.Confidence > 1.0 {
			t.Fatalf("Expected confidence between 0 and 1.0, got %f", result.Confidence)
		}
	})
}
