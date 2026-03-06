package docgen

import (
	"testing"
)

func TestDocumentationType_String(t *testing.T) {
	tests := []struct {
		name     string
		docType  DocumentationType
		expected string
	}{
		{"UserManual", DocTypeUserManual, "User Manual"},
		{"Installation", DocTypeInstallation, "Installation Guide"},
		{"API", DocTypeAPI, "API Reference"},
		{"Tutorial", DocTypeTutorial, "Tutorial"},
		{"General", DocTypeGeneral, "General Documentation"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.docType.String()
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestDocumentationType_Filename(t *testing.T) {
	tests := []struct {
		name     string
		docType  DocumentationType
		expected string
	}{
		{"UserManual", DocTypeUserManual, "USER_MANUAL.md"},
		{"Installation", DocTypeInstallation, "INSTALLATION.md"},
		{"API", DocTypeAPI, "API_REFERENCE.md"},
		{"Tutorial", DocTypeTutorial, "TUTORIAL.md"},
		{"General", DocTypeGeneral, "DOCUMENTATION.md"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.docType.Filename()
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestParsedCommand_Structure(t *testing.T) {
	// Test that ParsedCommand can be instantiated with all fields
	cmd := ParsedCommand{
		IsProjectWide:   true,
		IsDocRequest:    true,
		NaturalLanguage: "create a user manual",
		ScopeFilters:    []string{"internal/", "cmd/"},
	}

	if !cmd.IsProjectWide {
		t.Error("Expected IsProjectWide to be true")
	}
	if !cmd.IsDocRequest {
		t.Error("Expected IsDocRequest to be true")
	}
	if cmd.NaturalLanguage != "create a user manual" {
		t.Errorf("Expected NaturalLanguage to be 'create a user manual', got %s", cmd.NaturalLanguage)
	}
	if len(cmd.ScopeFilters) != 2 {
		t.Errorf("Expected 2 scope filters, got %d", len(cmd.ScopeFilters))
	}
}

func TestClassificationResult_Structure(t *testing.T) {
	// Test that ClassificationResult can be instantiated
	result := ClassificationResult{
		Types:      []DocumentationType{DocTypeUserManual, DocTypeAPI},
		Confidence: 0.85,
	}

	if len(result.Types) != 2 {
		t.Errorf("Expected 2 types, got %d", len(result.Types))
	}
	if result.Confidence != 0.85 {
		t.Errorf("Expected confidence 0.85, got %f", result.Confidence)
	}
}

func TestAnalysisResult_Structure(t *testing.T) {
	// Test that AnalysisResult can be instantiated with all nested structures
	result := AnalysisResult{
		CodeStructure: &CodeStructure{
			Packages: []PackageInfo{
				{Name: "main", Path: ".", Description: "Main package"},
			},
			Functions: []FunctionInfo{
				{Name: "HelloWorld", Package: "main", Signature: "func HelloWorld()", Comment: "Prints hello"},
			},
			Classes: []ClassInfo{},
			Exports: []ExportInfo{},
		},
		Configuration: &ConfigInfo{
			PackageManifests: []string{"go.mod"},
			BuildScripts:     []string{"Makefile"},
			ConfigFiles:      []string{},
		},
		Documentation: &ExistingDocs{
			ReadmeContent: "# Sample Project",
			Comments:      []CommentBlock{},
			Docstrings:    []DocstringBlock{},
		},
		Dependencies: &DependencyInfo{
			Runtime: []Dependency{
				{Name: "testify", Version: "v1.8.4", Source: "go.mod"},
			},
			Build: []Dependency{},
		},
	}

	if result.CodeStructure == nil {
		t.Error("Expected CodeStructure to be non-nil")
	}
	if result.Configuration == nil {
		t.Error("Expected Configuration to be non-nil")
	}
	if result.Documentation == nil {
		t.Error("Expected Documentation to be non-nil")
	}
	if result.Dependencies == nil {
		t.Error("Expected Dependencies to be non-nil")
	}
	if len(result.CodeStructure.Packages) != 1 {
		t.Errorf("Expected 1 package, got %d", len(result.CodeStructure.Packages))
	}
}

func TestGeneratedDoc_Structure(t *testing.T) {
	// Test that GeneratedDoc can be instantiated
	doc := GeneratedDoc{
		Type:     DocTypeUserManual,
		Content:  "# User Manual\n\nContent here",
		Filename: "USER_MANUAL.md",
	}

	if doc.Type != DocTypeUserManual {
		t.Errorf("Expected type UserManual, got %v", doc.Type)
	}
	if doc.Filename != "USER_MANUAL.md" {
		t.Errorf("Expected filename USER_MANUAL.md, got %s", doc.Filename)
	}
}

func TestWriteResult_Structure(t *testing.T) {
	// Test that WriteResult can be instantiated
	result := WriteResult{
		Filename: "USER_MANUAL.md",
		Path:     "/path/to/USER_MANUAL.md",
		Existed:  false,
		Written:  true,
	}

	if result.Filename != "USER_MANUAL.md" {
		t.Errorf("Expected filename USER_MANUAL.md, got %s", result.Filename)
	}
	if result.Existed {
		t.Error("Expected Existed to be false")
	}
	if !result.Written {
		t.Error("Expected Written to be true")
	}
}
