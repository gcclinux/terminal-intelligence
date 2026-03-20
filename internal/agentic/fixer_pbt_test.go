package agentic

import (
	"strings"
	"testing"

	"github.com/user/terminal-intelligence/internal/types"
	"pgregory.net/rapid"
)

// pbtMockAIClient is a minimal mock for NewAgenticCodeFixer in PBT tests.
type pbtMockAIClient struct{}

func (m *pbtMockAIClient) IsAvailable() (bool, error) { return true, nil }
func (m *pbtMockAIClient) Generate(prompt string, model string, context []int, onTokenUsage func(types.TokenUsage)) (<-chan string, error) {
	return nil, nil
}
func (m *pbtMockAIClient) ListModels() ([]string, error) { return nil, nil }

// validFileTypes lists the file types accepted by FixRequest.Validate.
var validFileTypes = []string{"bash", "shell", "powershell", "markdown", "python", "go"}

// validOperationTypes lists the operation types accepted by EditIntent.Validate.
var validOperationTypes = []string{"replace", "append", "insert", "patch"}

// genFixRequest generates a random valid FixRequest.
func genFixRequest(t *rapid.T) *FixRequest {
	// UserMessage must be non-empty after trimming
	msg := rapid.StringMatching(`[a-zA-Z0-9 ]{1,80}`).Draw(t, "userMessage")
	content := rapid.StringMatching(`[a-zA-Z0-9 \n]{0,200}`).Draw(t, "fileContent")
	filePath := rapid.StringMatching(`[a-z]{1,10}\.[a-z]{1,4}`).Draw(t, "filePath")
	fileType := rapid.SampledFrom(validFileTypes).Draw(t, "fileType")

	return &FixRequest{
		UserMessage: msg,
		FileContent: content,
		FilePath:    filePath,
		FileType:    fileType,
	}
}

// genEditIntent generates a random valid EditIntent.
func genEditIntent(t *rapid.T) EditIntent {
	opType := rapid.SampledFrom(validOperationTypes).Draw(t, "operationType")
	confidence := rapid.Float64Range(0.0, 1.0).Draw(t, "confidence")
	return EditIntent{
		OperationType: opType,
		Confidence:    confidence,
	}
}

// Feature: content-preserving-file-updates, Property 6: Prompt includes intent-specific instructions and file content
//
// For any valid FixRequest and for any valid EditIntent, the string returned by
// BuildPrompt(request, intent) should contain the request.FileContent (or "(empty file)"
// if content is empty) AND should contain intent-specific instruction text.
//
// - For append intent: prompt should contain "Generate ONLY the new content to append"
// - For insert intent: prompt should contain "Generate ONLY the new content to insert"
// - For replace intent: prompt should contain "Generate the complete fixed code"
// - For patch intent: prompt should contain "Generate the complete fixed code"
//
// **Validates: Requirements 2.1, 2.2, 2.3, 2.4, 2.5**
func TestProperty6_PromptIncludesIntentInstructionsAndFileContent(t *testing.T) {
	fixer := NewAgenticCodeFixer(&pbtMockAIClient{}, "test-model")

	rapid.Check(t, func(t *rapid.T) {
		request := genFixRequest(t)
		intent := genEditIntent(t)

		prompt := fixer.BuildPrompt(request, intent)

		// Req 2.5: prompt always contains file content or "(empty file)"
		if strings.TrimSpace(request.FileContent) == "" {
			if !strings.Contains(prompt, "(empty file)") {
				t.Fatalf("prompt should contain '(empty file)' for empty content, got:\n%s", prompt)
			}
		} else {
			if !strings.Contains(prompt, request.FileContent) {
				t.Fatalf("prompt should contain file content %q, got:\n%s", request.FileContent, prompt)
			}
		}

		// Check intent-specific instruction text
		switch intent.OperationType {
		case "append":
			if !strings.Contains(prompt, "Generate ONLY the new content to append") {
				t.Fatalf("append prompt missing expected instruction, got:\n%s", prompt)
			}
		case "insert":
			if !strings.Contains(prompt, "Generate ONLY the new content to insert") {
				t.Fatalf("insert prompt missing expected instruction, got:\n%s", prompt)
			}
		case "replace":
			if !strings.Contains(prompt, "Generate the complete fixed code") {
				t.Fatalf("replace prompt missing expected instruction, got:\n%s", prompt)
			}
		case "patch":
			if !strings.Contains(prompt, "Generate the complete fixed code") {
				t.Fatalf("patch prompt missing expected instruction, got:\n%s", prompt)
			}
		}
	})
}

// Feature: content-preserving-file-updates, Property 12: Code block extraction round-trip
//
// For any non-empty code string and for any language identifier, wrapping the code
// in markdown fences (```lang\ncode\n```) and then calling FixParser.ExtractCodeBlocks
// should return at least one CodeBlock whose Code field equals the original code string
// (after trimming trailing newlines, since ExtractCodeBlocks trims them).
//
// **Validates: Requirements 7.1, 7.3, 7.4**
func TestProperty12_CodeBlockExtractionRoundTrip(t *testing.T) {
	parser := NewFixParser()

	rapid.Check(t, func(t *rapid.T) {
		// Generate a language identifier matching the regex [a-zA-Z0-9_+-]*
		lang := rapid.StringMatching(`[a-zA-Z0-9_+\-]{1,12}`).Draw(t, "language")

		// Generate code that is non-empty after trimming and does not contain triple backticks.
		// Use safe characters: letters, digits, spaces, newlines, common punctuation.
		code := rapid.StringMatching(`[a-zA-Z0-9 \t=(){};:,.\n]{1,120}`).Draw(t, "code")

		// Ensure code is non-empty after trimming whitespace
		if strings.TrimSpace(code) == "" {
			t.Skip("skipping empty code after trim")
		}

		// Wrap in markdown fences
		markdown := "```" + lang + "\n" + code + "\n```"

		blocks := parser.ExtractCodeBlocks(markdown)

		// Must extract at least one block
		if len(blocks) == 0 {
			t.Fatalf("expected at least one code block, got none for markdown:\n%s", markdown)
		}

		// The extracted Code should equal the original code with trailing newlines trimmed
		// (ExtractCodeBlocks does strings.TrimRight(code, "\n"))
		expected := strings.TrimRight(code, "\n")
		found := false
		for _, block := range blocks {
			if block.Code == expected {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("no extracted block matched expected code.\nExpected: %q\nGot blocks: %v", expected, blocks)
		}
	})
}
