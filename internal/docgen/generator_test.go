package docgen

import (
	"strings"
	"testing"
)

// Test User Manual generation for CLI tool
func TestUserManualForCLITool(t *testing.T) {
	result := &AnalysisResult{
		CodeStructure: &CodeStructure{
			Functions: []FunctionInfo{
				{
					Name:       "Run",
					Package:    "main",
					Signature:  "func Run(args []string) error",
					Comment:    "Run executes the main application logic",
					IsExported: true,
				},
				{
					Name:       "Parse",
					Package:    "main",
					Signature:  "func Parse(input string) (*Config, error)",
					Comment:    "Parse parses the configuration from input",
					IsExported: true,
				},
			},
		},
		Configuration: &ConfigInfo{
			PackageManifests: []string{"go.mod"},
		},
		Documentation: &ExistingDocs{
			ReadmeContent: "# CLI Tool\n\nA powerful command-line tool for processing data.\n\n## Features\n\n- Fast processing\n- Easy to use",
		},
		Dependencies: &DependencyInfo{
			Runtime: []Dependency{
				{Name: "github.com/spf13/cobra", Version: "1.5.0", Source: "go.mod"},
			},
		},
	}

	gen := NewDocumentationGenerator(result)
	doc, err := gen.Generate(DocTypeUserManual)

	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if doc.Type != DocTypeUserManual {
		t.Errorf("Expected type UserManual, got %v", doc.Type)
	}

	if doc.Filename != "USER_MANUAL.md" {
		t.Errorf("Expected filename USER_MANUAL.md, got %s", doc.Filename)
	}

	content := doc.Content

	// Check for required sections
	if !strings.Contains(content, "# User Manual") {
		t.Error("Missing title")
	}

	if !strings.Contains(content, "## Overview") {
		t.Error("Missing Overview section")
	}

	if !strings.Contains(content, "## Usage") {
		t.Error("Missing Usage section")
	}

	if !strings.Contains(content, "## Features") {
		t.Error("Missing Features section")
	}

	// Check that exported functions are documented
	if !strings.Contains(content, "Run") {
		t.Error("Missing Run function in features")
	}
}

// Test User Manual with no comments
func TestUserManualWithNoComments(t *testing.T) {
	result := &AnalysisResult{
		CodeStructure: &CodeStructure{
			Functions: []FunctionInfo{
				{
					Name:       "Process",
					Package:    "main",
					Signature:  "func Process()",
					Comment:    "", // No comment
					IsExported: true,
				},
			},
		},
		Configuration: &ConfigInfo{},
		Documentation: &ExistingDocs{
			ReadmeContent: "", // No README
		},
		Dependencies: &DependencyInfo{},
	}

	gen := NewDocumentationGenerator(result)
	doc, err := gen.Generate(DocTypeUserManual)

	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	content := doc.Content

	// Should still have basic structure
	if !strings.Contains(content, "# User Manual") {
		t.Error("Missing title")
	}

	if !strings.Contains(content, "## Overview") {
		t.Error("Missing Overview section")
	}

	// Should have minimal/default content
	if !strings.Contains(content, "## Usage") {
		t.Error("Missing Usage section")
	}
}

// Test Installation Guide for Go project
func TestInstallationGuideForGoProject(t *testing.T) {
	result := &AnalysisResult{
		CodeStructure: &CodeStructure{},
		Configuration: &ConfigInfo{
			PackageManifests: []string{"go.mod"},
			ConfigFiles:      []string{"config.yaml"},
		},
		Documentation: &ExistingDocs{
			ReadmeContent: "# Project\n\n## Installation\n\n1. Clone the repository\n2. Run go build\n",
		},
		Dependencies: &DependencyInfo{
			Runtime: []Dependency{
				{Name: "github.com/gin-gonic/gin", Version: "1.8.0", Source: "go.mod"},
				{Name: "gorm.io/gorm", Version: "1.24.0", Source: "go.mod"},
			},
		},
	}

	gen := NewDocumentationGenerator(result)
	doc, err := gen.Generate(DocTypeInstallation)

	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if doc.Type != DocTypeInstallation {
		t.Errorf("Expected type Installation, got %v", doc.Type)
	}

	if doc.Filename != "INSTALLATION.md" {
		t.Errorf("Expected filename INSTALLATION.md, got %s", doc.Filename)
	}

	content := doc.Content

	// Check for required sections
	if !strings.Contains(content, "# Installation Guide") {
		t.Error("Missing title")
	}

	if !strings.Contains(content, "## System Requirements") {
		t.Error("Missing System Requirements section")
	}

	if !strings.Contains(content, "Go 1.16 or higher") {
		t.Error("Missing Go requirement")
	}

	if !strings.Contains(content, "## Dependencies") {
		t.Error("Missing Dependencies section")
	}

	// Check dependencies are listed
	if !strings.Contains(content, "github.com/gin-gonic/gin") {
		t.Error("Missing gin dependency")
	}

	if !strings.Contains(content, "## Installation Steps") {
		t.Error("Missing Installation Steps section")
	}

	// Check for configuration section since config files exist
	if !strings.Contains(content, "## Configuration") {
		t.Error("Missing Configuration section when config files exist")
	}

	if !strings.Contains(content, "config.yaml") {
		t.Error("Missing config file reference")
	}
}

// Test Installation Guide without config files
func TestInstallationGuideWithoutConfigFiles(t *testing.T) {
	result := &AnalysisResult{
		CodeStructure: &CodeStructure{},
		Configuration: &ConfigInfo{
			PackageManifests: []string{"package.json"},
			ConfigFiles:      []string{}, // No config files
		},
		Documentation: &ExistingDocs{},
		Dependencies: &DependencyInfo{
			Runtime: []Dependency{
				{Name: "express", Version: "4.18.0", Source: "package.json"},
			},
		},
	}

	gen := NewDocumentationGenerator(result)
	doc, err := gen.Generate(DocTypeInstallation)

	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	content := doc.Content

	// Should NOT have configuration section
	if strings.Contains(content, "## Configuration") {
		t.Error("Should not have Configuration section when no config files exist")
	}

	// Should have Node.js requirements
	if !strings.Contains(content, "Node.js") {
		t.Error("Missing Node.js requirement")
	}
}

// Test API Documentation for library
func TestAPIDocumentationForLibrary(t *testing.T) {
	result := &AnalysisResult{
		CodeStructure: &CodeStructure{
			Functions: []FunctionInfo{
				{
					Name:      "NewClient",
					Package:   "client",
					Signature: "func NewClient(config *Config) *Client",
					Parameters: []Parameter{
						{Name: "config", Type: "*Config"},
					},
					Returns: []ReturnValue{
						{Type: "*Client"},
					},
					Comment:    "NewClient creates a new client instance",
					IsExported: true,
				},
				{
					Name:      "Connect",
					Package:   "client",
					Signature: "func (c *Client) Connect() error",
					Returns: []ReturnValue{
						{Type: "error"},
					},
					Comment:    "Connect establishes a connection to the server",
					IsExported: true,
				},
			},
			Structs: []StructInfo{
				{
					Name:    "Client",
					Package: "client",
					Fields: []FieldInfo{
						{Name: "URL", Type: "string", Comment: "Server URL"},
						{Name: "Timeout", Type: "time.Duration", Comment: "Connection timeout"},
					},
					Comment:    "Client represents a connection client",
					IsExported: true,
				},
			},
			Interfaces: []InterfaceInfo{
				{
					Name:    "Handler",
					Package: "client",
					Methods: []MethodSignature{
						{
							Name: "Handle",
							Parameters: []Parameter{
								{Name: "req", Type: "*Request"},
							},
							Returns: []ReturnValue{
								{Type: "error"},
							},
						},
					},
					Comment:    "Handler processes requests",
					IsExported: true,
				},
			},
		},
		Configuration: &ConfigInfo{},
		Documentation: &ExistingDocs{},
		Dependencies:  &DependencyInfo{},
	}

	gen := NewDocumentationGenerator(result)
	doc, err := gen.Generate(DocTypeAPI)

	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if doc.Type != DocTypeAPI {
		t.Errorf("Expected type API, got %v", doc.Type)
	}

	if doc.Filename != "API_REFERENCE.md" {
		t.Errorf("Expected filename API_REFERENCE.md, got %s", doc.Filename)
	}

	content := doc.Content

	// Check for required sections
	if !strings.Contains(content, "# API Reference") {
		t.Error("Missing title")
	}

	if !strings.Contains(content, "## Table of Contents") {
		t.Error("Missing Table of Contents")
	}

	// Check package section
	if !strings.Contains(content, "## client") {
		t.Error("Missing client package section")
	}

	// Check functions are documented
	if !strings.Contains(content, "### Functions") {
		t.Error("Missing Functions section")
	}

	if !strings.Contains(content, "#### NewClient") {
		t.Error("Missing NewClient function")
	}

	if !strings.Contains(content, "NewClient creates a new client instance") {
		t.Error("Missing function comment")
	}

	// Check structs are documented
	if !strings.Contains(content, "### Types") {
		t.Error("Missing Types section")
	}

	if !strings.Contains(content, "#### Client") {
		t.Error("Missing Client struct")
	}

	if !strings.Contains(content, "**Fields:**") {
		t.Error("Missing Fields section")
	}

	if !strings.Contains(content, "URL") {
		t.Error("Missing URL field")
	}

	// Check interfaces are documented
	if !strings.Contains(content, "### Interfaces") {
		t.Error("Missing Interfaces section")
	}

	if !strings.Contains(content, "#### Handler") {
		t.Error("Missing Handler interface")
	}
}

// Test API Documentation with no exports
func TestAPIDocumentationWithNoExports(t *testing.T) {
	result := &AnalysisResult{
		CodeStructure: &CodeStructure{
			Functions: []FunctionInfo{
				{
					Name:       "helper",
					Package:    "internal",
					Signature:  "func helper()",
					IsExported: false, // Not exported
				},
			},
		},
		Configuration: &ConfigInfo{},
		Documentation: &ExistingDocs{},
		Dependencies:  &DependencyInfo{},
	}

	gen := NewDocumentationGenerator(result)
	doc, err := gen.Generate(DocTypeAPI)

	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	content := doc.Content

	// Should have basic structure
	if !strings.Contains(content, "# API Reference") {
		t.Error("Missing title")
	}

	// Should not document unexported functions
	if strings.Contains(content, "helper") {
		t.Error("Should not document unexported function")
	}
}

// Test multi-type generation
func TestMultiTypeGeneration(t *testing.T) {
	result := &AnalysisResult{
		CodeStructure: &CodeStructure{
			Functions: []FunctionInfo{
				{Name: "Run", Package: "main", IsExported: true, Signature: "func Run()"},
			},
		},
		Configuration: &ConfigInfo{
			PackageManifests: []string{"go.mod"},
		},
		Documentation: &ExistingDocs{
			ReadmeContent: "# Project\n\nA sample project.",
		},
		Dependencies: &DependencyInfo{
			Runtime: []Dependency{
				{Name: "example", Version: "1.0.0", Source: "go.mod"},
			},
		},
	}

	gen := NewDocumentationGenerator(result)
	docTypes := []DocumentationType{DocTypeUserManual, DocTypeInstallation, DocTypeAPI}
	docs, err := gen.GenerateMultiple(docTypes)

	if err != nil {
		t.Fatalf("GenerateMultiple failed: %v", err)
	}

	if len(docs) != 3 {
		t.Fatalf("Expected 3 docs, got %d", len(docs))
	}

	// Check each doc type
	typesSeen := make(map[DocumentationType]bool)
	for _, doc := range docs {
		if typesSeen[doc.Type] {
			t.Errorf("Duplicate doc type: %v", doc.Type)
		}
		typesSeen[doc.Type] = true

		if strings.TrimSpace(doc.Content) == "" {
			t.Errorf("Empty content for type %v", doc.Type)
		}

		if doc.Filename == "" {
			t.Errorf("Empty filename for type %v", doc.Type)
		}
	}

	// Verify all requested types were generated
	if !typesSeen[DocTypeUserManual] {
		t.Error("Missing UserManual doc")
	}
	if !typesSeen[DocTypeInstallation] {
		t.Error("Missing Installation doc")
	}
	if !typesSeen[DocTypeAPI] {
		t.Error("Missing API doc")
	}
}

// Test generation with one type failing (graceful degradation)
func TestMultiTypeGenerationWithFailure(t *testing.T) {
	result := &AnalysisResult{
		CodeStructure: &CodeStructure{},
		Configuration: &ConfigInfo{},
		Documentation: &ExistingDocs{},
		Dependencies:  &DependencyInfo{},
	}

	gen := NewDocumentationGenerator(result)

	// Include valid types
	docTypes := []DocumentationType{DocTypeUserManual, DocTypeInstallation}
	docs, err := gen.GenerateMultiple(docTypes)

	// Should succeed even if some content is minimal
	if err != nil {
		t.Fatalf("GenerateMultiple should not fail with valid types: %v", err)
	}

	if len(docs) != 2 {
		t.Fatalf("Expected 2 docs, got %d", len(docs))
	}
}

// Test markdown validity
func TestMarkdownValidity(t *testing.T) {
	result := &AnalysisResult{
		CodeStructure: &CodeStructure{
			Functions: []FunctionInfo{
				{
					Name:       "Example",
					Package:    "main",
					Signature:  "func Example() error",
					Comment:    "Example function",
					IsExported: true,
				},
			},
		},
		Configuration: &ConfigInfo{},
		Documentation: &ExistingDocs{},
		Dependencies:  &DependencyInfo{},
	}

	gen := NewDocumentationGenerator(result)

	// Test all doc types
	docTypes := []DocumentationType{DocTypeUserManual, DocTypeInstallation, DocTypeAPI}

	for _, docType := range docTypes {
		doc, err := gen.Generate(docType)
		if err != nil {
			t.Fatalf("Generate failed for %v: %v", docType, err)
		}

		content := doc.Content

		// Check code blocks are balanced
		codeBlockCount := strings.Count(content, "```")
		if codeBlockCount%2 != 0 {
			t.Errorf("Unclosed code block in %v", docType)
		}

		// Check headers have space after #
		lines := strings.Split(content, "\n")
		for i, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "#") && !strings.HasPrefix(trimmed, "####") {
				// Count hashes
				hashCount := 0
				for _, ch := range trimmed {
					if ch == '#' {
						hashCount++
					} else {
						break
					}
				}

				if len(trimmed) > hashCount && trimmed[hashCount] != ' ' {
					t.Errorf("Invalid header at line %d in %v: %s", i+1, docType, line)
				}
			}
		}

		// Content should not be empty
		if strings.TrimSpace(content) == "" {
			t.Errorf("Empty content for %v", docType)
		}
	}
}
