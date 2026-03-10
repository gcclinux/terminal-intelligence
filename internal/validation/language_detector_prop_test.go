package validation

import (
	"fmt"
	"strings"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// Feature: automatic-code-validation, Property 3: Language Detection from Extension
// **Validates: Requirements 2.1**
//
// For any file path with a supported extension (.go or .py), the LanguageDetector
// correctly identifies the programming language.
func TestProperty3_LanguageDetectionFromExtension(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("files with .go extension are detected as Go", prop.ForAll(
		func(pathComponents []string, caseVariant string) bool {
			detector := NewLanguageDetector()

			// Build file path with .go extension (with case variant)
			filePath := buildFilePath(pathComponents, caseVariant)

			// Detect language
			language := detector.DetectLanguage(filePath)

			// Should always detect as Go
			return language == LanguageGo
		},
		genPathComponents(),
		genGoExtension(),
	))

	properties.Property("files with .py extension are detected as Python", prop.ForAll(
		func(pathComponents []string, caseVariant string) bool {
			detector := NewLanguageDetector()

			// Build file path with .py extension (with case variant)
			filePath := buildFilePath(pathComponents, caseVariant)

			// Detect language
			language := detector.DetectLanguage(filePath)

			// Should always detect as Python
			return language == LanguagePython
		},
		genPathComponents(),
		genPyExtension(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// genPathComponents generates random path components for file paths
// Returns a slice of path segments that can be joined to form a file path
func genPathComponents() gopter.Gen {
	return gen.SliceOf(genPathSegment()).
		SuchThat(func(v interface{}) bool {
			segments := v.([]string)
			// Ensure at least one segment (the filename without extension)
			return len(segments) >= 1
		})
}

// genPathSegment generates a valid path segment (directory or filename without extension)
func genPathSegment() gopter.Gen {
	return gen.Identifier().
		SuchThat(func(v interface{}) bool {
			s := v.(string)
			// Ensure non-empty and doesn't contain path separators or dots
			return len(s) > 0 && !strings.ContainsAny(s, "/\\.")
		})
}

// genGoExtension generates case variants of the .go extension
func genGoExtension() gopter.Gen {
	return gen.OneConstOf(
		".go",
		".Go",
		".GO",
		".gO",
	)
}

// genPyExtension generates case variants of the .py extension
func genPyExtension() gopter.Gen {
	return gen.OneConstOf(
		".py",
		".Py",
		".PY",
		".pY",
	)
}

// buildFilePath constructs a file path from components and adds the extension
func buildFilePath(components []string, extension string) string {
	if len(components) == 0 {
		return "file" + extension
	}

	// Join components with forward slashes (works on all platforms for testing)
	path := strings.Join(components, "/")

	// Add extension to the last component
	return fmt.Sprintf("%s%s", path, extension)
}

// Feature: automatic-code-validation, Property 5: Unsupported Language Reporting
// **Validates: Requirements 2.5**
//
// For any file path with an unsupported or unknown extension, the LanguageDetector
// reports the file as unsupported.
func TestProperty5_UnsupportedLanguageReporting(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("files with unsupported extensions are reported as unsupported", prop.ForAll(
		func(pathComponents []string, extension string) bool {
			detector := NewLanguageDetector()

			// Build file path with unsupported extension
			filePath := buildFilePath(pathComponents, extension)

			// Detect language
			language := detector.DetectLanguage(filePath)

			// Should always report as unsupported
			return language == LanguageUnsupported
		},
		genPathComponents(),
		genUnsupportedExtension(),
	))

	properties.Property("files without extensions are reported as unsupported", prop.ForAll(
		func(pathComponents []string) bool {
			detector := NewLanguageDetector()

			// Build file path without extension
			filePath := strings.Join(pathComponents, "/")

			// Detect language
			language := detector.DetectLanguage(filePath)

			// Should always report as unsupported
			return language == LanguageUnsupported
		},
		genPathComponents(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// genUnsupportedExtension generates random unsupported file extensions
// Ensures the extension is not .go or .py (case-insensitive)
func genUnsupportedExtension() gopter.Gen {
	// Generate common unsupported extensions
	commonUnsupported := gen.OneConstOf(
		".js", ".ts", ".jsx", ".tsx",
		".java", ".cpp", ".c", ".h",
		".rb", ".php", ".rs", ".swift",
		".kt", ".scala", ".cs", ".vb",
		".txt", ".md", ".json", ".yaml", ".yml",
		".xml", ".html", ".css", ".scss",
		".sh", ".bat", ".ps1",
		".sql", ".db",
		".jpg", ".png", ".gif", ".svg",
		".pdf", ".doc", ".docx",
	)

	// Also generate random extensions
	randomExtension := gen.Identifier().Map(func(s string) string {
		return "." + s
	}).SuchThat(func(v interface{}) bool {
		ext := strings.ToLower(v.(string))
		// Ensure it's not a supported extension
		return ext != ".go" && ext != ".py"
	})

	// Combine both generators
	return gen.OneGenOf(commonUnsupported, randomExtension)
}

// Feature: automatic-code-validation, Property 4: Configured Language Detection
// **Validates: Requirements 2.4**
//
// For any language configuration added to the system, the LanguageDetector
// correctly identifies files with that language's extensions.
func TestProperty4_ConfiguredLanguageDetection(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("configured languages are detected correctly", prop.ForAll(
		func(config LanguageConfig, pathComponents []string) bool {
			// Create detector with configuration
			detector := NewLanguageDetectorWithConfig([]LanguageConfig{config})

			// Test each extension
			for _, ext := range config.Extensions {
				// Build file path with this extension
				filePath := buildFilePath(pathComponents, ext)

				// Detect language
				language := detector.DetectLanguage(filePath)

				// Should detect as the configured language
				expectedLang := Language(strings.ToLower(config.Name))
				if language != expectedLang {
					return false
				}
			}

			return true
		},
		genLanguageConfig(),
		genPathComponents(),
	))

	properties.Property("multiple configured languages coexist", prop.ForAll(
		func(configs []LanguageConfig, pathComponents []string) bool {
			// Create detector with multiple configurations
			detector := NewLanguageDetectorWithConfig(configs)

			// Test each configuration
			for _, config := range configs {
				expectedLang := Language(strings.ToLower(config.Name))

				// Test each extension for this language
				for _, ext := range config.Extensions {
					filePath := buildFilePath(pathComponents, ext)
					language := detector.DetectLanguage(filePath)

					if language != expectedLang {
						return false
					}
				}
			}

			return true
		},
		genLanguageConfigList(),
		genPathComponents(),
	))

	properties.Property("configured languages are reported as supported", prop.ForAll(
		func(config LanguageConfig) bool {
			detector := NewLanguageDetectorWithConfig([]LanguageConfig{config})

			expectedLang := Language(strings.ToLower(config.Name))

			// Language should be supported
			if !detector.IsSupported(expectedLang) {
				return false
			}

			// Language should appear in supported languages list
			supportedLangs := detector.GetSupportedLanguages()
			found := false
			for _, lang := range supportedLangs {
				if lang == expectedLang {
					found = true
					break
				}
			}

			return found
		},
		genLanguageConfig(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// genLanguageName generates random language names
func genLanguageName() gopter.Gen {
	return gen.OneConstOf(
		"JavaScript", "TypeScript", "Ruby", "Rust",
		"Java", "Kotlin", "Swift", "CSharp",
		"PHP", "Perl", "Scala", "Haskell",
	)
}

// genExtensionList generates a list of 1-2 file extensions
func genExtensionList() gopter.Gen {
	// Generate 1 or 2 extensions
	return gen.OneGenOf(
		// Single extension
		genCustomExtension().Map(func(ext string) []string {
			return []string{ext}
		}),
		// Two extensions
		gopter.CombineGens(
			genCustomExtension(),
			genCustomExtension(),
		).Map(func(values []interface{}) []string {
			ext1 := values[0].(string)
			ext2 := values[1].(string)
			// Ensure they're different
			if ext1 == ext2 {
				return []string{ext1}
			}
			return []string{ext1, ext2}
		}),
	)
}

// genCustomExtension generates custom file extensions
func genCustomExtension() gopter.Gen {
	return gen.OneConstOf(
		".js", ".jsx", ".ts", ".tsx",
		".rb", ".rs", ".java", ".kt",
		".swift", ".cs", ".php", ".pl",
		".scala", ".hs",
	)
}

// genLanguageConfig generates a random LanguageConfig
func genLanguageConfig() gopter.Gen {
	return gopter.CombineGens(
		genLanguageName(),
		genExtensionList(),
	).Map(func(values []interface{}) LanguageConfig {
		return LanguageConfig{
			Name:       values[0].(string),
			Extensions: values[1].([]string),
			Validator: ValidatorConfig{
				Command: "test-validator",
				Args:    []string{},
				Timeout: 30000000000, // 30 seconds
			},
		}
	})
}

// genLanguageConfigList generates a list of 1-3 unique LanguageConfig objects
func genLanguageConfigList() gopter.Gen {
	// Predefined unique configs to avoid conflicts
	configs := []LanguageConfig{
		{
			Name:       "JavaScript",
			Extensions: []string{".js", ".jsx"},
			Validator:  ValidatorConfig{Command: "eslint", Timeout: 30000000000},
		},
		{
			Name:       "TypeScript",
			Extensions: []string{".ts", ".tsx"},
			Validator:  ValidatorConfig{Command: "tsc", Timeout: 30000000000},
		},
		{
			Name:       "Ruby",
			Extensions: []string{".rb"},
			Validator:  ValidatorConfig{Command: "ruby", Timeout: 30000000000},
		},
		{
			Name:       "Rust",
			Extensions: []string{".rs"},
			Validator:  ValidatorConfig{Command: "rustc", Timeout: 30000000000},
		},
	}

	// Generate combinations of 1-3 configs
	return gen.OneGenOf(
		// Single config
		gen.IntRange(0, 3).Map(func(i int) []LanguageConfig {
			return []LanguageConfig{configs[i]}
		}),
		// Two configs
		gopter.CombineGens(
			gen.IntRange(0, 3),
			gen.IntRange(0, 3),
		).Map(func(values []interface{}) []LanguageConfig {
			i := values[0].(int)
			j := values[1].(int)
			if i == j {
				return []LanguageConfig{configs[i]}
			}
			return []LanguageConfig{configs[i], configs[j]}
		}),
		// Three configs
		gopter.CombineGens(
			gen.IntRange(0, 3),
			gen.IntRange(0, 3),
			gen.IntRange(0, 3),
		).Map(func(values []interface{}) []LanguageConfig {
			i := values[0].(int)
			j := values[1].(int)
			k := values[2].(int)
			// Use a map to deduplicate
			seen := make(map[int]bool)
			result := []LanguageConfig{}
			for _, idx := range []int{i, j, k} {
				if !seen[idx] {
					result = append(result, configs[idx])
					seen[idx] = true
				}
			}
			return result
		}),
	)
}
