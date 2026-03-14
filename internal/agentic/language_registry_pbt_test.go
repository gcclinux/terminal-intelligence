package agentic

import (
	"fmt"
	"testing"

	"pgregory.net/rapid"
)

// Feature: project-wide-agentic-fixer, Property 12: Language detection from file extensions
// **Validates: Requirements 7.2, 9.1**
func TestProperty12_LanguageDetectionFromFileExtensions(t *testing.T) {
	// Collect all language IDs and their extensions from the registry.
	type langInfo struct {
		id         string
		extensions []string
	}
	var langs []langInfo
	for id, cfg := range defaultLanguageRegistry {
		langs = append(langs, langInfo{id: id, extensions: cfg.Extensions})
	}

	rapid.Check(t, func(t *rapid.T) {
		// Pick a dominant language index.
		dominantIdx := rapid.IntRange(0, len(langs)-1).Draw(t, "dominantIdx")
		dominant := langs[dominantIdx]

		// Generate a count for the dominant language (at least 2 files).
		dominantCount := rapid.IntRange(2, 20).Draw(t, "dominantCount")

		// Build file paths for the dominant language.
		var filePaths []string
		for i := 0; i < dominantCount; i++ {
			// Pick a random extension from the dominant language's extensions.
			extIdx := rapid.IntRange(0, len(dominant.extensions)-1).Draw(t, fmt.Sprintf("domExt_%d", i))
			name := rapid.StringMatching(`[a-z]{1,10}`).Draw(t, fmt.Sprintf("domName_%d", i))
			filePaths = append(filePaths, fmt.Sprintf("src/%s%s", name, dominant.extensions[extIdx]))
		}

		// Optionally add files from other languages, each with strictly fewer files than dominant.
		for j, lang := range langs {
			if j == dominantIdx {
				continue
			}
			// 0 to dominantCount-1 files for each other language (strictly less).
			otherCount := rapid.IntRange(0, dominantCount-1).Draw(t, fmt.Sprintf("otherCount_%d", j))
			for k := 0; k < otherCount; k++ {
				extIdx := rapid.IntRange(0, len(lang.extensions)-1).Draw(t, fmt.Sprintf("otherExt_%d_%d", j, k))
				name := rapid.StringMatching(`[a-z]{1,10}`).Draw(t, fmt.Sprintf("otherName_%d_%d", j, k))
				filePaths = append(filePaths, fmt.Sprintf("src/%s%s", name, lang.extensions[extIdx]))
			}
		}

		// Optionally add some unrecognized files (should not affect detection).
		numUnrecognized := rapid.IntRange(0, 5).Draw(t, "numUnrecognized")
		for i := 0; i < numUnrecognized; i++ {
			ext := rapid.SampledFrom([]string{".md", ".txt", ".json", ".yaml", ".xml", ".csv"}).Draw(t, fmt.Sprintf("unrecExt_%d", i))
			name := rapid.StringMatching(`[a-z]{1,10}`).Draw(t, fmt.Sprintf("unrecName_%d", i))
			filePaths = append(filePaths, fmt.Sprintf("docs/%s%s", name, ext))
		}

		// Call detectLanguage and verify it returns the dominant language.
		detected := detectLanguage(filePaths)
		if detected != dominant.id {
			t.Fatalf("detectLanguage() = %q, want %q (dominant had %d files)\nfilePaths: %v",
				detected, dominant.id, dominantCount, filePaths)
		}
	})
}
