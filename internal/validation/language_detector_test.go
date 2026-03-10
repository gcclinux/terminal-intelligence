package validation

import (
	"testing"
)

func TestLanguageDetector_DetectLanguage(t *testing.T) {
	detector := NewLanguageDetector()

	tests := []struct {
		name     string
		filePath string
		want     Language
	}{
		{
			name:     "Go file with .go extension",
			filePath: "main.go",
			want:     LanguageGo,
		},
		{
			name:     "Python file with .py extension",
			filePath: "script.py",
			want:     LanguagePython,
		},
		{
			name:     "Go file with path",
			filePath: "internal/validation/detector.go",
			want:     LanguageGo,
		},
		{
			name:     "Python file with path",
			filePath: "tests/unit/test_validator.py",
			want:     LanguagePython,
		},
		{
			name:     "Unknown extension returns unsupported",
			filePath: "README.md",
			want:     LanguageUnsupported,
		},
		{
			name:     "JavaScript file returns unsupported",
			filePath: "app.js",
			want:     LanguageUnsupported,
		},
		{
			name:     "File without extension returns unsupported",
			filePath: "Makefile",
			want:     LanguageUnsupported,
		},
		{
			name:     "Case-insensitive matching - uppercase .GO",
			filePath: "main.GO",
			want:     LanguageGo,
		},
		{
			name:     "Case-insensitive matching - uppercase .PY",
			filePath: "script.PY",
			want:     LanguagePython,
		},
		{
			name:     "Case-insensitive matching - mixed case .Go",
			filePath: "handler.Go",
			want:     LanguageGo,
		},
		{
			name:     "Case-insensitive matching - mixed case .Py",
			filePath: "utils.Py",
			want:     LanguagePython,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detector.DetectLanguage(tt.filePath)
			if got != tt.want {
				t.Errorf("DetectLanguage(%q) = %v, want %v", tt.filePath, got, tt.want)
			}
		})
	}
}

func TestLanguageDetector_GetSupportedLanguages(t *testing.T) {
	detector := NewLanguageDetector()

	languages := detector.GetSupportedLanguages()

	// Should return exactly 2 languages: Go and Python
	if len(languages) != 2 {
		t.Errorf("GetSupportedLanguages() returned %d languages, want 2", len(languages))
	}

	// Check that both Go and Python are in the list
	hasGo := false
	hasPython := false
	for _, lang := range languages {
		if lang == LanguageGo {
			hasGo = true
		}
		if lang == LanguagePython {
			hasPython = true
		}
		if lang == LanguageUnsupported {
			t.Errorf("GetSupportedLanguages() should not include LanguageUnsupported")
		}
	}

	if !hasGo {
		t.Errorf("GetSupportedLanguages() missing LanguageGo")
	}
	if !hasPython {
		t.Errorf("GetSupportedLanguages() missing LanguagePython")
	}
}

func TestLanguageDetector_IsSupported(t *testing.T) {
	detector := NewLanguageDetector()

	tests := []struct {
		name     string
		language Language
		want     bool
	}{
		{
			name:     "Go is supported",
			language: LanguageGo,
			want:     true,
		},
		{
			name:     "Python is supported",
			language: LanguagePython,
			want:     true,
		},
		{
			name:     "Unsupported language is not supported",
			language: LanguageUnsupported,
			want:     false,
		},
		{
			name:     "Unknown language is not supported",
			language: Language("javascript"),
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detector.IsSupported(tt.language)
			if got != tt.want {
				t.Errorf("IsSupported(%v) = %v, want %v", tt.language, got, tt.want)
			}
		})
	}
}

// Test configuration system
func TestLanguageDetector_NewLanguageDetectorWithConfig(t *testing.T) {
	config := LanguageConfig{
		Name:       "JavaScript",
		Extensions: []string{".js", ".jsx"},
		Validator: ValidatorConfig{
			Command: "eslint",
			Args:    []string{"--format", "json"},
			Timeout: 30000000000,
		},
	}

	detector := NewLanguageDetectorWithConfig([]LanguageConfig{config})

	// Test .js extension
	lang := detector.DetectLanguage("test.js")
	if lang != Language("javascript") {
		t.Errorf("Expected 'javascript', got '%s'", lang)
	}

	// Test .jsx extension
	lang = detector.DetectLanguage("component.jsx")
	if lang != Language("javascript") {
		t.Errorf("Expected 'javascript', got '%s'", lang)
	}

	// Test that JavaScript is supported
	if !detector.IsSupported(Language("javascript")) {
		t.Error("Expected JavaScript to be supported")
	}
}

func TestLanguageDetector_RegisterLanguage(t *testing.T) {
	detector := NewLanguageDetector()

	// Initially, TypeScript should not be supported
	if detector.IsSupported(Language("typescript")) {
		t.Error("TypeScript should not be supported initially")
	}

	// Register TypeScript
	config := LanguageConfig{
		Name:       "TypeScript",
		Extensions: []string{".ts", ".tsx"},
		Validator: ValidatorConfig{
			Command: "tsc",
			Args:    []string{"--noEmit"},
			Timeout: 30000000000,
		},
	}
	detector.RegisterLanguage(config)

	// Now TypeScript should be supported
	if !detector.IsSupported(Language("typescript")) {
		t.Error("TypeScript should be supported after registration")
	}

	// Test detection
	lang := detector.DetectLanguage("app.ts")
	if lang != Language("typescript") {
		t.Errorf("Expected 'typescript', got '%s'", lang)
	}
}

func TestLanguageDetector_RegisterLanguage_WithoutDot(t *testing.T) {
	detector := NewLanguageDetector()

	// Register with extensions without leading dot
	config := LanguageConfig{
		Name:       "Ruby",
		Extensions: []string{"rb", "rake"},
		Validator: ValidatorConfig{
			Command: "ruby",
			Args:    []string{"-c"},
			Timeout: 10000000000,
		},
	}
	detector.RegisterLanguage(config)

	// Should still detect correctly
	lang := detector.DetectLanguage("script.rb")
	if lang != Language("ruby") {
		t.Errorf("Expected 'ruby', got '%s'", lang)
	}

	lang = detector.DetectLanguage("Rakefile.rake")
	if lang != Language("ruby") {
		t.Errorf("Expected 'ruby', got '%s'", lang)
	}
}

func TestLanguageDetector_GetLanguageConfig(t *testing.T) {
	config := LanguageConfig{
		Name:       "Rust",
		Extensions: []string{".rs"},
		Validator: ValidatorConfig{
			Command: "rustc",
			Args:    []string{"--crate-type", "lib"},
			Timeout: 60000000000,
		},
	}

	detector := NewLanguageDetectorWithConfig([]LanguageConfig{config})

	// Get config for Rust
	retrievedConfig, ok := detector.GetLanguageConfig(Language("rust"))
	if !ok {
		t.Fatal("Expected to find Rust configuration")
	}

	if retrievedConfig.Name != "Rust" {
		t.Errorf("Expected name 'Rust', got '%s'", retrievedConfig.Name)
	}

	if retrievedConfig.Validator.Command != "rustc" {
		t.Errorf("Expected command 'rustc', got '%s'", retrievedConfig.Validator.Command)
	}

	// Try to get config for non-existent language
	_, ok = detector.GetLanguageConfig(Language("nonexistent"))
	if ok {
		t.Error("Expected not to find configuration for non-existent language")
	}
}

func TestLanguageDetector_GetAllConfigs(t *testing.T) {
	configs := []LanguageConfig{
		{
			Name:       "JavaScript",
			Extensions: []string{".js"},
			Validator: ValidatorConfig{
				Command: "eslint",
				Timeout: 30000000000,
			},
		},
		{
			Name:       "TypeScript",
			Extensions: []string{".ts"},
			Validator: ValidatorConfig{
				Command: "tsc",
				Timeout: 30000000000,
			},
		},
	}

	detector := NewLanguageDetectorWithConfig(configs)

	allConfigs := detector.GetAllConfigs()

	if len(allConfigs) != 2 {
		t.Errorf("Expected 2 configurations, got %d", len(allConfigs))
	}

	// Verify JavaScript config
	jsConfig, ok := allConfigs[Language("javascript")]
	if !ok {
		t.Error("Expected to find JavaScript configuration")
	}
	if jsConfig.Name != "JavaScript" {
		t.Errorf("Expected JavaScript name, got '%s'", jsConfig.Name)
	}

	// Verify TypeScript config
	tsConfig, ok := allConfigs[Language("typescript")]
	if !ok {
		t.Error("Expected to find TypeScript configuration")
	}
	if tsConfig.Name != "TypeScript" {
		t.Errorf("Expected TypeScript name, got '%s'", tsConfig.Name)
	}
}

func TestLanguageDetector_MultipleLanguages(t *testing.T) {
	configs := []LanguageConfig{
		{
			Name:       "JavaScript",
			Extensions: []string{".js", ".jsx"},
			Validator:  ValidatorConfig{Command: "eslint"},
		},
		{
			Name:       "TypeScript",
			Extensions: []string{".ts", ".tsx"},
			Validator:  ValidatorConfig{Command: "tsc"},
		},
		{
			Name:       "Ruby",
			Extensions: []string{".rb"},
			Validator:  ValidatorConfig{Command: "ruby"},
		},
	}

	detector := NewLanguageDetectorWithConfig(configs)

	tests := []struct {
		file     string
		expected Language
	}{
		{"app.js", Language("javascript")},
		{"component.jsx", Language("javascript")},
		{"main.ts", Language("typescript")},
		{"Component.tsx", Language("typescript")},
		{"script.rb", Language("ruby")},
		{"unknown.xyz", LanguageUnsupported},
	}

	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			lang := detector.DetectLanguage(tt.file)
			if lang != tt.expected {
				t.Errorf("For file '%s', expected '%s', got '%s'", tt.file, tt.expected, lang)
			}
		})
	}
}

func TestLanguageDetector_CaseInsensitiveConfig(t *testing.T) {
	config := LanguageConfig{
		Name:       "Rust",
		Extensions: []string{".rs"},
		Validator:  ValidatorConfig{Command: "rustc"},
	}

	detector := NewLanguageDetectorWithConfig([]LanguageConfig{config})

	tests := []struct {
		file string
	}{
		{"main.rs"},
		{"main.RS"},
		{"main.Rs"},
		{"main.rS"},
	}

	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			lang := detector.DetectLanguage(tt.file)
			if lang != Language("rust") {
				t.Errorf("For file '%s', expected 'rust', got '%s'", tt.file, lang)
			}
		})
	}
}

func TestLanguageDetector_InvalidConfiguration(t *testing.T) {
	// Test with empty extensions
	config := LanguageConfig{
		Name:       "EmptyLang",
		Extensions: []string{},
		Validator:  ValidatorConfig{Command: "test"},
	}

	detector := NewLanguageDetectorWithConfig([]LanguageConfig{config})

	// Should not detect any files as EmptyLang
	lang := detector.DetectLanguage("test.txt")
	if lang == Language("emptylang") {
		t.Error("Should not detect files for language with empty extensions")
	}
}
