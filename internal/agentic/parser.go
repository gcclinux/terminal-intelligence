package agentic

import (
	"fmt"
	"log"
	"regexp"
	"strings"
)

// FixParser extracts and validates code fixes from AI responses.
// It provides methods to parse markdown-formatted AI responses, identify which code blocks
// represent actual fixes (versus examples or explanations), and validate syntax for different
// file types (bash, shell, powershell, markdown).
//
// The parser uses heuristics to distinguish fix blocks from explanatory code:
//   - Blocks matching the target file type are prioritized
//   - Larger blocks are more likely to be complete fixes
//   - Multiple blocks matching the file type support multi-step fixes
//
// Syntax validation is performed using language-specific rules:
//   - Bash/Shell: Checks for unmatched quotes, brackets, and control structures (if/fi, do/done, case/esac)
//   - PowerShell: Checks for unmatched quotes, brackets, and try/catch/finally blocks
//   - Markdown: Checks for unmatched code block markers
type FixParser struct {
	debug bool // Enable debug logging
}

// NewFixParser creates a new fix parser
func NewFixParser() *FixParser {
	return &FixParser{
		debug: false,
	}
}

// SetDebug enables or disables debug logging
func (p *FixParser) SetDebug(debug bool) {
	p.debug = debug
}

// logDebug logs debug messages (only when debug mode is enabled)
func (p *FixParser) logDebug(format string, args ...interface{}) {
	if p.debug {
		log.Printf("[DEBUG] FixParser: "+format, args...)
	}
}

// logError logs error messages (only when debug mode is enabled)
func (p *FixParser) logError(format string, args ...interface{}) {
	if p.debug {
		log.Printf("[ERROR] FixParser: "+format, args...)
	}
}

// ExtractCodeBlocks extracts all code blocks from markdown-formatted text
// Handles various markdown formats:
// - ```language\ncode\n```
// - ```\ncode\n```
// Returns a slice of CodeBlock structs
func (p *FixParser) ExtractCodeBlocks(response string) []CodeBlock {
	p.logDebug("Extracting code blocks from response (length: %d chars)", len(response))
	
	var blocks []CodeBlock
	
	// Regular expression to match markdown code blocks
	// Matches: ```optional-language\ncode content\n```
	codeBlockRegex := regexp.MustCompile("(?s)```([a-zA-Z0-9_+-]*)\n(.*?)```")
	
	matches := codeBlockRegex.FindAllStringSubmatch(response, -1)
	p.logDebug("Found %d potential code blocks", len(matches))
	
	for i, match := range matches {
		if len(match) >= 3 {
			language := strings.TrimSpace(match[1])
			code := match[2]
			
			// Skip empty code blocks
			if strings.TrimSpace(code) == "" {
				p.logDebug("Skipping empty code block %d", i+1)
				continue
			}
			
			// Remove trailing newline if present
			code = strings.TrimRight(code, "\n")
			
			p.logDebug("Extracted code block %d: language=%s, size=%d bytes", i+1, language, len(code))
			
			blocks = append(blocks, CodeBlock{
				Language: language,
				Code:     code,
				IsWhole:  false, // Will be determined by IdentifyFixBlocks
			})
		}
	}
	
	p.logDebug("Extracted %d valid code blocks", len(blocks))
	return blocks
}
// IdentifyFixBlocks determines which code blocks represent fixes versus explanations
// It analyzes the blocks and their context to distinguish actual code fixes from examples
// or explanatory code snippets. Returns only the blocks that should be applied as fixes.
//
// Heuristics used:
// - Blocks matching the file type are more likely to be fixes
// - Larger blocks are more likely to be complete fixes
// - All substantial blocks matching the file type are considered fixes (for multi-step fixes)
// - Blocks with generic/unspecified language may be fixes if they match file content patterns
func (p *FixParser) IdentifyFixBlocks(blocks []CodeBlock, fileType string) []CodeBlock {
	p.logDebug("Identifying fix blocks for file type: %s (total blocks: %d)", fileType, len(blocks))
	
	if len(blocks) == 0 {
		return []CodeBlock{}
	}

	var fixBlocks []CodeBlock

	// Normalize file type for comparison
	normalizedFileType := normalizeFileType(fileType)
	p.logDebug("Normalized file type: %s", normalizedFileType)

	// Strategy: Find all blocks that match the file type
	// This supports multi-step fixes where multiple code blocks need to be applied
	for i, block := range blocks {
		normalizedLang := normalizeLanguage(block.Language)

		// Check if this block matches the file type
		if languageMatchesFileType(normalizedLang, normalizedFileType) {
			// Mark as whole file replacement if it's substantial
			// (more than 3 lines suggests a complete implementation)
			lineCount := countLines(block.Code)
			block.IsWhole = lineCount > 3

			p.logDebug("Block %d matches file type (language: %s, lines: %d, isWhole: %v)", 
				i+1, block.Language, lineCount, block.IsWhole)
			
			fixBlocks = append(fixBlocks, block)
		} else {
			p.logDebug("Block %d does not match file type (language: %s vs %s)", 
				i+1, normalizedLang, normalizedFileType)
		}
	}

	// If no blocks matched the file type, check for unspecified language blocks
	// These might be fixes where the AI didn't specify the language
	if len(fixBlocks) == 0 {
		p.logDebug("No blocks matched file type, checking for unspecified language blocks")
		for i, block := range blocks {
			if block.Language == "" {
				// Assume it's a fix if it's substantial
				lineCount := countLines(block.Code)
				if lineCount > 2 {
					block.IsWhole = lineCount > 3
					p.logDebug("Using unspecified language block %d as fix (lines: %d)", i+1, lineCount)
					fixBlocks = append(fixBlocks, block)
					break
				}
			}
		}
	}

	p.logDebug("Identified %d fix blocks", len(fixBlocks))
	return fixBlocks
}

// normalizeFileType converts file type to a standard form for comparison
func normalizeFileType(fileType string) string {
	fileType = strings.ToLower(strings.TrimSpace(fileType))

	// Map file types to standard language identifiers
	switch fileType {
	case "bash", "shell":
		return "bash"
	case "powershell", "ps1":
		return "powershell"
	case "markdown", "md":
		return "markdown"
	default:
		return fileType
	}
}

// normalizeLanguage converts language identifier to standard form
func normalizeLanguage(language string) string {
	language = strings.ToLower(strings.TrimSpace(language))

	// Map common language identifiers to standard forms
	switch language {
	case "sh", "bash", "shell":
		return "bash"
	case "ps", "ps1", "powershell":
		return "powershell"
	case "md", "markdown":
		return "markdown"
	default:
		return language
	}
}

// languageMatchesFileType checks if a language identifier matches a file type
func languageMatchesFileType(language, fileType string) bool {
	if language == "" {
		return false
	}

	// Direct match
	if language == fileType {
		return true
	}

	// Special cases for shell scripts
	if fileType == "bash" && (language == "sh" || language == "shell") {
		return true
	}

	if fileType == "shell" && (language == "bash" || language == "sh") {
		return true
	}

	return false
}

// countLines counts the number of lines in a code string
func countLines(code string) int {
	if code == "" {
		return 0
	}

	lines := strings.Split(code, "\n")

	// Count non-empty lines
	count := 0
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}

	return count
}

// ValidateFixSyntax performs basic syntax validation on a fix
// Validates that the code is syntactically appropriate for the file type
// Supports: bash, shell, powershell, markdown
// Returns an error if validation fails, nil if validation passes
func (p *FixParser) ValidateFixSyntax(code string, fileType string) error {
	p.logDebug("Validating syntax for file type: %s (code length: %d bytes)", fileType, len(code))
	
	if strings.TrimSpace(code) == "" {
		p.logError("Code is empty")
		return fmt.Errorf("code cannot be empty")
	}

	normalizedFileType := normalizeFileType(fileType)

	switch normalizedFileType {
	case "bash":
		p.logDebug("Performing bash syntax validation")
		return validateBashSyntax(code)
	case "powershell":
		p.logDebug("Performing PowerShell syntax validation")
		return validatePowerShellSyntax(code)
	case "markdown":
		p.logDebug("Performing markdown syntax validation")
		return validateMarkdownSyntax(code)
	default:
		// For unknown file types, perform minimal validation
		p.logDebug("Unknown file type, skipping syntax validation")
		return nil
	}
}

// validateBashSyntax performs basic bash/shell syntax validation
func validateBashSyntax(code string) error {
	// Basic checks for common bash syntax errors
	
	// Check for unmatched quotes
	if err := checkUnmatchedQuotes(code); err != nil {
		return fmt.Errorf("bash syntax error: %w", err)
	}

	// Check for unmatched braces/brackets/parentheses
	if err := checkUnmatchedBrackets(code); err != nil {
		return fmt.Errorf("bash syntax error: %w", err)
	}

	// Check for incomplete control structures
	if err := checkBashControlStructures(code); err != nil {
		return fmt.Errorf("bash syntax error: %w", err)
	}

	return nil
}

// validatePowerShellSyntax performs basic PowerShell syntax validation
func validatePowerShellSyntax(code string) error {
	// Basic checks for common PowerShell syntax errors
	
	// Check for unmatched quotes
	if err := checkUnmatchedQuotes(code); err != nil {
		return fmt.Errorf("powershell syntax error: %w", err)
	}

	// Check for unmatched braces/brackets/parentheses
	if err := checkUnmatchedBrackets(code); err != nil {
		return fmt.Errorf("powershell syntax error: %w", err)
	}

	// Check for incomplete control structures
	if err := checkPowerShellControlStructures(code); err != nil {
		return fmt.Errorf("powershell syntax error: %w", err)
	}

	return nil
}

// validateMarkdownSyntax performs basic markdown syntax validation
func validateMarkdownSyntax(code string) error {
	// Basic checks for markdown syntax
	
	// Check for unmatched code block markers
	backtickCount := strings.Count(code, "```")
	if backtickCount%2 != 0 {
		return fmt.Errorf("markdown syntax error: unmatched code block markers (```)")
	}

	// Markdown is generally forgiving, so we only check critical issues
	return nil
}

// checkUnmatchedQuotes checks for unmatched single or double quotes
func checkUnmatchedQuotes(code string) error {
	inSingleQuote := false
	inDoubleQuote := false
	escaped := false

	for i, ch := range code {
		if escaped {
			escaped = false
			continue
		}

		if ch == '\\' {
			escaped = true
			continue
		}

		if ch == '\'' && !inDoubleQuote {
			inSingleQuote = !inSingleQuote
		} else if ch == '"' && !inSingleQuote {
			inDoubleQuote = !inDoubleQuote
		}

		// Check for newline inside quotes (common error)
		if ch == '\n' && (inSingleQuote || inDoubleQuote) {
			// Allow multiline strings in some contexts
			// This is a simplified check
		}

		// Prevent index out of bounds
		_ = i
	}

	if inSingleQuote {
		return fmt.Errorf("unmatched single quote")
	}

	if inDoubleQuote {
		return fmt.Errorf("unmatched double quote")
	}

	return nil
}

// checkUnmatchedBrackets checks for unmatched braces, brackets, and parentheses
func checkUnmatchedBrackets(code string) error {
	// Stack to track opening brackets
	stack := []rune{}
	inSingleQuote := false
	inDoubleQuote := false
	escaped := false
	inCasePattern := false

	matchingBracket := map[rune]rune{
		')': '(',
		'}': '{',
		']': '[',
	}

	lines := strings.Split(code, "\n")
	
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		
		// Check if we're in a case statement pattern (ends with ')')
		// Case patterns look like: pattern) command;;
		if strings.Contains(trimmed, ";;") {
			// This line likely contains case patterns, skip bracket checking for ')' in patterns
			inCasePattern = true
		}
		
		for _, ch := range line {
			// Skip characters inside quotes
			if escaped {
				escaped = false
				continue
			}

			if ch == '\\' {
				escaped = true
				continue
			}

			if ch == '\'' && !inDoubleQuote {
				inSingleQuote = !inSingleQuote
				continue
			}

			if ch == '"' && !inSingleQuote {
				inDoubleQuote = !inDoubleQuote
				continue
			}

			if inSingleQuote || inDoubleQuote {
				continue
			}

			// Track opening brackets
			if ch == '(' || ch == '{' || ch == '[' {
				stack = append(stack, ch)
			}

			// Check closing brackets
			if ch == ')' || ch == '}' || ch == ']' {
				// Special handling for case patterns - allow unmatched ')' if we see ';;' on the line
				if ch == ')' && inCasePattern {
					// Try to pop from stack, but don't error if stack is empty (case pattern)
					if len(stack) > 0 && stack[len(stack)-1] == '(' {
						stack = stack[:len(stack)-1]
					}
					continue
				}
				
				if len(stack) == 0 {
					return fmt.Errorf("unmatched closing bracket: %c", ch)
				}

				// Pop from stack and check if it matches
				top := stack[len(stack)-1]
				stack = stack[:len(stack)-1]

				if top != matchingBracket[ch] {
					return fmt.Errorf("mismatched brackets: expected closing for %c, got %c", top, ch)
				}
			}
		}
		
		// Reset case pattern flag at end of line
		inCasePattern = false
	}

	if len(stack) > 0 {
		return fmt.Errorf("unmatched opening bracket: %c", stack[len(stack)-1])
	}

	return nil
}

// checkBashControlStructures checks for incomplete bash control structures
func checkBashControlStructures(code string) error {
	// Count control structure keywords
	lines := strings.Split(code, "\n")
	
	ifCount := 0
	fiCount := 0
	doCount := 0
	doneCount := 0
	caseCount := 0
	esacCount := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		
		// Skip comments
		if strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Check for control structure keywords
		// Use word boundaries to avoid false matches
		words := strings.Fields(trimmed)
		for _, word := range words {
			switch word {
			case "if":
				ifCount++
			case "fi":
				fiCount++
			case "do":
				doCount++
			case "done":
				doneCount++
			case "case":
				caseCount++
			case "esac":
				esacCount++
			}
		}
	}

	// Check for matching control structures
	if ifCount != fiCount {
		return fmt.Errorf("unmatched if/fi statements (if: %d, fi: %d)", ifCount, fiCount)
	}

	if doCount != doneCount {
		return fmt.Errorf("unmatched do/done statements (do: %d, done: %d)", doCount, doneCount)
	}

	if caseCount != esacCount {
		return fmt.Errorf("unmatched case/esac statements (case: %d, esac: %d)", caseCount, esacCount)
	}

	return nil
}

// checkPowerShellControlStructures checks for incomplete PowerShell control structures
func checkPowerShellControlStructures(code string) error {
	// PowerShell uses braces for control structures, which are already checked
	// by checkUnmatchedBrackets. We can add more specific checks here if needed.
	
	// For now, we'll just ensure basic structure
	// PowerShell is case-insensitive, so we normalize to lowercase
	lowerCode := strings.ToLower(code)
	
	// Check for incomplete try/catch/finally blocks
	tryCount := strings.Count(lowerCode, "try")
	catchCount := strings.Count(lowerCode, "catch")
	finallyCount := strings.Count(lowerCode, "finally")
	
	// Try blocks should have at least one catch or finally
	if tryCount > 0 && (catchCount+finallyCount) == 0 {
		return fmt.Errorf("try block without catch or finally")
	}

	return nil
}

