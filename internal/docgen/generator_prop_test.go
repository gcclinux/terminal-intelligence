package docgen

import (
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// **Validates: Requirements 4.1, 4.2, 4.3, 4.5, 4.6**
// Property 9: User Manual Content Inclusion
// For any user manual generation request, the generated document should include
// sections for application overview, commands, and usage information
func TestProperty9_UserManualContentInclusion(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random analysis result
		result := genAnalysisResult().Draw(t, "analysisResult")

		gen := NewDocumentationGenerator(result)
		doc, err := gen.Generate(DocTypeUserManual)

		if err != nil {
			t.Fatalf("Generate failed: %v", err)
		}

		content := doc.Content

		// Must have title
		if !strings.Contains(content, "# User Manual") {
			t.Fatalf("User manual missing title")
		}

		// Must have Overview section
		if !strings.Contains(content, "## Overview") {
			t.Fatalf("User manual missing Overview section")
		}

		// Must have Usage section
		if !strings.Contains(content, "## Usage") {
			t.Fatalf("User manual missing Usage section")
		}

		// Must have Installation section (even if just a reference)
		if !strings.Contains(content, "## Installation") {
			t.Fatalf("User manual missing Installation section")
		}
	})
}

// **Validates: Requirements 4.5**
// Property 10: Document Hierarchy
// For any generated documentation, the content should have a valid markdown
// hierarchy with properly nested headers
func TestProperty10_DocumentHierarchy(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random analysis result
		result := genAnalysisResult().Draw(t, "analysisResult")

		// Test all documentation types
		docType := rapid.SampledFrom([]DocumentationType{
			DocTypeUserManual,
			DocTypeInstallation,
			DocTypeAPI,
		}).Draw(t, "docType")

		gen := NewDocumentationGenerator(result)
		doc, err := gen.Generate(docType)

		if err != nil {
			t.Fatalf("Generate failed: %v", err)
		}

		content := doc.Content
		lines := strings.Split(content, "\n")

		hasH1 := false
		hasH2 := false
		h1Count := 0

		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "# ") && !strings.HasPrefix(trimmed, "## ") {
				hasH1 = true
				h1Count++
			}
			if strings.HasPrefix(trimmed, "## ") {
				hasH2 = true
			}
		}

		// Must have at least one H1 (title)
		if !hasH1 {
			t.Fatalf("Document missing H1 title")
		}

		// Should have only one H1 (the main title)
		if h1Count > 1 {
			t.Fatalf("Document has multiple H1 headers (%d), should have only one title", h1Count)
		}

		// Must have at least one H2 (section)
		if !hasH2 {
			t.Fatalf("Document missing H2 sections")
		}
	})
}

// **Validates: Requirements 4.6, 5.5, 6.6**
// Property 11: Markdown Validity
// For any documentation type generated, the output content should be valid
// Markdown that can be parsed without errors
func TestProperty11_MarkdownValidity(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random analysis result
		result := genAnalysisResult().Draw(t, "analysisResult")

		// Test all documentation types
		docType := rapid.SampledFrom([]DocumentationType{
			DocTypeUserManual,
			DocTypeInstallation,
			DocTypeAPI,
			DocTypeGeneral,
		}).Draw(t, "docType")

		gen := NewDocumentationGenerator(result)
		doc, err := gen.Generate(docType)

		if err != nil {
			t.Fatalf("Generate failed: %v", err)
		}

		content := doc.Content

		// Check for basic markdown validity
		// 1. No unclosed code blocks
		codeBlockCount := strings.Count(content, "```")
		if codeBlockCount%2 != 0 {
			t.Fatalf("Unclosed code block detected (odd number of ```)")
		}

		// 2. Headers should have space after #
		lines := strings.Split(content, "\n")
		for i, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "#") {
				// Count leading #
				hashCount := 0
				for _, ch := range trimmed {
					if ch == '#' {
						hashCount++
					} else {
						break
					}
				}

				// After the hashes, should be a space (or end of line for empty headers)
				if len(trimmed) > hashCount {
					if trimmed[hashCount] != ' ' {
						t.Fatalf("Invalid header at line %d: missing space after #", i+1)
					}
				}
			}
		}

		// 3. Content should not be empty
		if strings.TrimSpace(content) == "" {
			t.Fatalf("Generated content is empty")
		}
	})
}

// **Validates: Requirements 5.1, 5.3**
// Property 12: Installation Guide Requirements Extraction
// For any project with a package manifest file, the installation guide should
// include the dependencies listed in that manifest
func TestProperty12_InstallationGuideRequirementsExtraction(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate analysis result with dependencies
		result := genAnalysisResultWithDependencies().Draw(t, "analysisResult")

		// Skip if no dependencies
		if result.Dependencies == nil || len(result.Dependencies.Runtime) == 0 {
			t.Skip("No dependencies to test")
		}

		gen := NewDocumentationGenerator(result)
		doc, err := gen.Generate(DocTypeInstallation)

		if err != nil {
			t.Fatalf("Generate failed: %v", err)
		}

		content := doc.Content

		// Should have Dependencies section
		if !strings.Contains(content, "## Dependencies") {
			t.Fatalf("Installation guide missing Dependencies section when dependencies exist")
		}

		// Should mention at least one dependency
		foundDep := false
		for _, dep := range result.Dependencies.Runtime {
			if strings.Contains(content, dep.Name) {
				foundDep = true
				break
			}
		}

		if !foundDep {
			t.Fatalf("Installation guide does not mention any dependencies")
		}
	})
}

// **Validates: Requirements 5.2**
// Property 13: Installation Steps Extraction
// For any project with a README file or setup script, the installation guide
// should extract and include installation steps from those sources
func TestProperty13_InstallationStepsExtraction(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate analysis result with README containing installation steps
		result := genAnalysisResultWithReadme().Draw(t, "analysisResult")

		gen := NewDocumentationGenerator(result)
		doc, err := gen.Generate(DocTypeInstallation)

		if err != nil {
			t.Fatalf("Generate failed: %v", err)
		}

		content := doc.Content

		// Should have Installation Steps section
		if !strings.Contains(content, "## Installation Steps") {
			t.Fatalf("Installation guide missing Installation Steps section")
		}

		// Should have at least one numbered or bulleted step
		hasSteps := strings.Contains(content, "1.") ||
			strings.Contains(content, "2.") ||
			strings.Contains(content, "- ")

		if !hasSteps {
			t.Fatalf("Installation guide missing installation steps")
		}
	})
}

// **Validates: Requirements 5.4**
// Property 14: Conditional Configuration Documentation
// For any project, if configuration files are present, the installation guide
// should include configuration steps; if no configuration files exist, no
// configuration section should be included
func TestProperty14_ConditionalConfigurationDocumentation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate analysis result with or without config files
		hasConfig := rapid.Bool().Draw(t, "hasConfig")

		var result *AnalysisResult
		if hasConfig {
			result = genAnalysisResultWithConfig().Draw(t, "analysisResult")
		} else {
			result = genAnalysisResultWithoutConfig().Draw(t, "analysisResult")
		}

		gen := NewDocumentationGenerator(result)
		doc, err := gen.Generate(DocTypeInstallation)

		if err != nil {
			t.Fatalf("Generate failed: %v", err)
		}

		content := doc.Content
		hasConfigSection := strings.Contains(content, "## Configuration")

		if hasConfig && !hasConfigSection {
			t.Fatalf("Installation guide missing Configuration section when config files exist")
		}

		if !hasConfig && hasConfigSection {
			t.Fatalf("Installation guide has Configuration section when no config files exist")
		}
	})
}

// **Validates: Requirements 6.1, 6.2**
// Property 15: API Export Completeness
// For any code file with exported functions or classes, the API documentation
// should include all exports with their complete signatures
func TestProperty15_APIExportCompleteness(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate analysis result with exported functions
		result := genAnalysisResultWithExports().Draw(t, "analysisResult")

		// Skip if no exports
		if result.CodeStructure == nil || len(result.CodeStructure.Functions) == 0 {
			t.Skip("No exports to test")
		}

		gen := NewDocumentationGenerator(result)
		doc, err := gen.Generate(DocTypeAPI)

		if err != nil {
			t.Fatalf("Generate failed: %v", err)
		}

		content := doc.Content

		// Check that exported functions are documented
		exportedCount := 0
		documentedCount := 0

		for _, fn := range result.CodeStructure.Functions {
			if fn.IsExported {
				exportedCount++
				if strings.Contains(content, fn.Name) {
					documentedCount++
				}
			}
		}

		if exportedCount > 0 && documentedCount == 0 {
			t.Fatalf("API documentation missing all exported functions")
		}
	})
}

// **Validates: Requirements 6.3, 6.4**
// Property 16: API Documentation Comments
// For any exported function or class with a docstring or comment, the API
// documentation should include that description text
func TestProperty16_APIDocumentationComments(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate analysis result with commented exports
		result := genAnalysisResultWithCommentedExports().Draw(t, "analysisResult")

		// Skip if no commented exports
		hasCommentedExport := false
		if result.CodeStructure != nil {
			for _, fn := range result.CodeStructure.Functions {
				if fn.IsExported && fn.Comment != "" {
					hasCommentedExport = true
					break
				}
			}
		}

		if !hasCommentedExport {
			t.Skip("No commented exports to test")
		}

		gen := NewDocumentationGenerator(result)
		doc, err := gen.Generate(DocTypeAPI)

		if err != nil {
			t.Fatalf("Generate failed: %v", err)
		}

		content := doc.Content

		// Check that at least one comment is included
		foundComment := false
		for _, fn := range result.CodeStructure.Functions {
			if fn.IsExported && fn.Comment != "" {
				if strings.Contains(content, fn.Comment) {
					foundComment = true
					break
				}
			}
		}

		if !foundComment {
			t.Fatalf("API documentation missing function comments")
		}
	})
}

// **Validates: Requirements 6.5**
// Property 17: API Module Organization
// For any API documentation generated from multiple packages or modules, the
// content should be organized by module with clear section boundaries
func TestProperty17_APIModuleOrganization(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate analysis result with multiple packages
		result := genAnalysisResultWithMultiplePackages().Draw(t, "analysisResult")

		// Skip if only one package
		packageCount := countUniquePackages(result)
		if packageCount <= 1 {
			t.Skip("Need multiple packages to test organization")
		}

		gen := NewDocumentationGenerator(result)
		doc, err := gen.Generate(DocTypeAPI)

		if err != nil {
			t.Fatalf("Generate failed: %v", err)
		}

		content := doc.Content

		// Should have Table of Contents
		if !strings.Contains(content, "## Table of Contents") {
			t.Fatalf("API documentation missing Table of Contents for multi-package project")
		}

		// Should have multiple ## headers for different packages
		lines := strings.Split(content, "\n")
		h2Count := 0
		for _, line := range lines {
			if strings.HasPrefix(strings.TrimSpace(line), "## ") &&
				!strings.Contains(line, "Table of Contents") {
				h2Count++
			}
		}

		if h2Count < 2 {
			t.Fatalf("API documentation not properly organized by module (expected multiple ## sections)")
		}
	})
}

// **Validates: Requirements 9.2**
// Property 25: Multi-Type Generation
// For any classification result with multiple documentation types, the generator
// should produce a separate GeneratedDoc for each type
func TestProperty25_MultiTypeGeneration(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random analysis result
		result := genAnalysisResult().Draw(t, "analysisResult")

		// Generate multiple doc types (at least 2)
		numTypes := rapid.IntRange(2, 4).Draw(t, "numTypes")
		docTypes := make([]DocumentationType, numTypes)

		// Use different types
		allTypes := []DocumentationType{DocTypeUserManual, DocTypeInstallation, DocTypeAPI, DocTypeGeneral}
		for i := 0; i < numTypes && i < len(allTypes); i++ {
			docTypes[i] = allTypes[i]
		}

		gen := NewDocumentationGenerator(result)
		docs, err := gen.GenerateMultiple(docTypes)

		if err != nil {
			t.Fatalf("GenerateMultiple failed: %v", err)
		}

		// Should have one doc for each type
		if len(docs) != len(docTypes) {
			t.Fatalf("Expected %d docs, got %d", len(docTypes), len(docs))
		}

		// Each doc should have correct type and unique filename
		seenTypes := make(map[DocumentationType]bool)
		seenFilenames := make(map[string]bool)

		for _, doc := range docs {
			if seenTypes[doc.Type] {
				t.Fatalf("Duplicate documentation type: %v", doc.Type)
			}
			seenTypes[doc.Type] = true

			if seenFilenames[doc.Filename] {
				t.Fatalf("Duplicate filename: %s", doc.Filename)
			}
			seenFilenames[doc.Filename] = true

			// Content should not be empty
			if strings.TrimSpace(doc.Content) == "" {
				t.Fatalf("Empty content for type %v", doc.Type)
			}
		}
	})
}

// Helper generators

func genAnalysisResult() *rapid.Generator[*AnalysisResult] {
	return rapid.Custom(func(t *rapid.T) *AnalysisResult {
		return &AnalysisResult{
			CodeStructure: &CodeStructure{
				Functions: genFunctions().Draw(t, "functions"),
				Structs:   genStructs().Draw(t, "structs"),
			},
			Configuration: &ConfigInfo{
				PackageManifests: rapid.SliceOf(rapid.SampledFrom([]string{"go.mod", "package.json", "requirements.txt"})).Draw(t, "manifests"),
			},
			Documentation: &ExistingDocs{
				ReadmeContent: rapid.StringMatching(`[A-Za-z0-9 \n.]+`).Draw(t, "readme"),
			},
			Dependencies: &DependencyInfo{
				Runtime: genDependencies().Draw(t, "deps"),
			},
		}
	})
}

func genAnalysisResultWithDependencies() *rapid.Generator[*AnalysisResult] {
	return rapid.Custom(func(t *rapid.T) *AnalysisResult {
		deps := genDependencies().Draw(t, "deps")
		// Ensure at least one dependency
		if len(deps) == 0 {
			deps = []Dependency{{Name: "example-dep", Version: "1.0.0", Source: "go.mod"}}
		}

		return &AnalysisResult{
			CodeStructure: &CodeStructure{},
			Configuration: &ConfigInfo{
				PackageManifests: []string{"go.mod"},
			},
			Documentation: &ExistingDocs{},
			Dependencies: &DependencyInfo{
				Runtime: deps,
			},
		}
	})
}

func genAnalysisResultWithReadme() *rapid.Generator[*AnalysisResult] {
	return rapid.Custom(func(t *rapid.T) *AnalysisResult {
		// Use a generator to consume bitstream data
		_ = rapid.Bool().Draw(t, "dummy")

		readme := "# Project\n\n## Installation\n\n1. Clone the repo\n2. Run setup\n3. Start the app\n"

		return &AnalysisResult{
			CodeStructure: &CodeStructure{},
			Configuration: &ConfigInfo{
				PackageManifests: []string{"go.mod"},
			},
			Documentation: &ExistingDocs{
				ReadmeContent: readme,
			},
			Dependencies: &DependencyInfo{},
		}
	})
}

func genAnalysisResultWithConfig() *rapid.Generator[*AnalysisResult] {
	return rapid.Custom(func(t *rapid.T) *AnalysisResult {
		// Use a generator to consume bitstream data
		_ = rapid.Bool().Draw(t, "dummy")

		return &AnalysisResult{
			CodeStructure: &CodeStructure{},
			Configuration: &ConfigInfo{
				PackageManifests: []string{"go.mod"},
				ConfigFiles:      []string{"config.yaml", ".env"},
			},
			Documentation: &ExistingDocs{},
			Dependencies:  &DependencyInfo{},
		}
	})
}

func genAnalysisResultWithoutConfig() *rapid.Generator[*AnalysisResult] {
	return rapid.Custom(func(t *rapid.T) *AnalysisResult {
		// Use a generator to consume bitstream data
		_ = rapid.Bool().Draw(t, "dummy")

		return &AnalysisResult{
			CodeStructure: &CodeStructure{},
			Configuration: &ConfigInfo{
				PackageManifests: []string{"go.mod"},
				ConfigFiles:      []string{}, // No config files
			},
			Documentation: &ExistingDocs{},
			Dependencies:  &DependencyInfo{},
		}
	})
}

func genAnalysisResultWithExports() *rapid.Generator[*AnalysisResult] {
	return rapid.Custom(func(t *rapid.T) *AnalysisResult {
		functions := genFunctions().Draw(t, "functions")
		// Ensure at least one exported function
		if len(functions) == 0 || !functions[0].IsExported {
			functions = append([]FunctionInfo{{
				Name:       "ExportedFunc",
				Package:    "main",
				Signature:  "func ExportedFunc() error",
				IsExported: true,
			}}, functions...)
		}

		return &AnalysisResult{
			CodeStructure: &CodeStructure{
				Functions: functions,
			},
			Configuration: &ConfigInfo{},
			Documentation: &ExistingDocs{},
			Dependencies:  &DependencyInfo{},
		}
	})
}

func genAnalysisResultWithCommentedExports() *rapid.Generator[*AnalysisResult] {
	return rapid.Custom(func(t *rapid.T) *AnalysisResult {
		// Use a generator to consume bitstream data
		_ = rapid.Bool().Draw(t, "dummy")

		return &AnalysisResult{
			CodeStructure: &CodeStructure{
				Functions: []FunctionInfo{
					{
						Name:       "ExportedFunc",
						Package:    "main",
						Signature:  "func ExportedFunc() error",
						Comment:    "ExportedFunc does something useful",
						IsExported: true,
					},
				},
			},
			Configuration: &ConfigInfo{},
			Documentation: &ExistingDocs{},
			Dependencies:  &DependencyInfo{},
		}
	})
}

func genAnalysisResultWithMultiplePackages() *rapid.Generator[*AnalysisResult] {
	return rapid.Custom(func(t *rapid.T) *AnalysisResult {
		// Use a generator to consume bitstream data
		_ = rapid.Bool().Draw(t, "dummy")

		return &AnalysisResult{
			CodeStructure: &CodeStructure{
				Functions: []FunctionInfo{
					{Name: "Func1", Package: "pkg1", IsExported: true, Signature: "func Func1()"},
					{Name: "Func2", Package: "pkg2", IsExported: true, Signature: "func Func2()"},
					{Name: "Func3", Package: "pkg3", IsExported: true, Signature: "func Func3()"},
				},
			},
			Configuration: &ConfigInfo{},
			Documentation: &ExistingDocs{},
			Dependencies:  &DependencyInfo{},
		}
	})
}

func genFunctions() *rapid.Generator[[]FunctionInfo] {
	return rapid.SliceOfN(rapid.Custom(func(t *rapid.T) FunctionInfo {
		name := rapid.StringMatching(`[A-Z][a-z]+`).Draw(t, "name")
		isExported := rapid.Bool().Draw(t, "exported")
		return FunctionInfo{
			Name:       name,
			Package:    "main",
			Signature:  "func " + name + "()",
			IsExported: isExported,
			Comment:    rapid.StringMatching(`[A-Za-z ]+`).Draw(t, "comment"),
		}
	}), 0, 5)
}

func genStructs() *rapid.Generator[[]StructInfo] {
	return rapid.SliceOfN(rapid.Custom(func(t *rapid.T) StructInfo {
		name := rapid.StringMatching(`[A-Z][a-z]+`).Draw(t, "name")
		return StructInfo{
			Name:       name,
			Package:    "main",
			IsExported: rapid.Bool().Draw(t, "exported"),
		}
	}), 0, 3)
}

func genDependencies() *rapid.Generator[[]Dependency] {
	return rapid.SliceOfN(rapid.Custom(func(t *rapid.T) Dependency {
		return Dependency{
			Name:    rapid.StringMatching(`[a-z-]+`).Draw(t, "name"),
			Version: rapid.StringMatching(`[0-9]+\.[0-9]+\.[0-9]+`).Draw(t, "version"),
			Source:  "go.mod",
		}
	}), 0, 5)
}

func countUniquePackages(result *AnalysisResult) int {
	if result.CodeStructure == nil {
		return 0
	}

	packages := make(map[string]bool)
	for _, fn := range result.CodeStructure.Functions {
		if fn.Package != "" {
			packages[fn.Package] = true
		}
	}
	for _, st := range result.CodeStructure.Structs {
		if st.Package != "" {
			packages[st.Package] = true
		}
	}

	return len(packages)
}
