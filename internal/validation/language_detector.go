package validation

import (
	"path/filepath"
	"strings"
)

// LanguageDetector identifies programming languages from file extensions
type LanguageDetector struct {
	extensionMap map[string]Language
	configs      map[Language]LanguageConfig
}

// NewLanguageDetector creates a new LanguageDetector with default extension mappings
func NewLanguageDetector() *LanguageDetector {
	return &LanguageDetector{
		extensionMap: map[string]Language{
			".go": LanguageGo,
			".py": LanguagePython,
		},
		configs: make(map[Language]LanguageConfig),
	}
}

// NewLanguageDetectorWithConfig creates a new LanguageDetector with custom configurations
func NewLanguageDetectorWithConfig(configs []LanguageConfig) *LanguageDetector {
	ld := &LanguageDetector{
		extensionMap: make(map[string]Language),
		configs:      make(map[Language]LanguageConfig),
	}

	// Register each configuration
	for _, config := range configs {
		ld.RegisterLanguage(config)
	}

	return ld
}

// RegisterLanguage adds a new language configuration to the detector
func (ld *LanguageDetector) RegisterLanguage(config LanguageConfig) {
	lang := Language(strings.ToLower(config.Name))
	ld.configs[lang] = config

	// Register all extensions for this language
	for _, ext := range config.Extensions {
		// Ensure extension starts with a dot
		if !strings.HasPrefix(ext, ".") {
			ext = "." + ext
		}
		// Store in lowercase for case-insensitive matching
		ld.extensionMap[strings.ToLower(ext)] = lang
	}
}

// DetectLanguage identifies the programming language from a file path
// Returns LanguageUnsupported for unknown extensions
// Extension matching is case-insensitive
func (ld *LanguageDetector) DetectLanguage(filePath string) Language {
	ext := filepath.Ext(filePath)
	// Convert to lowercase for case-insensitive matching
	ext = strings.ToLower(ext)

	if lang, ok := ld.extensionMap[ext]; ok {
		return lang
	}

	return LanguageUnsupported
}

// GetSupportedLanguages returns a list of all supported languages
func (ld *LanguageDetector) GetSupportedLanguages() []Language {
	// Use a map to deduplicate languages
	langSet := make(map[Language]bool)
	for _, lang := range ld.extensionMap {
		if lang != LanguageUnsupported {
			langSet[lang] = true
		}
	}

	// Convert map to slice
	languages := make([]Language, 0, len(langSet))
	for lang := range langSet {
		languages = append(languages, lang)
	}

	return languages
}

// IsSupported checks if a language is supported
func (ld *LanguageDetector) IsSupported(language Language) bool {
	if language == LanguageUnsupported {
		return false
	}

	for _, lang := range ld.extensionMap {
		if lang == language {
			return true
		}
	}

	return false
}

// GetLanguageConfig returns the configuration for a specific language
func (ld *LanguageDetector) GetLanguageConfig(language Language) (LanguageConfig, bool) {
	config, ok := ld.configs[language]
	return config, ok
}

// GetAllConfigs returns all registered language configurations
func (ld *LanguageDetector) GetAllConfigs() map[Language]LanguageConfig {
	// Return a copy to prevent external modification
	configs := make(map[Language]LanguageConfig, len(ld.configs))
	for lang, config := range ld.configs {
		configs[lang] = config
	}
	return configs
}
