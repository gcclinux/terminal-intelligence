package docgen

import (
	"testing"
)

func TestRequestClassifier_Classify_UserManual(t *testing.T) {
	classifier := NewRequestClassifier()

	tests := []struct {
		name  string
		input string
	}{
		{"user manual keyword", "create user manual"},
		{"manual keyword", "generate a manual"},
		{"how to use keyword", "how to use this application"},
		{"user guide keyword", "create a user guide"},
		{"mixed case", "Create User Manual"},
		{"uppercase", "USER MANUAL"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifier.Classify(tt.input)
			if len(result.Types) == 0 {
				t.Fatalf("Classify() returned no types, want at least one")
			}
			found := false
			for _, docType := range result.Types {
				if docType == DocTypeUserManual {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Classify() types = %v, want to include DocTypeUserManual", result.Types)
			}
			if result.Confidence <= 0 || result.Confidence > 1.0 {
				t.Errorf("Classify() confidence = %f, want between 0 and 1.0", result.Confidence)
			}
		})
	}
}

func TestRequestClassifier_Classify_Installation(t *testing.T) {
	classifier := NewRequestClassifier()

	tests := []struct {
		name  string
		input string
	}{
		{"installation keyword", "create installation guide"},
		{"setup keyword", "generate setup instructions"},
		{"install keyword", "how to install"},
		{"getting started keyword", "getting started documentation"},
		{"mixed case", "Installation and Setup"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifier.Classify(tt.input)
			if len(result.Types) == 0 {
				t.Fatalf("Classify() returned no types, want at least one")
			}
			found := false
			for _, docType := range result.Types {
				if docType == DocTypeInstallation {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Classify() types = %v, want to include DocTypeInstallation", result.Types)
			}
		})
	}
}

func TestRequestClassifier_Classify_API(t *testing.T) {
	classifier := NewRequestClassifier()

	tests := []struct {
		name  string
		input string
	}{
		{"API keyword", "create API documentation"},
		{"reference keyword", "generate reference docs"},
		{"function keyword", "document all functions"},
		{"class keyword", "document classes"},
		{"method keyword", "document methods"},
		{"mixed case", "API Reference"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifier.Classify(tt.input)
			if len(result.Types) == 0 {
				t.Fatalf("Classify() returned no types, want at least one")
			}
			found := false
			for _, docType := range result.Types {
				if docType == DocTypeAPI {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Classify() types = %v, want to include DocTypeAPI", result.Types)
			}
		})
	}
}

func TestRequestClassifier_Classify_Tutorial(t *testing.T) {
	classifier := NewRequestClassifier()

	tests := []struct {
		name  string
		input string
	}{
		{"tutorial keyword", "create a tutorial"},
		{"guide keyword", "generate a guide"},
		{"walkthrough keyword", "create a walkthrough"},
		{"example keyword", "provide examples"},
		{"mixed case", "Tutorial Guide"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifier.Classify(tt.input)
			if len(result.Types) == 0 {
				t.Fatalf("Classify() returned no types, want at least one")
			}
			found := false
			for _, docType := range result.Types {
				if docType == DocTypeTutorial {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Classify() types = %v, want to include DocTypeTutorial", result.Types)
			}
		})
	}
}

func TestRequestClassifier_Classify_General(t *testing.T) {
	classifier := NewRequestClassifier()

	tests := []struct {
		name  string
		input string
	}{
		{"no keywords", "create documentation"},
		{"unrelated text", "hello world"},
		{"random text", "something completely different"},
		{"empty string", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifier.Classify(tt.input)
			if len(result.Types) == 0 {
				t.Fatalf("Classify() returned no types, want at least one")
			}
			// Should return General type when no keywords match
			if result.Types[0] != DocTypeGeneral {
				t.Errorf("Classify() types[0] = %v, want DocTypeGeneral", result.Types[0])
			}
			if result.Confidence != 1.0 {
				t.Errorf("Classify() confidence = %f, want 1.0 for General type", result.Confidence)
			}
		})
	}
}

func TestRequestClassifier_Classify_MultipleTypes(t *testing.T) {
	classifier := NewRequestClassifier()

	tests := []struct {
		name      string
		input     string
		wantTypes []DocumentationType
	}{
		{
			"installation and API",
			"create installation guide and API reference",
			[]DocumentationType{DocTypeInstallation, DocTypeAPI},
		},
		{
			"user manual and tutorial",
			"generate user manual with tutorial examples",
			[]DocumentationType{DocTypeUserManual, DocTypeTutorial},
		},
		{
			"all types",
			"create user manual, installation guide, API reference, and tutorial",
			[]DocumentationType{DocTypeUserManual, DocTypeInstallation, DocTypeAPI, DocTypeTutorial},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifier.Classify(tt.input)
			if len(result.Types) < len(tt.wantTypes) {
				t.Errorf("Classify() returned %d types, want at least %d", len(result.Types), len(tt.wantTypes))
			}

			// Check that all expected types are present
			for _, wantType := range tt.wantTypes {
				found := false
				for _, gotType := range result.Types {
					if gotType == wantType {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Classify() types = %v, want to include %v", result.Types, wantType)
				}
			}
		})
	}
}

func TestRequestClassifier_Classify_ConfidenceScores(t *testing.T) {
	classifier := NewRequestClassifier()

	tests := []struct {
		name              string
		input             string
		wantMinConfidence float64
		wantMaxConfidence float64
	}{
		{"single keyword", "create manual", 0.1, 0.2},
		{"multiple keywords", "create user manual guide", 0.2, 0.4},
		{"many keywords", "installation setup install getting started", 0.4, 1.0},
		{"no keywords", "create documentation", 1.0, 1.0}, // General type has confidence 1.0
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifier.Classify(tt.input)
			if result.Confidence < tt.wantMinConfidence || result.Confidence > tt.wantMaxConfidence {
				t.Errorf("Classify() confidence = %f, want between %f and %f",
					result.Confidence, tt.wantMinConfidence, tt.wantMaxConfidence)
			}
		})
	}
}

func TestRequestClassifier_Classify_KeywordVariations(t *testing.T) {
	classifier := NewRequestClassifier()

	tests := []struct {
		name     string
		input    string
		wantType DocumentationType
	}{
		{"capitalized", "Create User Manual", DocTypeUserManual},
		{"uppercase", "INSTALLATION GUIDE", DocTypeInstallation},
		{"mixed case", "Api ReFeReNcE", DocTypeAPI},
		{"with punctuation", "create a tutorial!", DocTypeTutorial},
		{"with extra spaces", "user  manual  documentation", DocTypeUserManual},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifier.Classify(tt.input)
			found := false
			for _, docType := range result.Types {
				if docType == tt.wantType {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Classify() types = %v, want to include %v", result.Types, tt.wantType)
			}
		})
	}
}

// TestRequestClassifier_SpecificExamples tests the exact examples from Task 3.3
// Validates Requirements 2.1, 2.2, 2.3, 2.4, 2.5
func TestRequestClassifier_SpecificExamples(t *testing.T) {
	classifier := NewRequestClassifier()

	tests := []struct {
		name     string
		input    string
		wantType DocumentationType
	}{
		// Task 3.3 specific examples
		{"create user manual", "create user manual", DocTypeUserManual},
		{"installation and setup guide", "installation and setup guide", DocTypeInstallation},
		{"API reference documentation", "API reference documentation", DocTypeAPI},

		// Keyword variations - capitalization
		{"User Manual capitalized", "User Manual", DocTypeUserManual},
		{"INSTALLATION uppercase", "INSTALLATION", DocTypeInstallation},
		{"api lowercase", "api", DocTypeAPI},
		{"Setup mixed case", "Setup", DocTypeInstallation},

		// Keyword variations - plurals
		{"manuals plural", "create manuals", DocTypeUserManual},
		{"installations plural", "installations guide", DocTypeInstallation},
		{"APIs plural", "APIs documentation", DocTypeAPI},
		{"tutorials plural", "create tutorials", DocTypeTutorial},
		{"guides plural", "create guides", DocTypeTutorial},

		// Additional variations
		{"reference singular", "reference documentation", DocTypeAPI},
		{"references plural", "references documentation", DocTypeAPI},
		{"function singular", "function documentation", DocTypeAPI},
		{"functions plural", "functions documentation", DocTypeAPI},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifier.Classify(tt.input)

			if len(result.Types) == 0 {
				t.Fatalf("Classify(%q) returned no types, want at least one", tt.input)
			}

			found := false
			for _, docType := range result.Types {
				if docType == tt.wantType {
					found = true
					break
				}
			}

			if !found {
				t.Errorf("Classify(%q) types = %v, want to include %v", tt.input, result.Types, tt.wantType)
			}

			// Verify confidence is in valid range
			if result.Confidence <= 0 || result.Confidence > 1.0 {
				t.Errorf("Classify(%q) confidence = %f, want between 0 and 1.0", tt.input, result.Confidence)
			}
		})
	}
}
