package agentic

import (
	"path/filepath"
	"strings"
)

// LanguageConfig holds configuration for a supported language.
type LanguageConfig struct {
	Name        string   // e.g. "Go"
	Extensions  []string // e.g. [".go"]
	TestCommand string   // e.g. "go test ./..."
	LintCommand string   // e.g. "go vet ./..."
}

// defaultLanguageRegistry maps language identifiers to their configuration.
var defaultLanguageRegistry = map[string]LanguageConfig{
	"go":         {Name: "Go", Extensions: []string{".go"}, TestCommand: "go test ./...", LintCommand: "go vet ./..."},
	"python":     {Name: "Python", Extensions: []string{".py"}, TestCommand: "python -m pytest"},
	"bash":       {Name: "Bash", Extensions: []string{".sh", ".bash"}, TestCommand: "shellcheck"},
	"powershell": {Name: "PowerShell", Extensions: []string{".ps1"}, TestCommand: "Invoke-Pester"},
}

// detectLanguage examines the given file paths and returns the language
// identifier whose registered extensions have the highest file count.
// Returns an empty string if no recognized language is found.
func detectLanguage(filePaths []string) string {
	// Build a reverse map: extension → language identifier
	extToLang := make(map[string]string)
	for id, cfg := range defaultLanguageRegistry {
		for _, ext := range cfg.Extensions {
			extToLang[ext] = id
		}
	}

	// Count files per language
	langCount := make(map[string]int)
	for _, fp := range filePaths {
		ext := strings.ToLower(filepath.Ext(fp))
		if lang, ok := extToLang[ext]; ok {
			langCount[lang]++
		}
	}

	// Find the language with the most files
	bestLang := ""
	bestCount := 0
	for lang, count := range langCount {
		if count > bestCount {
			bestCount = count
			bestLang = lang
		}
	}

	return bestLang
}

// getTestCommand returns the test command for the given language identifier.
// Returns an empty string if the language is not in the registry.
func getTestCommand(lang string) string {
	if cfg, ok := defaultLanguageRegistry[lang]; ok {
		return cfg.TestCommand
	}
	return ""
}
