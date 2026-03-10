package docgen

import (
	"errors"
	"regexp"
	"strings"
)

// CommandParser parses user input to identify documentation generation requests
type CommandParser struct {
	// Stateless parser - no fields needed
}

// NewCommandParser creates a new CommandParser instance
func NewCommandParser() *CommandParser {
	return &CommandParser{}
}

// Parse parses user input and extracts command flags, natural language, and scope filters
func (p *CommandParser) Parse(input string) (*ParsedCommand, error) {
	// Validate input
	if strings.TrimSpace(input) == "" {
		return nil, errors.New("no command provided")
	}

	result := &ParsedCommand{
		IsProjectWide:   false,
		IsDocRequest:    false,
		NaturalLanguage: "",
		ScopeFilters:    []string{},
	}

	// Normalize input for flag detection (case-insensitive)
	lowerInput := strings.ToLower(input)

	// Detect /doc flag - this is now the primary flag for documentation generation
	if strings.Contains(lowerInput, "/doc") {
		result.IsDocRequest = true
		result.IsProjectWide = true // /doc always implies project-wide analysis
	}

	// Validate that /doc flag is present
	if !result.IsDocRequest {
		return nil, errors.New("no documentation flags detected")
	}

	// Extract natural language by removing flags
	naturalLanguage := input

	// Remove /project flag if present (for backward compatibility)
	projectFlagRegex := regexp.MustCompile(`(?i)/project\s*`)
	naturalLanguage = projectFlagRegex.ReplaceAllString(naturalLanguage, "")

	// Remove /doc flag (case-insensitive)
	docFlagRegex := regexp.MustCompile(`(?i)/doc\s*`)
	naturalLanguage = docFlagRegex.ReplaceAllString(naturalLanguage, "")

	// Clean up extra whitespace
	naturalLanguage = strings.TrimSpace(naturalLanguage)
	naturalLanguage = regexp.MustCompile(`\s+`).ReplaceAllString(naturalLanguage, " ")

	result.NaturalLanguage = naturalLanguage

	// Extract scope filters
	result.ScopeFilters = p.extractScopeFilters(naturalLanguage)

	return result, nil
}

// extractScopeFilters extracts scope filters from natural language text
// Looks for patterns like "for module X", "in directory Y", "for package Z"
func (p *CommandParser) extractScopeFilters(text string) []string {
	filters := []string{}

	// Pattern: "for module X", "for package X", "in directory X", "in folder X"
	patterns := []string{
		`(?i)for\s+module\s+([^\s,]+)`,
		`(?i)for\s+package\s+([^\s,]+)`,
		`(?i)in\s+directory\s+([^\s,]+)`,
		`(?i)in\s+folder\s+([^\s,]+)`,
		`(?i)for\s+([^\s,]+)\s+module`,
		`(?i)for\s+([^\s,]+)\s+package`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(text, -1)
		for _, match := range matches {
			if len(match) > 1 {
				filter := strings.TrimSpace(match[1])
				if filter != "" {
					filters = append(filters, filter)
				}
			}
		}
	}

	return filters
}
