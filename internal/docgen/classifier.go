package docgen

import "strings"

// RequestClassifier maps natural language requests to specific documentation types
type RequestClassifier struct {
	patterns map[DocumentationType][]string
}

// NewRequestClassifier creates a new RequestClassifier with predefined keyword patterns
func NewRequestClassifier() *RequestClassifier {
	return &RequestClassifier{
		patterns: map[DocumentationType][]string{
			DocTypeUserManual: {
				"user manual",
				"manual",
				"how to use",
				"user guide",
			},
			DocTypeInstallation: {
				"installation",
				"setup",
				"install",
				"getting started",
			},
			DocTypeAPI: {
				"api",
				"reference",
				"function",
				"class",
				"method",
			},
			DocTypeTutorial: {
				"tutorial",
				"guide",
				"walkthrough",
				"step by step",
				"step-by-step",
				"example",
			},
		},
	}
}

// Classify analyzes natural language text and returns matching documentation types
// with a confidence score based on keyword matches
func (c *RequestClassifier) Classify(naturalLanguage string) *ClassificationResult {
	if naturalLanguage == "" {
		return &ClassificationResult{
			Types:      []DocumentationType{DocTypeGeneral},
			Confidence: 1.0,
		}
	}

	// Normalize input for case-insensitive matching
	normalized := strings.ToLower(strings.TrimSpace(naturalLanguage))

	// Track matches for each documentation type
	matches := make(map[DocumentationType]int)
	totalMatches := 0

	// Check each documentation type's patterns
	for docType, keywords := range c.patterns {
		for _, keyword := range keywords {
			if strings.Contains(normalized, strings.ToLower(keyword)) {
				matches[docType]++
				totalMatches++
			}
		}
	}

	// If no matches found, return General type
	if totalMatches == 0 {
		return &ClassificationResult{
			Types:      []DocumentationType{DocTypeGeneral},
			Confidence: 1.0,
		}
	}

	// Collect all matched types
	var types []DocumentationType
	for docType := range matches {
		types = append(types, docType)
	}

	// Calculate confidence based on number of keyword matches
	// More matches = higher confidence
	confidence := float64(totalMatches) / 10.0
	if confidence > 1.0 {
		confidence = 1.0
	}
	if confidence < 0.1 {
		confidence = 0.1
	}

	return &ClassificationResult{
		Types:      types,
		Confidence: confidence,
	}
}
