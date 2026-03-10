package docgen

import (
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// safeText generates simple text without special characters that could interfere with parsing
func safeText() *rapid.Generator[string] {
	return rapid.StringMatching("[a-zA-Z][a-zA-Z0-9 ]{0,30}")
}

// flagVariant generates different case variations of a flag
func flagVariant(flag string) *rapid.Generator[string] {
	variants := []string{
		flag,
		strings.ToUpper(flag),
		strings.ToLower(flag),
		strings.Title(strings.ToLower(flag)),
	}
	return rapid.SampledFrom(variants)
}

// TestProperty1_CommandFlagRecognition verifies that for any input string
// containing the /doc flag, the parser correctly identifies it and sets both
// IsDocRequest and IsProjectWide to true (since /doc now implies project-wide).
// **Validates: Requirements 1.1, 1.2**
func TestProperty1_CommandFlagRecognition(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random text and the /doc flag
		text := safeText().Draw(t, "text")

		// Generate /doc flag with random case variation
		flag := flagVariant("/doc").Draw(t, "docFlag")

		// Position flag randomly: before, after, or in middle of text
		position := rapid.IntRange(0, 2).Draw(t, "position")
		var input string
		switch position {
		case 0: // Flag before text
			input = flag + " " + text
		case 1: // Flag after text
			input = text + " " + flag
		case 2: // Flag in middle (split text)
			words := strings.Fields(text)
			if len(words) > 1 {
				mid := len(words) / 2
				input = strings.Join(words[:mid], " ") + " " + flag + " " + strings.Join(words[mid:], " ")
			} else {
				input = flag + " " + text
			}
		}

		parser := NewCommandParser()
		result, err := parser.Parse(input)

		if err != nil {
			t.Fatalf("Parse failed: %v", err)
		}

		// Verify /doc flag was detected and implies project-wide
		if !result.IsDocRequest {
			t.Fatalf("Expected IsDocRequest=true for input %q, got false", input)
		}
		if !result.IsProjectWide {
			t.Fatalf("Expected IsProjectWide=true for input %q (implied by /doc), got false", input)
		}
	})
}

// TestProperty2_CombinedFlagDetection verifies that for any input string
// containing both /project and /doc flags (in any order), the parser sets
// both IsProjectWide and IsDocRequest to true.
// **Validates: Requirements 1.3**
func TestProperty2_CombinedFlagDetection(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random text
		text := safeText().Draw(t, "text")

		// Generate both flags with random case variations
		projectFlag := flagVariant("/project").Draw(t, "projectFlag")
		docFlag := flagVariant("/doc").Draw(t, "docFlag")

		// Randomly order the flags
		flagsFirst := rapid.Bool().Draw(t, "flagsFirst")
		projectFirst := rapid.Bool().Draw(t, "projectFirst")

		var input string
		if projectFirst {
			if flagsFirst {
				input = projectFlag + " " + docFlag + " " + text
			} else {
				input = text + " " + projectFlag + " " + docFlag
			}
		} else {
			if flagsFirst {
				input = docFlag + " " + projectFlag + " " + text
			} else {
				input = text + " " + docFlag + " " + projectFlag
			}
		}

		parser := NewCommandParser()
		result, err := parser.Parse(input)

		if err != nil {
			t.Fatalf("Parse failed: %v", err)
		}

		// Verify both flags were detected
		if !result.IsProjectWide {
			t.Fatalf("Expected IsProjectWide=true for input %q, got false", input)
		}
		if !result.IsDocRequest {
			t.Fatalf("Expected IsDocRequest=true for input %q, got false", input)
		}
	})
}

// TestProperty3_NaturalLanguageExtraction verifies that for any input string
// with command flags, the parser extracts the non-flag portion as the natural
// language request, preserving the original text without the flags.
// **Validates: Requirements 1.4**
func TestProperty3_NaturalLanguageExtraction(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random natural language text (the part we want to preserve)
		naturalText := safeText().Draw(t, "naturalText")

		// Skip empty or whitespace-only text
		if strings.TrimSpace(naturalText) == "" {
			t.Skip("empty natural text")
		}

		// Generate flags to add - /doc is required, /project is optional for backward compatibility
		includeProject := rapid.Bool().Draw(t, "includeProject")

		// Build input with flags
		var input string
		if includeProject {
			// Both flags (backward compatibility)
			projectFlag := flagVariant("/project").Draw(t, "projectFlag")
			docFlag := flagVariant("/doc").Draw(t, "docFlag")
			input = projectFlag + " " + docFlag + " " + naturalText
		} else {
			// Only /doc flag (new primary usage)
			docFlag := flagVariant("/doc").Draw(t, "docFlag")
			input = docFlag + " " + naturalText
		}

		parser := NewCommandParser()
		result, err := parser.Parse(input)

		if err != nil {
			t.Fatalf("Parse failed: %v", err)
		}

		// Verify the natural language was extracted correctly
		// The parser should remove flags and normalize whitespace
		expectedNatural := strings.TrimSpace(naturalText)
		// Normalize multiple spaces to single space
		expectedNatural = strings.Join(strings.Fields(expectedNatural), " ")

		if result.NaturalLanguage != expectedNatural {
			t.Fatalf("Expected NaturalLanguage=%q for input %q, got %q",
				expectedNatural, input, result.NaturalLanguage)
		}
	})
}
