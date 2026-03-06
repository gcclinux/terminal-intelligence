package docgen

import (
	"fmt"
	"strings"
)

// DocumentationGenerator generates documentation content from analysis results
type DocumentationGenerator struct {
	analysisResult *AnalysisResult
}

// NewDocumentationGenerator creates a new documentation generator
func NewDocumentationGenerator(result *AnalysisResult) *DocumentationGenerator {
	return &DocumentationGenerator{
		analysisResult: result,
	}
}

// Generate generates documentation for a specific type
func (g *DocumentationGenerator) Generate(docType DocumentationType) (*GeneratedDoc, error) {
	var content string
	var err error

	switch docType {
	case DocTypeUserManual:
		content, err = g.generateUserManual()
	case DocTypeInstallation:
		content, err = g.generateInstallationGuide()
	case DocTypeAPI:
		content, err = g.generateAPIDocumentation()
	case DocTypeTutorial:
		content = g.generateTutorial()
	case DocTypeGeneral:
		content = g.generateGeneral()
	default:
		return nil, fmt.Errorf("unsupported documentation type: %v", docType)
	}

	if err != nil {
		return nil, err
	}

	return &GeneratedDoc{
		Type:     docType,
		Content:  content,
		Filename: docType.Filename(),
	}, nil
}

// GenerateMultiple generates documentation for multiple types
func (g *DocumentationGenerator) GenerateMultiple(docTypes []DocumentationType) ([]*GeneratedDoc, error) {
	var docs []*GeneratedDoc
	var errors []error

	for _, docType := range docTypes {
		doc, err := g.Generate(docType)
		if err != nil {
			errors = append(errors, fmt.Errorf("failed to generate %s: %w", docType.String(), err))
			continue
		}
		docs = append(docs, doc)
	}

	// Return partial results even if some failed
	if len(errors) > 0 && len(docs) == 0 {
		return nil, fmt.Errorf("all documentation generation failed: %v", errors)
	}

	return docs, nil
}

// generateUserManual generates a user manual
func (g *DocumentationGenerator) generateUserManual() (string, error) {
	var sb strings.Builder

	// Title
	sb.WriteString("# User Manual\n\n")

	// Overview section
	sb.WriteString("## Overview\n\n")
	if g.analysisResult.Documentation != nil && g.analysisResult.Documentation.ReadmeContent != "" {
		// Extract first paragraph or first few lines from README
		overview := extractOverview(g.analysisResult.Documentation.ReadmeContent)
		sb.WriteString(overview)
		sb.WriteString("\n\n")
	} else {
		sb.WriteString("This application provides various features and capabilities.\n\n")
	}

	// Installation section
	sb.WriteString("## Installation\n\n")
	sb.WriteString("Please refer to the INSTALLATION.md file for detailed installation instructions.\n\n")

	// Usage section
	sb.WriteString("## Usage\n\n")

	// Extract command-line flags and options from main function
	commands := g.extractCommands()
	if len(commands) > 0 {
		sb.WriteString("### Commands and Options\n\n")
		for _, cmd := range commands {
			sb.WriteString(fmt.Sprintf("- **%s**: %s\n", cmd.Name, cmd.Description))
		}
		sb.WriteString("\n")
	}

	// Extract keyboard shortcuts if any
	shortcuts := g.extractKeyboardShortcuts()
	if len(shortcuts) > 0 {
		sb.WriteString("### Keyboard Shortcuts\n\n")
		for _, shortcut := range shortcuts {
			sb.WriteString(fmt.Sprintf("- **%s**: %s\n", shortcut.Key, shortcut.Description))
		}
		sb.WriteString("\n")
	}

	// Features section
	if g.analysisResult.CodeStructure != nil && len(g.analysisResult.CodeStructure.Functions) > 0 {
		sb.WriteString("## Features\n\n")
		sb.WriteString("This application provides the following capabilities:\n\n")

		// List exported functions as features
		for _, fn := range g.analysisResult.CodeStructure.Functions {
			if fn.IsExported && fn.Comment != "" {
				sb.WriteString(fmt.Sprintf("- **%s**: %s\n", fn.Name, fn.Comment))
			}
		}
		sb.WriteString("\n")
	}

	// Troubleshooting section
	sb.WriteString("## Troubleshooting\n\n")
	sb.WriteString("For issues and support, please refer to the project documentation or contact the maintainers.\n\n")

	return sb.String(), nil
}

// generateInstallationGuide generates an installation guide
func (g *DocumentationGenerator) generateInstallationGuide() (string, error) {
	var sb strings.Builder

	// Title
	sb.WriteString("# Installation Guide\n\n")

	// System Requirements
	sb.WriteString("## System Requirements\n\n")
	if g.analysisResult.Dependencies != nil {
		sb.WriteString(g.formatSystemRequirements())
	} else {
		sb.WriteString("No specific system requirements identified.\n\n")
	}

	// Dependencies
	if g.analysisResult.Dependencies != nil && len(g.analysisResult.Dependencies.Runtime) > 0 {
		sb.WriteString("## Dependencies\n\n")
		sb.WriteString("This project requires the following dependencies:\n\n")
		for _, dep := range g.analysisResult.Dependencies.Runtime {
			if dep.Version != "" {
				sb.WriteString(fmt.Sprintf("- %s (version %s)\n", dep.Name, dep.Version))
			} else {
				sb.WriteString(fmt.Sprintf("- %s\n", dep.Name))
			}
		}
		sb.WriteString("\n")
	}

	// Installation Steps
	sb.WriteString("## Installation Steps\n\n")
	steps := g.extractInstallationSteps()
	if len(steps) > 0 {
		for i, step := range steps {
			sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, step))
		}
		sb.WriteString("\n")
	} else {
		sb.WriteString(g.generateDefaultInstallationSteps())
	}

	// Configuration
	if g.analysisResult.Configuration != nil && len(g.analysisResult.Configuration.ConfigFiles) > 0 {
		sb.WriteString("## Configuration\n\n")
		sb.WriteString("The following configuration files are available:\n\n")
		for _, cfg := range g.analysisResult.Configuration.ConfigFiles {
			sb.WriteString(fmt.Sprintf("- `%s`\n", cfg))
		}
		sb.WriteString("\nPlease configure these files according to your environment before running the application.\n\n")
	}

	return sb.String(), nil
}

// generateAPIDocumentation generates API reference documentation
func (g *DocumentationGenerator) generateAPIDocumentation() (string, error) {
	var sb strings.Builder

	// Title
	sb.WriteString("# API Reference\n\n")

	if g.analysisResult.CodeStructure == nil {
		sb.WriteString("No API documentation available.\n\n")
		return sb.String(), nil
	}

	// Table of Contents
	sb.WriteString("## Table of Contents\n\n")

	// Group by package
	packageMap := g.groupByPackage()
	for pkgName := range packageMap {
		sb.WriteString(fmt.Sprintf("- [%s](#%s)\n", pkgName, strings.ToLower(strings.ReplaceAll(pkgName, "/", ""))))
	}
	sb.WriteString("\n")

	// Document each package
	for pkgName, items := range packageMap {
		sb.WriteString(fmt.Sprintf("## %s\n\n", pkgName))

		// Document functions
		if len(items.Functions) > 0 {
			sb.WriteString("### Functions\n\n")
			for _, fn := range items.Functions {
				if !fn.IsExported {
					continue
				}
				sb.WriteString(fmt.Sprintf("#### %s\n\n", fn.Name))
				if fn.Comment != "" {
					sb.WriteString(fmt.Sprintf("%s\n\n", fn.Comment))
				}
				sb.WriteString(fmt.Sprintf("```go\n%s\n```\n\n", fn.Signature))

				if len(fn.Parameters) > 0 {
					sb.WriteString("**Parameters:**\n\n")
					for _, param := range fn.Parameters {
						sb.WriteString(fmt.Sprintf("- `%s` (%s)\n", param.Name, param.Type))
					}
					sb.WriteString("\n")
				}

				if len(fn.Returns) > 0 {
					sb.WriteString("**Returns:**\n\n")
					for _, ret := range fn.Returns {
						if ret.Name != "" {
							sb.WriteString(fmt.Sprintf("- `%s` (%s)\n", ret.Name, ret.Type))
						} else {
							sb.WriteString(fmt.Sprintf("- %s\n", ret.Type))
						}
					}
					sb.WriteString("\n")
				}
			}
		}

		// Document structs
		if len(items.Structs) > 0 {
			sb.WriteString("### Types\n\n")
			for _, st := range items.Structs {
				if !st.IsExported {
					continue
				}
				sb.WriteString(fmt.Sprintf("#### %s\n\n", st.Name))
				if st.Comment != "" {
					sb.WriteString(fmt.Sprintf("%s\n\n", st.Comment))
				}

				if len(st.Fields) > 0 {
					sb.WriteString("**Fields:**\n\n")
					for _, field := range st.Fields {
						sb.WriteString(fmt.Sprintf("- `%s` (%s)", field.Name, field.Type))
						if field.Comment != "" {
							sb.WriteString(fmt.Sprintf(" - %s", field.Comment))
						}
						sb.WriteString("\n")
					}
					sb.WriteString("\n")
				}

				if len(st.Methods) > 0 {
					sb.WriteString("**Methods:**\n\n")
					for _, method := range st.Methods {
						sb.WriteString(fmt.Sprintf("- `%s`", method.Signature))
						if method.Comment != "" {
							sb.WriteString(fmt.Sprintf(" - %s", method.Comment))
						}
						sb.WriteString("\n")
					}
					sb.WriteString("\n")
				}
			}
		}

		// Document interfaces
		if len(items.Interfaces) > 0 {
			sb.WriteString("### Interfaces\n\n")
			for _, iface := range items.Interfaces {
				if !iface.IsExported {
					continue
				}
				sb.WriteString(fmt.Sprintf("#### %s\n\n", iface.Name))
				if iface.Comment != "" {
					sb.WriteString(fmt.Sprintf("%s\n\n", iface.Comment))
				}

				if len(iface.Methods) > 0 {
					sb.WriteString("**Methods:**\n\n")
					for _, method := range iface.Methods {
						sb.WriteString(fmt.Sprintf("- `%s(", method.Name))
						for i, param := range method.Parameters {
							if i > 0 {
								sb.WriteString(", ")
							}
							sb.WriteString(fmt.Sprintf("%s %s", param.Name, param.Type))
						}
						sb.WriteString(")")
						if len(method.Returns) > 0 {
							sb.WriteString(" (")
							for i, ret := range method.Returns {
								if i > 0 {
									sb.WriteString(", ")
								}
								sb.WriteString(ret.Type)
							}
							sb.WriteString(")")
						}
						sb.WriteString("`\n")
					}
					sb.WriteString("\n")
				}
			}
		}
	}

	return sb.String(), nil
}

// generateTutorial generates a tutorial
func (g *DocumentationGenerator) generateTutorial() string {
	var sb strings.Builder
	sb.WriteString("# Tutorial\n\n")
	sb.WriteString("This tutorial will guide you through using this application.\n\n")
	return sb.String()
}

// generateGeneral generates general documentation
func (g *DocumentationGenerator) generateGeneral() string {
	var sb strings.Builder
	sb.WriteString("# Documentation\n\n")

	if g.analysisResult.Documentation != nil && g.analysisResult.Documentation.ReadmeContent != "" {
		sb.WriteString(g.analysisResult.Documentation.ReadmeContent)
		sb.WriteString("\n\n")
	}

	return sb.String()
}

// Helper types and functions

type command struct {
	Name        string
	Description string
}

type keyboardShortcut struct {
	Key         string
	Description string
}

type packageItems struct {
	Functions  []FunctionInfo
	Structs    []StructInfo
	Interfaces []InterfaceInfo
}

// extractOverview extracts the overview from README content
func extractOverview(readme string) string {
	lines := strings.Split(readme, "\n")
	var overview strings.Builder
	inOverview := false
	lineCount := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip title lines
		if strings.HasPrefix(trimmed, "#") {
			inOverview = true
			continue
		}

		if inOverview && trimmed != "" {
			overview.WriteString(line)
			overview.WriteString("\n")
			lineCount++

			// Take first paragraph (up to 10 lines)
			if lineCount >= 10 {
				break
			}
		}

		// Stop at first empty line after content
		if inOverview && trimmed == "" && lineCount > 0 {
			break
		}
	}

	result := overview.String()
	if result == "" {
		return "No overview available."
	}
	return result
}

// extractCommands extracts command-line commands from the codebase
func (g *DocumentationGenerator) extractCommands() []command {
	var commands []command

	// Look for flag definitions in code comments
	if g.analysisResult.Documentation != nil {
		for _, comment := range g.analysisResult.Documentation.Comments {
			content := strings.ToLower(comment.Content)
			if strings.Contains(content, "flag") || strings.Contains(content, "command") || strings.Contains(content, "option") {
				// Extract command info from comment
				lines := strings.Split(comment.Content, "\n")
				for _, line := range lines {
					if strings.Contains(line, "-") && !strings.HasPrefix(strings.TrimSpace(line), "//") {
						commands = append(commands, command{
							Name:        strings.TrimSpace(line),
							Description: "Command option",
						})
					}
				}
			}
		}
	}

	return commands
}

// extractKeyboardShortcuts extracts keyboard shortcuts from the codebase
func (g *DocumentationGenerator) extractKeyboardShortcuts() []keyboardShortcut {
	var shortcuts []keyboardShortcut

	// Look for keyboard shortcut definitions in comments
	if g.analysisResult.Documentation != nil {
		for _, comment := range g.analysisResult.Documentation.Comments {
			content := strings.ToLower(comment.Content)
			if strings.Contains(content, "keyboard") || strings.Contains(content, "shortcut") || strings.Contains(content, "key") {
				// This is a simplified extraction - could be enhanced
				shortcuts = append(shortcuts, keyboardShortcut{
					Key:         "Ctrl+Key",
					Description: "Keyboard shortcut",
				})
			}
		}
	}

	return shortcuts
}

// formatSystemRequirements formats system requirements from dependencies
func (g *DocumentationGenerator) formatSystemRequirements() string {
	var sb strings.Builder

	// Detect language/runtime from dependencies
	hasGo := false
	hasPython := false
	hasNode := false

	if g.analysisResult.Configuration != nil {
		for _, manifest := range g.analysisResult.Configuration.PackageManifests {
			if strings.Contains(manifest, "go.mod") {
				hasGo = true
			} else if strings.Contains(manifest, "requirements.txt") || strings.Contains(manifest, "setup.py") {
				hasPython = true
			} else if strings.Contains(manifest, "package.json") {
				hasNode = true
			}
		}
	}

	if hasGo {
		sb.WriteString("- Go 1.16 or higher\n")
	}
	if hasPython {
		sb.WriteString("- Python 3.7 or higher\n")
	}
	if hasNode {
		sb.WriteString("- Node.js 14 or higher\n")
		sb.WriteString("- npm or yarn package manager\n")
	}

	if sb.Len() == 0 {
		sb.WriteString("No specific system requirements identified.\n")
	}

	sb.WriteString("\n")
	return sb.String()
}

// extractInstallationSteps extracts installation steps from README
func (g *DocumentationGenerator) extractInstallationSteps() []string {
	var steps []string

	if g.analysisResult.Documentation == nil || g.analysisResult.Documentation.ReadmeContent == "" {
		return steps
	}

	readme := g.analysisResult.Documentation.ReadmeContent
	lines := strings.Split(readme, "\n")
	inInstallSection := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		lower := strings.ToLower(trimmed)

		// Detect installation section
		if strings.HasPrefix(trimmed, "#") && (strings.Contains(lower, "install") || strings.Contains(lower, "setup") || strings.Contains(lower, "getting started")) {
			inInstallSection = true
			continue
		}

		// Exit installation section on next header
		if inInstallSection && strings.HasPrefix(trimmed, "#") {
			break
		}

		// Extract numbered steps or bullet points
		if inInstallSection && trimmed != "" {
			if strings.HasPrefix(trimmed, "-") || strings.HasPrefix(trimmed, "*") || strings.HasPrefix(trimmed, "1.") || strings.HasPrefix(trimmed, "2.") {
				// Remove bullet/number prefix
				step := strings.TrimLeft(trimmed, "-*123456789. ")
				if step != "" {
					steps = append(steps, step)
				}
			}
		}
	}

	return steps
}

// generateDefaultInstallationSteps generates default installation steps based on project type
func (g *DocumentationGenerator) generateDefaultInstallationSteps() string {
	var sb strings.Builder

	if g.analysisResult.Configuration == nil {
		sb.WriteString("1. Clone the repository\n")
		sb.WriteString("2. Follow the project-specific setup instructions\n\n")
		return sb.String()
	}

	// Generate steps based on detected package managers
	for _, manifest := range g.analysisResult.Configuration.PackageManifests {
		if strings.Contains(manifest, "go.mod") {
			sb.WriteString("1. Clone the repository\n")
			sb.WriteString("2. Run `go mod download` to install dependencies\n")
			sb.WriteString("3. Run `go build` to build the application\n")
			sb.WriteString("4. Run the compiled binary\n\n")
			return sb.String()
		} else if strings.Contains(manifest, "package.json") {
			sb.WriteString("1. Clone the repository\n")
			sb.WriteString("2. Run `npm install` to install dependencies\n")
			sb.WriteString("3. Run `npm start` to start the application\n\n")
			return sb.String()
		} else if strings.Contains(manifest, "requirements.txt") {
			sb.WriteString("1. Clone the repository\n")
			sb.WriteString("2. Create a virtual environment: `python -m venv venv`\n")
			sb.WriteString("3. Activate the virtual environment\n")
			sb.WriteString("4. Run `pip install -r requirements.txt` to install dependencies\n")
			sb.WriteString("5. Run the application\n\n")
			return sb.String()
		}
	}

	sb.WriteString("1. Clone the repository\n")
	sb.WriteString("2. Follow the project-specific setup instructions\n\n")
	return sb.String()
}

// groupByPackage groups code elements by package
func (g *DocumentationGenerator) groupByPackage() map[string]*packageItems {
	packageMap := make(map[string]*packageItems)

	if g.analysisResult.CodeStructure == nil {
		return packageMap
	}

	// Group functions
	for _, fn := range g.analysisResult.CodeStructure.Functions {
		pkg := fn.Package
		if pkg == "" {
			pkg = "main"
		}
		if _, exists := packageMap[pkg]; !exists {
			packageMap[pkg] = &packageItems{}
		}
		packageMap[pkg].Functions = append(packageMap[pkg].Functions, fn)
	}

	// Group structs
	for _, st := range g.analysisResult.CodeStructure.Structs {
		pkg := st.Package
		if pkg == "" {
			pkg = "main"
		}
		if _, exists := packageMap[pkg]; !exists {
			packageMap[pkg] = &packageItems{}
		}
		packageMap[pkg].Structs = append(packageMap[pkg].Structs, st)
	}

	// Group interfaces
	for _, iface := range g.analysisResult.CodeStructure.Interfaces {
		pkg := iface.Package
		if pkg == "" {
			pkg = "main"
		}
		if _, exists := packageMap[pkg]; !exists {
			packageMap[pkg] = &packageItems{}
		}
		packageMap[pkg].Interfaces = append(packageMap[pkg].Interfaces, iface)
	}

	return packageMap
}
