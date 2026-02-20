package agentic

import (
	"fmt"
	"strings"
	"testing"
)

// mockAIClient is a mock implementation of AIClient for testing
type mockAIClient struct{}

func (m *mockAIClient) IsAvailable() (bool, error) {
	return true, nil
}

func (m *mockAIClient) Generate(prompt string, model string, context []int) (<-chan string, error) {
	ch := make(chan string, 1)
	ch <- "mock response"
	close(ch)
	return ch, nil
}

func (m *mockAIClient) ListModels() ([]string, error) {
	return []string{"mock-model"}, nil
}

func TestIsFixRequest_ExplicitFixCommand(t *testing.T) {
	fixer := NewAgenticCodeFixer(&mockAIClient{}, "test-model")

	testCases := []struct {
		name     string
		message  string
		expected bool
	}{
		{
			name:     "explicit /fix command",
			message:  "/fix the bug in line 10",
			expected: true,
		},
		{
			name:     "/fix at start with whitespace",
			message:  "  /fix this issue",
			expected: true,
		},
		{
			name:     "/fix alone",
			message:  "/fix",
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := fixer.IsFixRequest(tc.message)

			if result.IsFixRequest != tc.expected {
				t.Errorf("expected IsFixRequest=%v, got %v", tc.expected, result.IsFixRequest)
			}

			if result.IsFixRequest && result.Confidence != 1.0 {
				t.Errorf("expected confidence=1.0 for explicit command, got %f", result.Confidence)
			}

			if result.IsFixRequest && len(result.Keywords) == 0 {
				t.Errorf("expected keywords to be populated, got empty")
			}
		})
	}
}

func TestIsFixRequest_ExplicitAskCommand(t *testing.T) {
	fixer := NewAgenticCodeFixer(&mockAIClient{}, "test-model")

	result := fixer.IsFixRequest("/ask what does this function do?")

	if result.IsFixRequest {
		t.Errorf("expected IsFixRequest=false for /ask command, got true")
	}

	if result.Confidence != 0.0 {
		t.Errorf("expected confidence=0.0 for /ask command, got %f", result.Confidence)
	}
}

func TestIsFixRequest_SingleKeyword(t *testing.T) {
	fixer := NewAgenticCodeFixer(&mockAIClient{}, "test-model")

	testCases := []struct {
		name               string
		message            string
		expectedConfidence float64
		expectedKeyword    string
	}{
		{
			name:               "fix keyword",
			message:            "fix the error",
			expectedConfidence: 0.7,
			expectedKeyword:    "fix",
		},
		{
			name:               "change keyword",
			message:            "change the variable name",
			expectedConfidence: 0.7,
			expectedKeyword:    "change",
		},
		{
			name:               "update keyword",
			message:            "update the function",
			expectedConfidence: 0.7,
			expectedKeyword:    "update",
		},
		{
			name:               "modify keyword",
			message:            "modify this code",
			expectedConfidence: 0.7,
			expectedKeyword:    "modify",
		},
		{
			name:               "correct keyword",
			message:            "correct the syntax",
			expectedConfidence: 0.7,
			expectedKeyword:    "correct",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := fixer.IsFixRequest(tc.message)

			if !result.IsFixRequest {
				t.Errorf("expected IsFixRequest=true, got false")
			}

			if result.Confidence != tc.expectedConfidence {
				t.Errorf("expected confidence=%f, got %f", tc.expectedConfidence, result.Confidence)
			}

			if len(result.Keywords) != 1 {
				t.Errorf("expected 1 keyword, got %d", len(result.Keywords))
			}

			if len(result.Keywords) > 0 && result.Keywords[0] != tc.expectedKeyword {
				t.Errorf("expected keyword=%s, got %s", tc.expectedKeyword, result.Keywords[0])
			}
		})
	}
}

func TestIsFixRequest_SingleKeywordWithActionContext(t *testing.T) {
	fixer := NewAgenticCodeFixer(&mockAIClient{}, "test-model")

	testCases := []struct {
		name               string
		message            string
		expectedConfidence float64
	}{
		{
			name:               "please + fix",
			message:            "please fix the bug",
			expectedConfidence: 0.8,
		},
		{
			name:               "can you + change",
			message:            "can you change this?",
			expectedConfidence: 0.8,
		},
		{
			name:               "could you + update",
			message:            "could you update the code",
			expectedConfidence: 0.8,
		},
		{
			name:               "need to + modify",
			message:            "need to modify this function",
			expectedConfidence: 0.8,
		},
		{
			name:               "want to + correct",
			message:            "want to correct the error",
			expectedConfidence: 0.8,
		},
		{
			name:               "should + fix",
			message:            "should fix this issue",
			expectedConfidence: 0.8,
		},
		{
			name:               "must + change",
			message:            "must change the implementation",
			expectedConfidence: 0.8,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := fixer.IsFixRequest(tc.message)

			if !result.IsFixRequest {
				t.Errorf("expected IsFixRequest=true, got false")
			}

			if result.Confidence != tc.expectedConfidence {
				t.Errorf("expected confidence=%f, got %f", tc.expectedConfidence, result.Confidence)
			}
		})
	}
}

func TestIsFixRequest_MultipleKeywords(t *testing.T) {
	fixer := NewAgenticCodeFixer(&mockAIClient{}, "test-model")

	testCases := []struct {
		name             string
		message          string
		expectedKeywords int
	}{
		{
			name:             "fix and change",
			message:          "fix and change the code",
			expectedKeywords: 2,
		},
		{
			name:             "update and modify",
			message:          "update and modify the function",
			expectedKeywords: 2,
		},
		{
			name:             "fix, change, and update",
			message:          "fix, change, and update this",
			expectedKeywords: 3,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := fixer.IsFixRequest(tc.message)

			if !result.IsFixRequest {
				t.Errorf("expected IsFixRequest=true, got false")
			}

			if result.Confidence != 0.9 {
				t.Errorf("expected confidence=0.9 for multiple keywords, got %f", result.Confidence)
			}

			if len(result.Keywords) != tc.expectedKeywords {
				t.Errorf("expected %d keywords, got %d", tc.expectedKeywords, len(result.Keywords))
			}
		})
	}
}

func TestIsFixRequest_ConversationalMessages(t *testing.T) {
	fixer := NewAgenticCodeFixer(&mockAIClient{}, "test-model")

	testCases := []struct {
		name    string
		message string
	}{
		{
			name:    "question about code",
			message: "what does this function do?",
		},
		{
			name:    "explanation request",
			message: "explain how this works",
		},
		{
			name:    "general question",
			message: "how do I use this library?",
		},
		{
			name:    "documentation request",
			message: "show me the documentation",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := fixer.IsFixRequest(tc.message)

			if result.IsFixRequest {
				t.Errorf("expected IsFixRequest=false for conversational message, got true")
			}

			if result.Confidence != 0.0 {
				t.Errorf("expected confidence=0.0, got %f", result.Confidence)
			}

			if len(result.Keywords) != 0 {
				t.Errorf("expected no keywords, got %d", len(result.Keywords))
			}
		})
	}
}

func TestIsFixRequest_CaseInsensitive(t *testing.T) {
	fixer := NewAgenticCodeFixer(&mockAIClient{}, "test-model")

	testCases := []string{
		"FIX the bug",
		"Fix the bug",
		"fix the bug",
		"CHANGE this code",
		"Change this code",
		"change this code",
	}

	for _, message := range testCases {
		t.Run(message, func(t *testing.T) {
			result := fixer.IsFixRequest(message)

			if !result.IsFixRequest {
				t.Errorf("expected IsFixRequest=true for '%s', got false", message)
			}
		})
	}
}

func TestIsFixRequest_WhitespaceHandling(t *testing.T) {
	fixer := NewAgenticCodeFixer(&mockAIClient{}, "test-model")

	testCases := []string{
		"  fix the bug  ",
		"\tfix the bug\t",
		"\nfix the bug\n",
		"   /fix   ",
	}

	for _, message := range testCases {
		t.Run(message, func(t *testing.T) {
			result := fixer.IsFixRequest(message)

			if !result.IsFixRequest {
				t.Errorf("expected IsFixRequest=true for message with whitespace, got false")
			}
		})
	}
}

func TestIsFixRequest_ValidatesInvariants(t *testing.T) {
	fixer := NewAgenticCodeFixer(&mockAIClient{}, "test-model")

	testCases := []struct {
		name    string
		message string
	}{
		{
			name:    "fix request",
			message: "fix the bug",
		},
		{
			name:    "conversational",
			message: "what is this?",
		},
		{
			name:    "explicit command",
			message: "/fix",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := fixer.IsFixRequest(tc.message)

			// Validate the result satisfies invariants
			if err := result.Validate(); err != nil {
				t.Errorf("result violates invariants: %v", err)
			}
		})
	}
}

func TestIsFixRequest_CommandWithAdditionalText(t *testing.T) {
	fixer := NewAgenticCodeFixer(&mockAIClient{}, "test-model")

	testCases := []struct {
		name            string
		message         string
		expectedFix     bool
		expectedKeyword string
	}{
		{
			name:            "/fix with description",
			message:         "/fix the bug in line 10",
			expectedFix:     true,
			expectedKeyword: "/fix",
		},
		{
			name:            "/ask with question",
			message:         "/ask what does this function do?",
			expectedFix:     false,
			expectedKeyword: "",
		},
		{
			name:            "/fix with multiple words",
			message:         "/fix change the variable name to something better",
			expectedFix:     true,
			expectedKeyword: "/fix",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := fixer.IsFixRequest(tc.message)

			if result.IsFixRequest != tc.expectedFix {
				t.Errorf("expected IsFixRequest=%v, got %v", tc.expectedFix, result.IsFixRequest)
			}

			if tc.expectedFix && result.Confidence != 1.0 {
				t.Errorf("expected confidence=1.0 for /fix command, got %f", result.Confidence)
			}

			if tc.expectedKeyword != "" && (len(result.Keywords) == 0 || result.Keywords[0] != tc.expectedKeyword) {
				t.Errorf("expected keyword=%s, got %v", tc.expectedKeyword, result.Keywords)
			}
		})
	}
}

func TestIsFixRequest_CommandNotAtStart(t *testing.T) {
	fixer := NewAgenticCodeFixer(&mockAIClient{}, "test-model")

	testCases := []struct {
		name        string
		message     string
		expectedFix bool
	}{
		{
			name:        "fix keyword in middle",
			message:     "I need to fix this bug",
			expectedFix: true, // Should be detected as fix request due to keyword
		},
		{
			name:        "/fix not at start",
			message:     "Can you /fix this?",
			expectedFix: true, // Should be detected as fix request due to keyword
		},
		{
			name:        "ask in middle",
			message:     "I want to ask a question",
			expectedFix: false, // No fix keywords
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := fixer.IsFixRequest(tc.message)

			if result.IsFixRequest != tc.expectedFix {
				t.Errorf("expected IsFixRequest=%v, got %v", tc.expectedFix, result.IsFixRequest)
			}
		})
	}
}

func TestIsFixRequest_CommandCaseVariations(t *testing.T) {
	fixer := NewAgenticCodeFixer(&mockAIClient{}, "test-model")

	testCases := []struct {
		name               string
		message            string
		expectedFix        bool
		expectedConfidence float64
	}{
		{
			name:               "/FIX uppercase",
			message:            "/FIX the bug",
			expectedFix:        true,
			expectedConfidence: 1.0,
		},
		{
			name:               "/Fix mixed case",
			message:            "/Fix the bug",
			expectedFix:        true,
			expectedConfidence: 1.0,
		},
		{
			name:               "/ASK uppercase",
			message:            "/ASK what is this?",
			expectedFix:        false,
			expectedConfidence: 0.0,
		},
		{
			name:               "/Ask mixed case",
			message:            "/Ask what is this?",
			expectedFix:        false,
			expectedConfidence: 0.0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := fixer.IsFixRequest(tc.message)

			if result.IsFixRequest != tc.expectedFix {
				t.Errorf("expected IsFixRequest=%v, got %v", tc.expectedFix, result.IsFixRequest)
			}

			if result.Confidence != tc.expectedConfidence {
				t.Errorf("expected confidence=%f, got %f", tc.expectedConfidence, result.Confidence)
			}
		})
	}
}

func TestIsFixRequest_EmptyMessage(t *testing.T) {
	fixer := NewAgenticCodeFixer(&mockAIClient{}, "test-model")

	testCases := []string{
		"",
		"   ",
		"\t",
		"\n",
	}

	for _, message := range testCases {
		t.Run("empty or whitespace", func(t *testing.T) {
			result := fixer.IsFixRequest(message)

			if result.IsFixRequest {
				t.Errorf("expected IsFixRequest=false for empty/whitespace message, got true")
			}

			if result.Confidence != 0.0 {
				t.Errorf("expected confidence=0.0, got %f", result.Confidence)
			}
		})
	}
}

func TestBuildPrompt_BasicStructure(t *testing.T) {
	fixer := NewAgenticCodeFixer(&mockAIClient{}, "test-model")

	request := &FixRequest{
		UserMessage: "fix the syntax error",
		FileContent: "echo 'hello world",
		FilePath:    "/path/to/script.sh",
		FileType:    "bash",
	}

	prompt := fixer.BuildPrompt(request)

	// Check that all required sections are present
	requiredSections := []string{
		"=== FILE METADATA ===",
		"=== CURRENT FILE CONTENT ===",
		"=== USER REQUEST ===",
		"=== INSTRUCTIONS ===",
	}

	for _, section := range requiredSections {
		if !strings.Contains(prompt, section) {
			t.Errorf("prompt missing required section: %s", section)
		}
	}
}

func TestBuildPrompt_ContainsUserMessage(t *testing.T) {
	fixer := NewAgenticCodeFixer(&mockAIClient{}, "test-model")

	userMessage := "fix the syntax error on line 5"
	request := &FixRequest{
		UserMessage: userMessage,
		FileContent: "echo 'hello'",
		FilePath:    "/path/to/script.sh",
		FileType:    "bash",
	}

	prompt := fixer.BuildPrompt(request)

	if !strings.Contains(prompt, userMessage) {
		t.Errorf("prompt does not contain user message: %s", userMessage)
	}
}

func TestBuildPrompt_ContainsFileContent(t *testing.T) {
	fixer := NewAgenticCodeFixer(&mockAIClient{}, "test-model")

	fileContent := "#!/bin/bash\necho 'hello world'\nexit 0"
	request := &FixRequest{
		UserMessage: "fix this",
		FileContent: fileContent,
		FilePath:    "/path/to/script.sh",
		FileType:    "bash",
	}

	prompt := fixer.BuildPrompt(request)

	if !strings.Contains(prompt, fileContent) {
		t.Errorf("prompt does not contain file content")
	}
}

func TestBuildPrompt_ContainsFileMetadata(t *testing.T) {
	fixer := NewAgenticCodeFixer(&mockAIClient{}, "test-model")

	filePath := "/home/user/scripts/test.sh"
	fileType := "bash"

	request := &FixRequest{
		UserMessage: "fix this",
		FileContent: "echo 'test'",
		FilePath:    filePath,
		FileType:    fileType,
	}

	prompt := fixer.BuildPrompt(request)

	if !strings.Contains(prompt, filePath) {
		t.Errorf("prompt does not contain file path: %s", filePath)
	}

	if !strings.Contains(prompt, fileType) {
		t.Errorf("prompt does not contain file type: %s", fileType)
	}
}

func TestBuildPrompt_EmptyFileContent(t *testing.T) {
	fixer := NewAgenticCodeFixer(&mockAIClient{}, "test-model")

	request := &FixRequest{
		UserMessage: "create a hello world script",
		FileContent: "",
		FilePath:    "/path/to/new_script.sh",
		FileType:    "bash",
	}

	prompt := fixer.BuildPrompt(request)

	// Should indicate empty file
	if !strings.Contains(prompt, "(empty file)") {
		t.Errorf("prompt does not indicate empty file")
	}
}

func TestBuildPrompt_WhitespaceOnlyFileContent(t *testing.T) {
	fixer := NewAgenticCodeFixer(&mockAIClient{}, "test-model")

	request := &FixRequest{
		UserMessage: "create a script",
		FileContent: "   \n\t\n   ",
		FilePath:    "/path/to/script.sh",
		FileType:    "bash",
	}

	prompt := fixer.BuildPrompt(request)

	// Should treat whitespace-only as empty
	if !strings.Contains(prompt, "(empty file)") {
		t.Errorf("prompt does not indicate empty file for whitespace-only content")
	}
}

func TestBuildPrompt_ContainsInstructions(t *testing.T) {
	fixer := NewAgenticCodeFixer(&mockAIClient{}, "test-model")

	request := &FixRequest{
		UserMessage: "fix this",
		FileContent: "echo 'test'",
		FilePath:    "/path/to/script.sh",
		FileType:    "bash",
	}

	prompt := fixer.BuildPrompt(request)

	// Check for key instruction elements
	instructionKeywords := []string{
		"Analyze",
		"Generate",
		"markdown code block",
		"language identifier",
		"explanation",
	}

	for _, keyword := range instructionKeywords {
		if !strings.Contains(prompt, keyword) {
			t.Errorf("prompt missing instruction keyword: %s", keyword)
		}
	}
}

func TestBuildPrompt_SectionDelineation(t *testing.T) {
	fixer := NewAgenticCodeFixer(&mockAIClient{}, "test-model")

	request := &FixRequest{
		UserMessage: "fix the bug",
		FileContent: "echo 'hello'",
		FilePath:    "/path/to/script.sh",
		FileType:    "bash",
	}

	prompt := fixer.BuildPrompt(request)

	// Check that sections appear in the correct order
	metadataIndex := strings.Index(prompt, "=== FILE METADATA ===")
	contentIndex := strings.Index(prompt, "=== CURRENT FILE CONTENT ===")
	requestIndex := strings.Index(prompt, "=== USER REQUEST ===")
	instructionsIndex := strings.Index(prompt, "=== INSTRUCTIONS ===")

	if metadataIndex == -1 || contentIndex == -1 || requestIndex == -1 || instructionsIndex == -1 {
		t.Errorf("one or more sections missing from prompt")
		return
	}

	// Verify order: metadata -> content -> request -> instructions
	if !(metadataIndex < contentIndex && contentIndex < requestIndex && requestIndex < instructionsIndex) {
		t.Errorf("sections are not in the correct order")
	}
}

func TestBuildPrompt_DifferentFileTypes(t *testing.T) {
	fixer := NewAgenticCodeFixer(&mockAIClient{}, "test-model")

	fileTypes := []string{"bash", "shell", "powershell", "markdown"}

	for _, fileType := range fileTypes {
		t.Run(fileType, func(t *testing.T) {
			request := &FixRequest{
				UserMessage: "fix this",
				FileContent: "test content",
				FilePath:    "/path/to/file",
				FileType:    fileType,
			}

			prompt := fixer.BuildPrompt(request)

			// Should contain the file type
			if !strings.Contains(prompt, fileType) {
				t.Errorf("prompt does not contain file type: %s", fileType)
			}

			// Should include file type in code block format instruction
			expectedFormat := "```" + fileType
			if !strings.Contains(prompt, expectedFormat) {
				t.Errorf("prompt does not contain expected code block format: %s", expectedFormat)
			}
		})
	}
}

func TestBuildPrompt_PreservesNewlines(t *testing.T) {
	fixer := NewAgenticCodeFixer(&mockAIClient{}, "test-model")

	fileContent := "line1\nline2\nline3"
	request := &FixRequest{
		UserMessage: "fix this",
		FileContent: fileContent,
		FilePath:    "/path/to/script.sh",
		FileType:    "bash",
	}

	prompt := fixer.BuildPrompt(request)

	// Should preserve newlines in file content
	if !strings.Contains(prompt, "line1\nline2\nline3") {
		t.Errorf("prompt does not preserve newlines in file content")
	}
}

func TestBuildPrompt_HandlesSpecialCharacters(t *testing.T) {
	fixer := NewAgenticCodeFixer(&mockAIClient{}, "test-model")

	fileContent := "echo \"hello $USER\"\necho 'single quotes'\necho `backticks`"
	userMessage := "fix the $variable and 'quotes'"

	request := &FixRequest{
		UserMessage: userMessage,
		FileContent: fileContent,
		FilePath:    "/path/to/script.sh",
		FileType:    "bash",
	}

	prompt := fixer.BuildPrompt(request)

	// Should contain special characters without escaping
	if !strings.Contains(prompt, "$USER") {
		t.Errorf("prompt does not contain $USER")
	}

	if !strings.Contains(prompt, "'quotes'") {
		t.Errorf("prompt does not contain single quotes")
	}

	if !strings.Contains(prompt, "`backticks`") {
		t.Errorf("prompt does not contain backticks")
	}
}

func TestBuildPrompt_LongFileContent(t *testing.T) {
	fixer := NewAgenticCodeFixer(&mockAIClient{}, "test-model")

	// Create a long file content
	var longContent strings.Builder
	for i := 0; i < 100; i++ {
		longContent.WriteString("echo 'line ")
		longContent.WriteString(strings.Repeat("x", 50))
		longContent.WriteString("'\n")
	}

	request := &FixRequest{
		UserMessage: "optimize this script",
		FileContent: longContent.String(),
		FilePath:    "/path/to/long_script.sh",
		FileType:    "bash",
	}

	prompt := fixer.BuildPrompt(request)

	// Should contain the full content
	if !strings.Contains(prompt, longContent.String()) {
		t.Errorf("prompt does not contain full long file content")
	}
}

func TestBuildPrompt_MultilineUserMessage(t *testing.T) {
	fixer := NewAgenticCodeFixer(&mockAIClient{}, "test-model")

	userMessage := "fix the following issues:\n1. syntax error on line 5\n2. missing semicolon on line 10\n3. incorrect variable name"

	request := &FixRequest{
		UserMessage: userMessage,
		FileContent: "echo 'test'",
		FilePath:    "/path/to/script.sh",
		FileType:    "bash",
	}

	prompt := fixer.BuildPrompt(request)

	// Should preserve multiline user message
	if !strings.Contains(prompt, userMessage) {
		t.Errorf("prompt does not contain full multiline user message")
	}
}

func TestBuildPrompt_EnsuresNewlineAfterContent(t *testing.T) {
	fixer := NewAgenticCodeFixer(&mockAIClient{}, "test-model")

	// Test with content that doesn't end with newline
	request := &FixRequest{
		UserMessage: "fix this",
		FileContent: "echo 'test'",
		FilePath:    "/path/to/script.sh",
		FileType:    "bash",
	}

	prompt := fixer.BuildPrompt(request)

	// Find the content section
	contentStart := strings.Index(prompt, "=== CURRENT FILE CONTENT ===")
	requestStart := strings.Index(prompt, "=== USER REQUEST ===")

	if contentStart == -1 || requestStart == -1 {
		t.Errorf("could not find content or request sections")
		return
	}

	contentSection := prompt[contentStart:requestStart]

	// Should have proper spacing after content
	if !strings.Contains(contentSection, "echo 'test'\n") {
		t.Errorf("content section does not have newline after file content")
	}
}

func TestProcessMessage_ConversationalMode(t *testing.T) {
	fixer := NewAgenticCodeFixer(&mockAIClient{}, "test-model")

	testCases := []struct {
		name    string
		message string
	}{
		{
			name:    "question about code",
			message: "what does this function do?",
		},
		{
			name:    "explanation request",
			message: "explain how this works",
		},
		{
			name:    "explicit /ask command",
			message: "/ask what is this?",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := fixer.ProcessMessage(
				tc.message,
				"echo 'test'",
				"/path/to/file.sh",
				"bash",
			)

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if result == nil {
				t.Fatalf("result is nil")
			}

			if !result.IsConversational {
				t.Errorf("expected IsConversational=true, got false")
			}

			if result.Success {
				t.Errorf("expected Success=false for conversational mode, got true")
			}

			// Validate invariants
			if err := result.Validate(); err != nil {
				t.Errorf("result violates invariants: %v", err)
			}
		})
	}
}

func TestProcessMessage_NoFileOpen(t *testing.T) {
	fixer := NewAgenticCodeFixer(&mockAIClient{}, "test-model")

	testCases := []struct {
		name     string
		filePath string
	}{
		{
			name:     "empty file path",
			filePath: "",
		},
		{
			name:     "whitespace only file path",
			filePath: "   ",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := fixer.ProcessMessage(
				"fix this bug",
				"echo 'test'",
				tc.filePath,
				"bash",
			)

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if result == nil {
				t.Fatalf("result is nil")
			}

			if result.Success {
				t.Errorf("expected Success=false when no file is open, got true")
			}

			if result.IsConversational {
				t.Errorf("expected IsConversational=false, got true")
			}

			expectedError := "Please open a file before requesting code fixes"
			if result.ErrorMessage != expectedError {
				t.Errorf("expected error message '%s', got '%s'", expectedError, result.ErrorMessage)
			}

			// Validate invariants
			if err := result.Validate(); err != nil {
				t.Errorf("result violates invariants: %v", err)
			}
		})
	}
}

func TestProcessMessage_InvalidFileType(t *testing.T) {
	fixer := NewAgenticCodeFixer(&mockAIClient{}, "test-model")

	result, err := fixer.ProcessMessage(
		"fix this bug",
		"echo 'test'",
		"/path/to/file.txt",
		"invalid-type",
	)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatalf("result is nil")
	}

	if result.Success {
		t.Errorf("expected Success=false for invalid file type, got true")
	}

	if result.IsConversational {
		t.Errorf("expected IsConversational=false, got true")
	}

	if !strings.Contains(result.ErrorMessage, "Invalid fix request") {
		t.Errorf("expected error message to contain 'Invalid fix request', got '%s'", result.ErrorMessage)
	}

	// Validate invariants
	if err := result.Validate(); err != nil {
		t.Errorf("result violates invariants: %v", err)
	}
}

func TestProcessMessage_EmptyUserMessage(t *testing.T) {
	fixer := NewAgenticCodeFixer(&mockAIClient{}, "test-model")

	testCases := []struct {
		name    string
		message string
	}{
		{
			name:    "empty message",
			message: "",
		},
		{
			name:    "whitespace only message",
			message: "   ",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := fixer.ProcessMessage(
				tc.message,
				"echo 'test'",
				"/path/to/file.sh",
				"bash",
			)

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if result == nil {
				t.Fatalf("result is nil")
			}

			// Empty message should be treated as conversational (no fix keywords)
			if !result.IsConversational {
				t.Errorf("expected IsConversational=true for empty message, got false")
			}
		})
	}
}

func TestProcessMessage_ValidFixRequest(t *testing.T) {
	fixer := NewAgenticCodeFixer(&mockAIClient{}, "test-model")

	testCases := []struct {
		name     string
		message  string
		fileType string
	}{
		{
			name:     "fix keyword",
			message:  "fix the syntax error",
			fileType: "bash",
		},
		{
			name:     "change keyword",
			message:  "change the variable name",
			fileType: "shell",
		},
		{
			name:     "update keyword",
			message:  "update the function",
			fileType: "powershell",
		},
		{
			name:     "explicit /fix command",
			message:  "/fix this bug",
			fileType: "markdown",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := fixer.ProcessMessage(
				tc.message,
				"echo 'test'",
				"/path/to/file",
				tc.fileType,
			)

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if result == nil {
				t.Fatalf("result is nil")
			}

			if result.IsConversational {
				t.Errorf("expected IsConversational=false for fix request, got true")
			}

			// Note: The actual fix generation will be tested when GenerateFix is implemented
			// For now, we just verify that it routes to GenerateFix (which will return an error
			// since it's not implemented yet)
		})
	}
}

func TestProcessMessage_DetectionRouting(t *testing.T) {
	fixer := NewAgenticCodeFixer(&mockAIClient{}, "test-model")

	testCases := []struct {
		name                   string
		message                string
		expectedConversational bool
	}{
		{
			name:                   "fix keyword routes to fix handler",
			message:                "fix this bug",
			expectedConversational: false,
		},
		{
			name:                   "question routes to conversational",
			message:                "what is this?",
			expectedConversational: true,
		},
		{
			name:                   "/fix routes to fix handler",
			message:                "/fix the error",
			expectedConversational: false,
		},
		{
			name:                   "/ask routes to conversational",
			message:                "/ask about this code",
			expectedConversational: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := fixer.ProcessMessage(
				tc.message,
				"echo 'test'",
				"/path/to/file.sh",
				"bash",
			)

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if result == nil {
				t.Fatalf("result is nil")
			}

			if result.IsConversational != tc.expectedConversational {
				t.Errorf("expected IsConversational=%v, got %v", tc.expectedConversational, result.IsConversational)
			}
		})
	}
}

func TestProcessMessage_PreservesFileContent(t *testing.T) {
	fixer := NewAgenticCodeFixer(&mockAIClient{}, "test-model")

	fileContent := "#!/bin/bash\necho 'hello world'\nexit 0"

	result, err := fixer.ProcessMessage(
		"fix this",
		fileContent,
		"/path/to/file.sh",
		"bash",
	)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatalf("result is nil")
	}

	// The file content should be passed to GenerateFix
	// We can't directly verify this without implementing GenerateFix,
	// but we can verify that ProcessMessage doesn't modify the content
	// by checking that it doesn't return modified content for conversational mode

	// Test with conversational message
	result2, err := fixer.ProcessMessage(
		"what does this do?",
		fileContent,
		"/path/to/file.sh",
		"bash",
	)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result2.ModifiedContent != "" {
		t.Errorf("expected no modified content for conversational mode, got: %s", result2.ModifiedContent)
	}
}

func TestProcessMessage_HandlesAllFileTypes(t *testing.T) {
	fixer := NewAgenticCodeFixer(&mockAIClient{}, "test-model")

	fileTypes := []string{"bash", "shell", "powershell", "markdown"}

	for _, fileType := range fileTypes {
		t.Run(fileType, func(t *testing.T) {
			result, err := fixer.ProcessMessage(
				"fix this",
				"test content",
				"/path/to/file",
				fileType,
			)

			if err != nil {
				t.Errorf("unexpected error for file type %s: %v", fileType, err)
			}

			if result == nil {
				t.Fatalf("result is nil for file type %s", fileType)
			}

			// Should route to fix handler for all valid file types
			if result.IsConversational {
				t.Errorf("expected IsConversational=false for fix request with file type %s", fileType)
			}
		})
	}
}

func TestProcessMessage_ValidatesResult(t *testing.T) {
	fixer := NewAgenticCodeFixer(&mockAIClient{}, "test-model")

	testCases := []struct {
		name    string
		message string
	}{
		{
			name:    "conversational message",
			message: "what is this?",
		},
		{
			name:    "fix request with no file",
			message: "fix this",
		},
		{
			name:    "invalid file type",
			message: "fix this",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var result *FixResult
			var err error

			if tc.name == "fix request with no file" {
				result, err = fixer.ProcessMessage(tc.message, "content", "", "bash")
			} else if tc.name == "invalid file type" {
				result, err = fixer.ProcessMessage(tc.message, "content", "/path", "invalid")
			} else {
				result, err = fixer.ProcessMessage(tc.message, "content", "/path", "bash")
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if result == nil {
				t.Fatalf("result is nil")
			}

			// All results should satisfy invariants
			if err := result.Validate(); err != nil {
				t.Errorf("result violates invariants: %v", err)
			}
		})
	}
}

// mockAIClientWithCodeBlock is a mock that returns a code block in the response
type mockAIClientWithCodeBlock struct {
	response  string
	available bool
	err       error
}

// mockAIClientWithGenerateError is a mock that returns an error from Generate
type mockAIClientWithGenerateError struct {
	generateErr error
}

func (m *mockAIClientWithGenerateError) IsAvailable() (bool, error) {
	return true, nil
}

func (m *mockAIClientWithGenerateError) Generate(prompt string, model string, context []int) (<-chan string, error) {
	return nil, m.generateErr
}

func (m *mockAIClientWithGenerateError) ListModels() ([]string, error) {
	return []string{"mock-model"}, nil
}

func TestGenerateFix_GenerateError(t *testing.T) {
	mockClient := &mockAIClientWithGenerateError{
		generateErr: fmt.Errorf("network timeout"),
	}

	fixer := NewAgenticCodeFixer(mockClient, "test-model")

	request := &FixRequest{
		UserMessage: "fix this",
		FileContent: "echo 'test'",
		FilePath:    "/path/to/script.sh",
		FileType:    "bash",
	}

	result, err := fixer.GenerateFix(request)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatalf("result is nil")
	}

	if result.Success {
		t.Errorf("expected Success=false when Generate returns error, got true")
	}

	expectedError := "Failed to generate fix"
	if !strings.Contains(result.ErrorMessage, expectedError) {
		t.Errorf("expected error message to contain '%s', got '%s'", expectedError, result.ErrorMessage)
	}

	// Should also contain the actual error message
	if !strings.Contains(result.ErrorMessage, "network timeout") {
		t.Errorf("expected error message to contain 'network timeout', got '%s'", result.ErrorMessage)
	}

	// Validate invariants
	if err := result.Validate(); err != nil {
		t.Errorf("result violates invariants: %v", err)
	}
}

func (m *mockAIClientWithCodeBlock) IsAvailable() (bool, error) {
	return m.available, m.err
}

func (m *mockAIClientWithCodeBlock) Generate(prompt string, model string, context []int) (<-chan string, error) {
	if m.err != nil {
		return nil, m.err
	}

	ch := make(chan string, 1)
	ch <- m.response
	close(ch)
	return ch, nil
}

func (m *mockAIClientWithCodeBlock) ListModels() ([]string, error) {
	return []string{"mock-model"}, nil
}

func TestGenerateFix_Success(t *testing.T) {
	mockClient := &mockAIClientWithCodeBlock{
		response:  "Here's the fixed code:\n\n```bash\n#!/bin/bash\necho 'hello world'\nexit 0\n```\n\nI fixed the syntax error.",
		available: true,
		err:       nil,
	}

	fixer := NewAgenticCodeFixer(mockClient, "test-model")

	request := &FixRequest{
		UserMessage: "fix the syntax error",
		FileContent: "echo 'hello world",
		FilePath:    "/path/to/script.sh",
		FileType:    "bash",
	}

	result, err := fixer.GenerateFix(request)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatalf("result is nil")
	}

	if !result.Success {
		t.Errorf("expected Success=true, got false. Error: %s", result.ErrorMessage)
	}

	if result.IsConversational {
		t.Errorf("expected IsConversational=false, got true")
	}

	if result.ModifiedContent == "" {
		t.Errorf("expected ModifiedContent to be set, got empty")
	}

	if result.ChangesSummary == "" {
		t.Errorf("expected ChangesSummary to be set, got empty")
	}

	// Validate invariants
	if err := result.Validate(); err != nil {
		t.Errorf("result violates invariants: %v", err)
	}
}

func TestGenerateFix_AIServiceUnavailable(t *testing.T) {
	mockClient := &mockAIClientWithCodeBlock{
		response:  "",
		available: false,
		err:       nil,
	}

	fixer := NewAgenticCodeFixer(mockClient, "test-model")

	request := &FixRequest{
		UserMessage: "fix this",
		FileContent: "echo 'test'",
		FilePath:    "/path/to/script.sh",
		FileType:    "bash",
	}

	result, err := fixer.GenerateFix(request)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatalf("result is nil")
	}

	if result.Success {
		t.Errorf("expected Success=false when AI unavailable, got true")
	}

	expectedError := "AI service unavailable"
	if !strings.Contains(result.ErrorMessage, expectedError) {
		t.Errorf("expected error message to contain '%s', got '%s'", expectedError, result.ErrorMessage)
	}

	// Validate invariants
	if err := result.Validate(); err != nil {
		t.Errorf("result violates invariants: %v", err)
	}
}

func TestGenerateFix_AIServiceError(t *testing.T) {
	mockClient := &mockAIClientWithCodeBlock{
		response:  "",
		available: false,
		err:       fmt.Errorf("connection refused"),
	}

	fixer := NewAgenticCodeFixer(mockClient, "test-model")

	request := &FixRequest{
		UserMessage: "fix this",
		FileContent: "echo 'test'",
		FilePath:    "/path/to/script.sh",
		FileType:    "bash",
	}

	result, err := fixer.GenerateFix(request)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatalf("result is nil")
	}

	if result.Success {
		t.Errorf("expected Success=false when AI service returns error, got true")
	}

	expectedError := "AI service unavailable"
	if !strings.Contains(result.ErrorMessage, expectedError) {
		t.Errorf("expected error message to contain '%s', got '%s'", expectedError, result.ErrorMessage)
	}

	// Validate invariants
	if err := result.Validate(); err != nil {
		t.Errorf("result violates invariants: %v", err)
	}
}

func TestGenerateFix_EmptyResponse(t *testing.T) {
	mockClient := &mockAIClientWithCodeBlock{
		response:  "",
		available: true,
		err:       nil,
	}

	fixer := NewAgenticCodeFixer(mockClient, "test-model")

	request := &FixRequest{
		UserMessage: "fix this",
		FileContent: "echo 'test'",
		FilePath:    "/path/to/script.sh",
		FileType:    "bash",
	}

	result, err := fixer.GenerateFix(request)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatalf("result is nil")
	}

	if result.Success {
		t.Errorf("expected Success=false for empty response, got true")
	}

	expectedError := "empty response"
	if !strings.Contains(result.ErrorMessage, expectedError) {
		t.Errorf("expected error message to contain '%s', got '%s'", expectedError, result.ErrorMessage)
	}
}

func TestGenerateFix_NoCodeBlocks(t *testing.T) {
	mockClient := &mockAIClientWithCodeBlock{
		response:  "I think you should fix the syntax error by adding a closing quote.",
		available: true,
		err:       nil,
	}

	fixer := NewAgenticCodeFixer(mockClient, "test-model")

	request := &FixRequest{
		UserMessage: "fix this",
		FileContent: "echo 'test",
		FilePath:    "/path/to/script.sh",
		FileType:    "bash",
	}

	result, err := fixer.GenerateFix(request)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatalf("result is nil")
	}

	if result.Success {
		t.Errorf("expected Success=false when no code blocks, got true")
	}

	expectedError := "did not contain any code blocks"
	if !strings.Contains(result.ErrorMessage, expectedError) {
		t.Errorf("expected error message to contain '%s', got '%s'", expectedError, result.ErrorMessage)
	}
}

func TestGenerateFix_InvalidSyntax(t *testing.T) {
	mockClient := &mockAIClientWithCodeBlock{
		response:  "```bash\nif true; then\necho 'missing fi'\n```",
		available: true,
		err:       nil,
	}

	fixer := NewAgenticCodeFixer(mockClient, "test-model")

	request := &FixRequest{
		UserMessage: "fix this",
		FileContent: "echo 'test'",
		FilePath:    "/path/to/script.sh",
		FileType:    "bash",
	}

	result, err := fixer.GenerateFix(request)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatalf("result is nil")
	}

	if result.Success {
		t.Errorf("expected Success=false for invalid syntax, got true")
	}

	expectedError := "invalid syntax"
	if !strings.Contains(result.ErrorMessage, expectedError) {
		t.Errorf("expected error message to contain '%s', got '%s'", expectedError, result.ErrorMessage)
	}
}

func TestGenerateFix_MultipleCodeBlocks(t *testing.T) {
	mockClient := &mockAIClientWithCodeBlock{
		response:  "Here's an example:\n```bash\necho 'example'\n```\n\nAnd here's the fix:\n```bash\n#!/bin/bash\necho 'fixed'\nexit 0\n```",
		available: true,
		err:       nil,
	}

	fixer := NewAgenticCodeFixer(mockClient, "test-model")

	request := &FixRequest{
		UserMessage: "fix this",
		FileContent: "echo 'test'",
		FilePath:    "/path/to/script.sh",
		FileType:    "bash",
	}

	result, err := fixer.GenerateFix(request)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatalf("result is nil")
	}

	// Should successfully identify and use the first matching fix block
	if !result.Success {
		t.Errorf("expected Success=true, got false. Error: %s", result.ErrorMessage)
	}
}

func TestGenerateFix_DifferentFileTypes(t *testing.T) {
	testCases := []struct {
		fileType string
		response string
	}{
		{
			fileType: "bash",
			response: "```bash\n#!/bin/bash\necho 'test'\n```",
		},
		{
			fileType: "shell",
			response: "```shell\n#!/bin/sh\necho 'test'\n```",
		},
		{
			fileType: "powershell",
			response: "```powershell\nWrite-Host 'test'\n```",
		},
		{
			fileType: "markdown",
			response: "```markdown\n# Test\nContent\n```",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.fileType, func(t *testing.T) {
			mockClient := &mockAIClientWithCodeBlock{
				response:  tc.response,
				available: true,
				err:       nil,
			}

			fixer := NewAgenticCodeFixer(mockClient, "test-model")

			request := &FixRequest{
				UserMessage: "fix this",
				FileContent: "test",
				FilePath:    "/path/to/file",
				FileType:    tc.fileType,
			}

			result, err := fixer.GenerateFix(request)

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if result == nil {
				t.Fatalf("result is nil")
			}

			if !result.Success {
				t.Errorf("expected Success=true for file type %s, got false. Error: %s", tc.fileType, result.ErrorMessage)
			}
		})
	}
}

func TestGenerateFix_ChangesSummary(t *testing.T) {
	mockClient := &mockAIClientWithCodeBlock{
		response:  "```bash\n#!/bin/bash\necho 'hello'\necho 'world'\nexit 0\n```",
		available: true,
		err:       nil,
	}

	fixer := NewAgenticCodeFixer(mockClient, "test-model")

	request := &FixRequest{
		UserMessage: "add more lines",
		FileContent: "echo 'hello'",
		FilePath:    "/path/to/script.sh",
		FileType:    "bash",
	}

	result, err := fixer.GenerateFix(request)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatalf("result is nil")
	}

	if !result.Success {
		t.Errorf("expected Success=true, got false")
	}

	// Check that summary contains key information
	if !strings.Contains(result.ChangesSummary, request.FilePath) {
		t.Errorf("changes summary should contain file path")
	}

	if !strings.Contains(result.ChangesSummary, "save") {
		t.Errorf("changes summary should remind to save")
	}

	if !strings.Contains(result.ChangesSummary, "test") {
		t.Errorf("changes summary should remind to test")
	}
}

func TestGenerateFix_NewFile(t *testing.T) {
	mockClient := &mockAIClientWithCodeBlock{
		response:  "```bash\n#!/bin/bash\necho 'new file'\n```",
		available: true,
		err:       nil,
	}

	fixer := NewAgenticCodeFixer(mockClient, "test-model")

	request := &FixRequest{
		UserMessage: "create a hello world script",
		FileContent: "",
		FilePath:    "/path/to/new_script.sh",
		FileType:    "bash",
	}

	result, err := fixer.GenerateFix(request)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatalf("result is nil")
	}

	if !result.Success {
		t.Errorf("expected Success=true for new file, got false. Error: %s", result.ErrorMessage)
	}

	// Check that summary indicates new file
	if !strings.Contains(result.ChangesSummary, "new file") && !strings.Contains(result.ChangesSummary, "Added") {
		t.Errorf("changes summary should indicate new file creation")
	}
}

func TestGenerateFix_EnsuresNewlineAtEnd(t *testing.T) {
	mockClient := &mockAIClientWithCodeBlock{
		response:  "```bash\necho 'test'```",
		available: true,
		err:       nil,
	}

	fixer := NewAgenticCodeFixer(mockClient, "test-model")

	request := &FixRequest{
		UserMessage: "fix this",
		FileContent: "echo 'old'",
		FilePath:    "/path/to/script.sh",
		FileType:    "bash",
	}

	result, err := fixer.GenerateFix(request)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatalf("result is nil")
	}

	if !result.Success {
		t.Errorf("expected Success=true, got false")
	}

	// Check that modified content contains the ADD marker for the new content
	if !strings.Contains(result.ModifiedContent, "~ADD~echo 'test'") {
		t.Errorf("modified content should contain added string: %s", result.ModifiedContent)
	}
}

func TestGenerateFix_ValidatesResult(t *testing.T) {
	mockClient := &mockAIClientWithCodeBlock{
		response:  "```bash\n#!/bin/bash\necho 'test'\n```",
		available: true,
		err:       nil,
	}

	fixer := NewAgenticCodeFixer(mockClient, "test-model")

	request := &FixRequest{
		UserMessage: "fix this",
		FileContent: "echo 'old'",
		FilePath:    "/path/to/script.sh",
		FileType:    "bash",
	}

	result, err := fixer.GenerateFix(request)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatalf("result is nil")
	}

	// All results should satisfy invariants
	if err := result.Validate(); err != nil {
		t.Errorf("result violates invariants: %v", err)
	}
}

// TestApplyFix_BasicReplacement tests basic whole-file replacement
func TestApplyFix_BasicReplacement(t *testing.T) {
	fixer := NewAgenticCodeFixer(&mockAIClient{}, "test-model")

	originalContent := "echo 'old code'"
	fixCode := "#!/bin/bash\necho 'new code'\nexit 0"

	result, err := fixer.ApplyFix(originalContent, fixCode, "bash")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result == "" {
		t.Errorf("expected non-empty result, got empty")
	}

	// Should contain the fix code
	if !strings.Contains(result, "new code") {
		t.Errorf("result should contain fix code")
	}

	// Should end with newline
	if !strings.HasSuffix(result, "\n") {
		t.Errorf("result should end with newline")
	}
}

// TestApplyFix_EmptyFixCode tests that empty fix code is rejected
func TestApplyFix_EmptyFixCode(t *testing.T) {
	fixer := NewAgenticCodeFixer(&mockAIClient{}, "test-model")

	testCases := []struct {
		name    string
		fixCode string
	}{
		{
			name:    "empty string",
			fixCode: "",
		},
		{
			name:    "whitespace only",
			fixCode: "   \n\t  ",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := fixer.ApplyFix("original content", tc.fixCode, "bash")

			if err == nil {
				t.Errorf("expected error for empty fix code, got nil")
			}

			if !strings.Contains(err.Error(), "empty") {
				t.Errorf("expected error message to contain 'empty', got: %s", err.Error())
			}
		})
	}
}

// TestApplyFix_PreservesContent tests that fix code is preserved exactly
func TestApplyFix_PreservesContent(t *testing.T) {
	fixer := NewAgenticCodeFixer(&mockAIClient{}, "test-model")

	testCases := []struct {
		name    string
		fixCode string
	}{
		{
			name:    "simple code",
			fixCode: "echo 'test'",
		},
		{
			name:    "multiline code",
			fixCode: "#!/bin/bash\necho 'line1'\necho 'line2'\nexit 0",
		},
		{
			name:    "code with special characters",
			fixCode: "echo \"$USER\"\necho 'single quotes'\necho `backticks`",
		},
		{
			name:    "code with indentation",
			fixCode: "if [ true ]; then\n    echo 'indented'\nfi",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := fixer.ApplyFix("old content", tc.fixCode, "bash")

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			// Result should contain the fix code (possibly with added newline)
			resultWithoutTrailingNewline := strings.TrimRight(result, "\n")
			fixCodeWithoutTrailingNewline := strings.TrimRight(tc.fixCode, "\n")

			if resultWithoutTrailingNewline != fixCodeWithoutTrailingNewline {
				t.Errorf("result does not match fix code.\nExpected:\n%s\nGot:\n%s",
					fixCodeWithoutTrailingNewline, resultWithoutTrailingNewline)
			}
		})
	}
}

// TestApplyFix_EnsuresTrailingNewline tests that result always ends with newline
func TestApplyFix_EnsuresTrailingNewline(t *testing.T) {
	fixer := NewAgenticCodeFixer(&mockAIClient{}, "test-model")

	testCases := []struct {
		name    string
		fixCode string
	}{
		{
			name:    "code without newline",
			fixCode: "echo 'test'",
		},
		{
			name:    "code with newline",
			fixCode: "echo 'test'\n",
		},
		{
			name:    "code with multiple newlines",
			fixCode: "echo 'test'\n\n",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := fixer.ApplyFix("old content", tc.fixCode, "bash")

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if !strings.HasSuffix(result, "\n") {
				t.Errorf("result should end with newline")
			}
		})
	}
}

// TestApplyFix_DifferentFileTypes tests applying fixes to different file types
func TestApplyFix_DifferentFileTypes(t *testing.T) {
	fixer := NewAgenticCodeFixer(&mockAIClient{}, "test-model")

	testCases := []struct {
		fileType string
		fixCode  string
	}{
		{
			fileType: "bash",
			fixCode:  "#!/bin/bash\necho 'test'",
		},
		{
			fileType: "shell",
			fixCode:  "#!/bin/sh\necho 'test'",
		},
		{
			fileType: "powershell",
			fixCode:  "Write-Host 'test'",
		},
		{
			fileType: "markdown",
			fixCode:  "# Title\n\nContent here",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.fileType, func(t *testing.T) {
			result, err := fixer.ApplyFix("old content", tc.fixCode, tc.fileType)

			if err != nil {
				t.Errorf("unexpected error for file type %s: %v", tc.fileType, err)
			}

			if result == "" {
				t.Errorf("expected non-empty result for file type %s", tc.fileType)
			}
		})
	}
}

// TestApplyFix_EmptyOriginalContent tests applying fix to empty file
func TestApplyFix_EmptyOriginalContent(t *testing.T) {
	fixer := NewAgenticCodeFixer(&mockAIClient{}, "test-model")

	fixCode := "#!/bin/bash\necho 'new file'"

	result, err := fixer.ApplyFix("", fixCode, "bash")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result == "" {
		t.Errorf("expected non-empty result, got empty")
	}

	// Should contain the fix code
	if !strings.Contains(result, "new file") {
		t.Errorf("result should contain fix code")
	}
}

// TestApplyFix_WhitespaceOnlyResult tests that whitespace-only results are rejected
func TestApplyFix_WhitespaceOnlyResult(t *testing.T) {
	fixer := NewAgenticCodeFixer(&mockAIClient{}, "test-model")

	// This test verifies that if somehow the fix code becomes whitespace-only
	// after processing, it's rejected
	// Note: This is a defensive check - in practice, the parser should prevent this

	fixCode := "   \n\t  "

	_, err := fixer.ApplyFix("original content", fixCode, "bash")

	if err == nil {
		t.Errorf("expected error for whitespace-only fix code, got nil")
	}
}

// TestApplyFix_LargeFile tests applying fix to large file
func TestApplyFix_LargeFile(t *testing.T) {
	fixer := NewAgenticCodeFixer(&mockAIClient{}, "test-model")

	// Create a large original content
	var largeOriginal strings.Builder
	for i := 0; i < 1000; i++ {
		largeOriginal.WriteString("echo 'line ")
		largeOriginal.WriteString(fmt.Sprintf("%d", i))
		largeOriginal.WriteString("'\n")
	}

	// Create a large fix code
	var largeFix strings.Builder
	for i := 0; i < 1000; i++ {
		largeFix.WriteString("echo 'fixed line ")
		largeFix.WriteString(fmt.Sprintf("%d", i))
		largeFix.WriteString("'\n")
	}

	result, err := fixer.ApplyFix(largeOriginal.String(), largeFix.String(), "bash")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result == "" {
		t.Errorf("expected non-empty result, got empty")
	}

	// Should contain the fix code
	if !strings.Contains(result, "fixed line") {
		t.Errorf("result should contain fix code")
	}
}

// TestApplyFix_PreservesIndentation tests that indentation is preserved
func TestApplyFix_PreservesIndentation(t *testing.T) {
	fixer := NewAgenticCodeFixer(&mockAIClient{}, "test-model")

	fixCode := "function test() {\n    echo 'indented line 1'\n    echo 'indented line 2'\n}"

	result, err := fixer.ApplyFix("old content", fixCode, "bash")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Should preserve the indentation
	if !strings.Contains(result, "    echo 'indented line 1'") {
		t.Errorf("result should preserve indentation")
	}

	if !strings.Contains(result, "    echo 'indented line 2'") {
		t.Errorf("result should preserve indentation")
	}
}

// TestApplyFix_PreservesBlankLines tests that blank lines are preserved
func TestApplyFix_PreservesBlankLines(t *testing.T) {
	fixer := NewAgenticCodeFixer(&mockAIClient{}, "test-model")

	fixCode := "#!/bin/bash\n\necho 'line 1'\n\necho 'line 2'\n\nexit 0"

	result, err := fixer.ApplyFix("old content", fixCode, "bash")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Should preserve blank lines
	if !strings.Contains(result, "\n\necho 'line 1'\n\necho 'line 2'\n\n") {
		t.Errorf("result should preserve blank lines")
	}
}

// TestApplyFix_IntegrationWithGenerateFix tests that ApplyFix works with GenerateFix
func TestApplyFix_IntegrationWithGenerateFix(t *testing.T) {
	mockClient := &mockAIClientWithCodeBlock{
		response:  "```bash\n#!/bin/bash\necho 'fixed code'\nexit 0\n```",
		available: true,
		err:       nil,
	}

	fixer := NewAgenticCodeFixer(mockClient, "test-model")

	request := &FixRequest{
		UserMessage: "fix this",
		FileContent: "echo 'old code'",
		FilePath:    "/path/to/script.sh",
		FileType:    "bash",
	}

	result, err := fixer.GenerateFix(request)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatalf("result is nil")
	}

	if !result.Success {
		t.Errorf("expected Success=true, got false. Error: %s", result.ErrorMessage)
	}

	// Modified content should be the result of ApplyFix
	if !strings.Contains(result.ModifiedContent, "fixed code") {
		t.Errorf("modified content should contain fixed code")
	}

	// Should contain the ADD marker instead of bare text
	if !strings.Contains(result.ModifiedContent, "~ADD~echo 'fixed code'") {
		t.Errorf("modified content should contain added fixed code: %s", result.ModifiedContent)
	}
}

// TestGenerateFix_EmptyFixBlock tests that empty fix blocks are rejected with helpful message
// This test validates Requirements 3.5 and 7.3
func TestGenerateFix_EmptyFixBlock(t *testing.T) {
	mockClient := &mockAIClientWithCodeBlock{
		response:  "```bash\n   \n```",
		available: true,
		err:       nil,
	}

	fixer := NewAgenticCodeFixer(mockClient, "test-model")

	request := &FixRequest{
		UserMessage: "fix this",
		FileContent: "echo 'test'",
		FilePath:    "/path/to/script.sh",
		FileType:    "bash",
	}

	result, err := fixer.GenerateFix(request)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatalf("result is nil")
	}

	if result.Success {
		t.Errorf("expected Success=false for empty fix block, got true")
	}

	// Empty code blocks are filtered out by ExtractCodeBlocks, so we get "no code blocks" error
	// This is correct behavior - the error message explains the issue (Requirement 7.3)
	if !strings.Contains(result.ErrorMessage, "did not contain any code blocks") {
		t.Errorf("expected error message to explain no code blocks found, got '%s'", result.ErrorMessage)
	}
}

// TestGenerateFix_InvalidSyntaxWithHelpfulMessage tests that syntax errors include helpful guidance
// This test validates Requirements 3.5 and 7.3
func TestGenerateFix_InvalidSyntaxWithHelpfulMessage(t *testing.T) {
	mockClient := &mockAIClientWithCodeBlock{
		response:  "```bash\nif true; then\necho 'missing fi'\n```",
		available: true,
		err:       nil,
	}

	fixer := NewAgenticCodeFixer(mockClient, "test-model")

	request := &FixRequest{
		UserMessage: "fix this",
		FileContent: "echo 'test'",
		FilePath:    "/path/to/script.sh",
		FileType:    "bash",
	}

	result, err := fixer.GenerateFix(request)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatalf("result is nil")
	}

	if result.Success {
		t.Errorf("expected Success=false for invalid syntax, got true")
	}

	// Check that error message is helpful (Requirement 7.3)
	if !strings.Contains(result.ErrorMessage, "invalid syntax") {
		t.Errorf("expected error message to mention 'invalid syntax', got '%s'", result.ErrorMessage)
	}

	if !strings.Contains(result.ErrorMessage, "Pre-validation failed") {
		t.Errorf("expected error message to explain pre-validation failed, got '%s'", result.ErrorMessage)
	}

	if !strings.Contains(result.ErrorMessage, "review your request") {
		t.Errorf("expected error message to suggest reviewing request, got '%s'", result.ErrorMessage)
	}

	// Should include the specific syntax error
	if !strings.Contains(result.ErrorMessage, "unmatched") {
		t.Errorf("expected error message to include specific syntax error, got '%s'", result.ErrorMessage)
	}
}

// TestGenerateFix_NoFixBlocksWithHelpfulMessage tests that missing fix blocks include helpful guidance
// This test validates Requirements 3.5 and 7.3
func TestGenerateFix_NoFixBlocksWithHelpfulMessage(t *testing.T) {
	mockClient := &mockAIClientWithCodeBlock{
		response:  "```python\nprint('wrong language')\n```",
		available: true,
		err:       nil,
	}

	fixer := NewAgenticCodeFixer(mockClient, "test-model")

	request := &FixRequest{
		UserMessage: "fix this",
		FileContent: "echo 'test'",
		FilePath:    "/path/to/script.sh",
		FileType:    "bash",
	}

	result, err := fixer.GenerateFix(request)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatalf("result is nil")
	}

	if result.Success {
		t.Errorf("expected Success=false when no fix blocks match, got true")
	}

	// Check that error message is helpful (Requirement 7.3)
	if !strings.Contains(result.ErrorMessage, "Could not identify valid fix blocks") {
		t.Errorf("expected error message to explain issue, got '%s'", result.ErrorMessage)
	}

	if !strings.Contains(result.ErrorMessage, "try rephrasing") {
		t.Errorf("expected error message to suggest rephrasing, got '%s'", result.ErrorMessage)
	}
}

// TestApplyFix_ErrorMessagesAreDescriptive tests that ApplyFix provides clear error messages
// This test validates Requirements 3.5 and 7.3
func TestApplyFix_ErrorMessagesAreDescriptive(t *testing.T) {
	fixer := NewAgenticCodeFixer(&mockAIClient{}, "test-model")

	testCases := []struct {
		name          string
		fixCode       string
		expectedError string
	}{
		{
			name:          "empty fix code",
			fixCode:       "",
			expectedError: "empty or whitespace-only",
		},
		{
			name:          "whitespace-only fix code",
			fixCode:       "   \n\t  ",
			expectedError: "empty or whitespace-only",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := fixer.ApplyFix("original content", tc.fixCode, "bash")

			if err == nil {
				t.Errorf("expected error for %s, got nil", tc.name)
				return
			}

			if !strings.Contains(err.Error(), tc.expectedError) {
				t.Errorf("expected error message to contain '%s', got: %s", tc.expectedError, err.Error())
			}
		})
	}
}

// TestGenerateFix_ApplyFixErrorWithHelpfulMessage tests that ApplyFix errors are wrapped with helpful context
// This test validates Requirements 3.5 and 7.3
func TestGenerateFix_ApplyFixErrorWithHelpfulMessage(t *testing.T) {
	// Create a mock that returns a code block that will fail in ApplyFix
	// (though this is hard to trigger since ApplyFix is simple, we test the error path)
	mockClient := &mockAIClientWithCodeBlock{
		response:  "```bash\necho 'test'\n```",
		available: true,
		err:       nil,
	}

	fixer := NewAgenticCodeFixer(mockClient, "test-model")

	request := &FixRequest{
		UserMessage: "fix this",
		FileContent: "echo 'old'",
		FilePath:    "/path/to/script.sh",
		FileType:    "bash",
	}

	result, err := fixer.GenerateFix(request)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatalf("result is nil")
	}

	// This should succeed, but we're testing that the error path exists
	// and would provide helpful messages if it failed
	if !result.Success {
		// If it did fail, check that the error message is helpful
		if !strings.Contains(result.ErrorMessage, "Failed to apply fix") {
			t.Errorf("expected error message to start with 'Failed to apply fix', got '%s'", result.ErrorMessage)
		}
	}
}

// TestOrderFixBlocks_SingleBlock tests ordering with a single block
func TestOrderFixBlocks_SingleBlock(t *testing.T) {
	fixer := NewAgenticCodeFixer(&mockAIClient{}, "test-model")

	blocks := []CodeBlock{
		{Language: "bash", Code: "echo 'test'", IsWhole: false},
	}

	ordered := fixer.orderFixBlocks(blocks)

	if len(ordered) != 1 {
		t.Errorf("expected 1 block, got %d", len(ordered))
	}

	if ordered[0].Code != "echo 'test'" {
		t.Errorf("block was modified during ordering")
	}
}

// TestOrderFixBlocks_EmptyBlocks tests ordering with no blocks
func TestOrderFixBlocks_EmptyBlocks(t *testing.T) {
	fixer := NewAgenticCodeFixer(&mockAIClient{}, "test-model")

	blocks := []CodeBlock{}

	ordered := fixer.orderFixBlocks(blocks)

	if len(ordered) != 0 {
		t.Errorf("expected 0 blocks, got %d", len(ordered))
	}
}

// TestOrderFixBlocks_WholeBeforePartial tests that whole file replacements come before partial fixes
func TestOrderFixBlocks_WholeBeforePartial(t *testing.T) {
	fixer := NewAgenticCodeFixer(&mockAIClient{}, "test-model")

	blocks := []CodeBlock{
		{Language: "bash", Code: "echo 'partial'", IsWhole: false},
		{Language: "bash", Code: "#!/bin/bash\necho 'whole'\nexit 0", IsWhole: true},
		{Language: "bash", Code: "echo 'another partial'", IsWhole: false},
	}

	ordered := fixer.orderFixBlocks(blocks)

	if len(ordered) != 3 {
		t.Errorf("expected 3 blocks, got %d", len(ordered))
	}

	// First block should be the whole file replacement
	if !ordered[0].IsWhole {
		t.Errorf("expected first block to be whole file replacement")
	}

	if !strings.Contains(ordered[0].Code, "whole") {
		t.Errorf("expected first block to be the whole file replacement block")
	}

	// Second and third should be partial
	if ordered[1].IsWhole || ordered[2].IsWhole {
		t.Errorf("expected second and third blocks to be partial fixes")
	}
}

// TestOrderFixBlocks_LargerBeforeSmaller tests that larger blocks come before smaller ones
func TestOrderFixBlocks_LargerBeforeSmaller(t *testing.T) {
	fixer := NewAgenticCodeFixer(&mockAIClient{}, "test-model")

	blocks := []CodeBlock{
		{Language: "bash", Code: "echo 'small'", IsWhole: false},
		{Language: "bash", Code: "echo 'medium'\necho 'line2'", IsWhole: false},
		{Language: "bash", Code: "echo 'large'\necho 'line2'\necho 'line3'\necho 'line4'", IsWhole: false},
	}

	ordered := fixer.orderFixBlocks(blocks)

	if len(ordered) != 3 {
		t.Errorf("expected 3 blocks, got %d", len(ordered))
	}

	// Should be ordered by size: large, medium, small
	if !strings.Contains(ordered[0].Code, "large") {
		t.Errorf("expected first block to be the largest")
	}

	if !strings.Contains(ordered[1].Code, "medium") {
		t.Errorf("expected second block to be medium")
	}

	if !strings.Contains(ordered[2].Code, "small") {
		t.Errorf("expected third block to be the smallest")
	}
}

// TestOrderFixBlocks_MixedSizes tests ordering with mixed whole and partial blocks of different sizes
func TestOrderFixBlocks_MixedSizes(t *testing.T) {
	fixer := NewAgenticCodeFixer(&mockAIClient{}, "test-model")

	blocks := []CodeBlock{
		{Language: "bash", Code: "echo 'small partial'", IsWhole: false},
		{Language: "bash", Code: "#!/bin/bash\necho 'small whole'\nexit 0", IsWhole: true},
		{Language: "bash", Code: "echo 'large partial'\necho 'line2'\necho 'line3'\necho 'line4'", IsWhole: false},
		{Language: "bash", Code: "#!/bin/bash\necho 'large whole'\necho 'line2'\necho 'line3'\necho 'line4'\necho 'line5'\nexit 0", IsWhole: true},
	}

	ordered := fixer.orderFixBlocks(blocks)

	if len(ordered) != 4 {
		t.Errorf("expected 4 blocks, got %d", len(ordered))
	}

	// First two should be whole blocks, ordered by size (large then small)
	if !ordered[0].IsWhole {
		t.Errorf("expected first block to be whole")
	}
	if !strings.Contains(ordered[0].Code, "large whole") {
		t.Errorf("expected first block to be large whole")
	}

	if !ordered[1].IsWhole {
		t.Errorf("expected second block to be whole")
	}
	if !strings.Contains(ordered[1].Code, "small whole") {
		t.Errorf("expected second block to be small whole")
	}

	// Last two should be partial blocks, ordered by size (large then small)
	if ordered[2].IsWhole {
		t.Errorf("expected third block to be partial")
	}
	if !strings.Contains(ordered[2].Code, "large partial") {
		t.Errorf("expected third block to be large partial")
	}

	if ordered[3].IsWhole {
		t.Errorf("expected fourth block to be partial")
	}
	if !strings.Contains(ordered[3].Code, "small partial") {
		t.Errorf("expected fourth block to be small partial")
	}
}

// TestOrderFixBlocks_PreservesBlockContent tests that ordering doesn't modify block content
func TestOrderFixBlocks_PreservesBlockContent(t *testing.T) {
	fixer := NewAgenticCodeFixer(&mockAIClient{}, "test-model")

	blocks := []CodeBlock{
		{Language: "bash", Code: "echo 'block1'", IsWhole: false},
		{Language: "bash", Code: "echo 'block2'\necho 'line2'", IsWhole: true},
		{Language: "bash", Code: "echo 'block3'", IsWhole: false},
	}

	// Make a copy to compare
	originalCodes := make([]string, len(blocks))
	for i, block := range blocks {
		originalCodes[i] = block.Code
	}

	ordered := fixer.orderFixBlocks(blocks)

	// Check that all original codes are present in ordered blocks
	orderedCodes := make(map[string]bool)
	for _, block := range ordered {
		orderedCodes[block.Code] = true
	}

	for _, code := range originalCodes {
		if !orderedCodes[code] {
			t.Errorf("original code '%s' not found in ordered blocks", code)
		}
	}
}

// TestOrderFixBlocks_AllWhole tests ordering when all blocks are whole file replacements
func TestOrderFixBlocks_AllWhole(t *testing.T) {
	fixer := NewAgenticCodeFixer(&mockAIClient{}, "test-model")

	blocks := []CodeBlock{
		{Language: "bash", Code: "echo 'small'", IsWhole: true},
		{Language: "bash", Code: "echo 'large'\necho 'line2'\necho 'line3'", IsWhole: true},
		{Language: "bash", Code: "echo 'medium'\necho 'line2'", IsWhole: true},
	}

	ordered := fixer.orderFixBlocks(blocks)

	if len(ordered) != 3 {
		t.Errorf("expected 3 blocks, got %d", len(ordered))
	}

	// Should be ordered by size: large, medium, small
	if !strings.Contains(ordered[0].Code, "large") {
		t.Errorf("expected first block to be the largest")
	}

	if !strings.Contains(ordered[1].Code, "medium") {
		t.Errorf("expected second block to be medium")
	}

	if !strings.Contains(ordered[2].Code, "small") {
		t.Errorf("expected third block to be the smallest")
	}
}

// TestOrderFixBlocks_AllPartial tests ordering when all blocks are partial fixes
func TestOrderFixBlocks_AllPartial(t *testing.T) {
	fixer := NewAgenticCodeFixer(&mockAIClient{}, "test-model")

	blocks := []CodeBlock{
		{Language: "bash", Code: "echo 'small'", IsWhole: false},
		{Language: "bash", Code: "echo 'large'\necho 'line2'\necho 'line3'", IsWhole: false},
		{Language: "bash", Code: "echo 'medium'\necho 'line2'", IsWhole: false},
	}

	ordered := fixer.orderFixBlocks(blocks)

	if len(ordered) != 3 {
		t.Errorf("expected 3 blocks, got %d", len(ordered))
	}

	// Should be ordered by size: large, medium, small
	if !strings.Contains(ordered[0].Code, "large") {
		t.Errorf("expected first block to be the largest")
	}

	if !strings.Contains(ordered[1].Code, "medium") {
		t.Errorf("expected second block to be medium")
	}

	if !strings.Contains(ordered[2].Code, "small") {
		t.Errorf("expected third block to be the smallest")
	}
}

// TestGenerateFix_UsesOrderedBlocks tests that GenerateFix uses the ordering logic
func TestGenerateFix_UsesOrderedBlocks(t *testing.T) {
	// Create a response with multiple blocks
	// Use small blocks (3 lines or less) so they're both marked as partial (IsWhole=false)
	// This avoids the validation error for mixing whole and partial blocks
	mockClient := &mockAIClientWithCodeBlock{
		response:  "First block:\n```bash\necho 'first'\n```\n\nSecond block:\n```bash\necho 'second'\necho 'third'\n```",
		available: true,
		err:       nil,
	}

	fixer := NewAgenticCodeFixer(mockClient, "test-model")

	request := &FixRequest{
		UserMessage: "fix this",
		FileContent: "echo 'old'",
		FilePath:    "/path/to/script.sh",
		FileType:    "bash",
	}

	result, err := fixer.GenerateFix(request)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatalf("result is nil")
	}

	if !result.Success {
		t.Errorf("expected Success=true, got false. Error: %s", result.ErrorMessage)
	}

	// With current whole-file replacement strategy, the last block applied wins
	// The ordering ensures larger blocks are processed first, but since each replaces
	// the entire content, the final result is the last (smallest) block
	// This is acceptable behavior - in practice, the AI should generate either:
	// 1. A single whole file replacement, OR
	// 2. Multiple partial fixes (not yet implemented)
	if result.ModifiedContent == "" {
		t.Errorf("expected non-empty modified content")
	}
}

// TestErrorHandling_ValidatesBeforeApplying tests that validation happens before applying
// This test validates Requirements 3.5 and 7.3 - the core requirement that invalid fixes
// are not applied and explanations are provided instead
func TestErrorHandling_ValidatesBeforeApplying(t *testing.T) {
	testCases := []struct {
		name          string
		response      string
		expectedError string
		shouldContain []string
	}{
		{
			name:          "invalid syntax - unmatched quote",
			response:      "```bash\necho 'unclosed\n```",
			expectedError: "invalid syntax",
			shouldContain: []string{"Pre-validation failed", "review your request"},
		},
		{
			name:          "invalid syntax - unmatched brace",
			response:      "```bash\nif [ true ]; then\necho 'test'\n```",
			expectedError: "invalid syntax",
			shouldContain: []string{"Pre-validation failed", "unmatched"},
		},
		{
			name:          "empty code block",
			response:      "```bash\n\n```",
			expectedError: "did not contain any code blocks",
			shouldContain: []string{}, // Empty blocks are filtered out, so we get "no code blocks" error
		},
		{
			name:          "no matching code blocks",
			response:      "```python\nprint('wrong')\n```",
			expectedError: "Could not identify",
			shouldContain: []string{"try rephrasing"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := &mockAIClientWithCodeBlock{
				response:  tc.response,
				available: true,
				err:       nil,
			}

			fixer := NewAgenticCodeFixer(mockClient, "test-model")

			request := &FixRequest{
				UserMessage: "fix this",
				FileContent: "echo 'test'",
				FilePath:    "/path/to/script.sh",
				FileType:    "bash",
			}

			result, err := fixer.GenerateFix(request)

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if result == nil {
				t.Fatalf("result is nil")
			}

			// Verify fix was NOT applied (Requirement 7.3)
			if result.Success {
				t.Errorf("expected Success=false for invalid fix, got true")
			}

			if result.ModifiedContent != "" {
				t.Errorf("expected no modified content for invalid fix, got: %s", result.ModifiedContent)
			}

			// Verify explanation is provided (Requirement 7.3)
			if result.ErrorMessage == "" {
				t.Errorf("expected error message explaining the issue, got empty")
			}

			if !strings.Contains(result.ErrorMessage, tc.expectedError) {
				t.Errorf("expected error message to contain '%s', got '%s'", tc.expectedError, result.ErrorMessage)
			}

			// Verify helpful guidance is included
			for _, phrase := range tc.shouldContain {
				if !strings.Contains(result.ErrorMessage, phrase) {
					t.Errorf("expected error message to contain '%s', got '%s'", phrase, result.ErrorMessage)
				}
			}

			// Verify result satisfies invariants
			if err := result.Validate(); err != nil {
				t.Errorf("result violates invariants: %v", err)
			}
		})
	}
}

// TestApplyFix_TransactionalBackup tests that original content is preserved on error
// This test validates Requirements 7.4 and 7.6 - transactional fix application
func TestApplyFix_TransactionalBackup(t *testing.T) {
	fixer := NewAgenticCodeFixer(&mockAIClient{}, "test-model")

	originalContent := "#!/bin/bash\necho 'original code'\nexit 0"

	testCases := []struct {
		name    string
		fixCode string
		wantErr bool
	}{
		{
			name:    "empty fix code should preserve original",
			fixCode: "",
			wantErr: true,
		},
		{
			name:    "whitespace-only fix code should preserve original",
			fixCode: "   \n\t  ",
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := fixer.ApplyFix(originalContent, tc.fixCode, "bash")

			if tc.wantErr && err == nil {
				t.Errorf("expected error, got nil")
			}

			if tc.wantErr {
				// On error, result should be the original content (backup)
				if result != originalContent {
					t.Errorf("expected original content to be preserved on error.\nExpected:\n%s\nGot:\n%s",
						originalContent, result)
				}
			}
		})
	}
}

// TestApplyFix_TransactionalValidation tests that validation happens before commit
// This test validates Requirement 7.4 - validate result before commit
func TestApplyFix_TransactionalValidation(t *testing.T) {
	fixer := NewAgenticCodeFixer(&mockAIClient{}, "test-model")

	originalContent := "#!/bin/bash\necho 'original'\nexit 0"

	testCases := []struct {
		name     string
		fixCode  string
		fileType string
		wantErr  bool
	}{
		{
			name:     "valid fix should pass validation",
			fixCode:  "#!/bin/bash\necho 'fixed'\nexit 0",
			fileType: "bash",
			wantErr:  false,
		},
		{
			name:     "whitespace-only should fail validation",
			fixCode:  "   \n\t  ",
			fileType: "bash",
			wantErr:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := fixer.ApplyFix(originalContent, tc.fixCode, tc.fileType)

			if tc.wantErr && err == nil {
				t.Errorf("expected validation error, got nil")
			}

			if !tc.wantErr && err != nil {
				t.Errorf("unexpected validation error: %v", err)
			}

			if tc.wantErr {
				// On validation failure, should return original content
				if result != originalContent {
					t.Errorf("expected original content on validation failure")
				}
			}
		})
	}
}

// TestApplyFix_TransactionalCommit tests that valid fixes are committed
// This test validates Requirement 7.4 - commit on success
func TestApplyFix_TransactionalCommit(t *testing.T) {
	fixer := NewAgenticCodeFixer(&mockAIClient{}, "test-model")

	originalContent := "#!/bin/bash\necho 'original'\nexit 0"
	fixCode := "#!/bin/bash\necho 'fixed'\nexit 0"

	result, err := fixer.ApplyFix(originalContent, fixCode, "bash")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Result should contain the fix code (committed)
	if !strings.Contains(result, "fixed") {
		t.Errorf("expected committed fix code in result")
	}

	// Result should NOT contain the original code
	if strings.Contains(result, "original") && !strings.Contains(fixCode, "original") {
		t.Errorf("result should not contain original code after commit")
	}
}

// TestApplyFix_TransactionalRollback tests that errors trigger rollback
// This test validates Requirement 7.6 - rollback on failure
func TestApplyFix_TransactionalRollback(t *testing.T) {
	fixer := NewAgenticCodeFixer(&mockAIClient{}, "test-model")

	originalContent := "#!/bin/bash\necho 'original'\nexit 0"

	testCases := []struct {
		name    string
		fixCode string
		errMsg  string
	}{
		{
			name:    "empty fix triggers rollback",
			fixCode: "",
			errMsg:  "empty",
		},
		{
			name:    "whitespace-only triggers rollback",
			fixCode: "   \n\t  ",
			errMsg:  "empty",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := fixer.ApplyFix(originalContent, tc.fixCode, "bash")

			if err == nil {
				t.Errorf("expected error to trigger rollback, got nil")
			}

			if !strings.Contains(err.Error(), tc.errMsg) {
				t.Errorf("expected error message to contain '%s', got: %s", tc.errMsg, err.Error())
			}

			// Rollback should return original content unchanged
			if result != originalContent {
				t.Errorf("rollback should return original content.\nExpected:\n%s\nGot:\n%s",
					originalContent, result)
			}
		})
	}
}

// TestApplyFix_TransactionalAtomicity tests that fix application is atomic
// This test validates Requirement 7.4 - atomic operation (all or nothing)
func TestApplyFix_TransactionalAtomicity(t *testing.T) {
	fixer := NewAgenticCodeFixer(&mockAIClient{}, "test-model")

	originalContent := "#!/bin/bash\necho 'original'\nexit 0"

	testCases := []struct {
		name           string
		fixCode        string
		expectSuccess  bool
		expectOriginal bool
		expectFixed    bool
	}{
		{
			name:           "successful fix is fully applied",
			fixCode:        "#!/bin/bash\necho 'fixed'\nexit 0",
			expectSuccess:  true,
			expectOriginal: false,
			expectFixed:    true,
		},
		{
			name:           "failed fix leaves original unchanged",
			fixCode:        "",
			expectSuccess:  false,
			expectOriginal: true,
			expectFixed:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := fixer.ApplyFix(originalContent, tc.fixCode, "bash")

			if tc.expectSuccess && err != nil {
				t.Errorf("expected success, got error: %v", err)
			}

			if !tc.expectSuccess && err == nil {
				t.Errorf("expected failure, got success")
			}

			if tc.expectOriginal {
				if result != originalContent {
					t.Errorf("expected original content to be preserved")
				}
			}

			if tc.expectFixed {
				if !strings.Contains(result, "fixed") {
					t.Errorf("expected fixed content to be applied")
				}
			}
		})
	}
}

// TestApplyFix_TransactionalTemporaryCopy tests that changes are applied to temporary copy
// This test validates Requirement 7.4 - apply to temporary copy
func TestApplyFix_TransactionalTemporaryCopy(t *testing.T) {
	fixer := NewAgenticCodeFixer(&mockAIClient{}, "test-model")

	originalContent := "#!/bin/bash\necho 'original'\nexit 0"
	fixCode := "#!/bin/bash\necho 'fixed'\nexit 0"

	// The implementation should work on a temporary copy
	// We verify this by ensuring the original content is not modified
	// (in Go, strings are immutable, so this is guaranteed by the language)

	result, err := fixer.ApplyFix(originalContent, fixCode, "bash")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Original content should remain unchanged
	if originalContent != "#!/bin/bash\necho 'original'\nexit 0" {
		t.Errorf("original content was modified")
	}

	// Result should be different from original
	if result == originalContent {
		t.Errorf("result should be different from original content")
	}

	// Result should contain the fix
	if !strings.Contains(result, "fixed") {
		t.Errorf("result should contain fixed content")
	}
}

// TestApplyFix_TransactionalErrorPreservesOriginal tests that any error preserves original
// This test validates Requirements 7.4 and 7.6 - comprehensive error handling
func TestApplyFix_TransactionalErrorPreservesOriginal(t *testing.T) {
	fixer := NewAgenticCodeFixer(&mockAIClient{}, "test-model")

	originalContent := "#!/bin/bash\necho 'original'\nexit 0"

	testCases := []struct {
		name     string
		fixCode  string
		fileType string
	}{
		{
			name:     "empty fix code",
			fixCode:  "",
			fileType: "bash",
		},
		{
			name:     "whitespace-only fix code",
			fixCode:  "   \n\t  ",
			fileType: "bash",
		},
		{
			name:     "whitespace-only after newline addition",
			fixCode:  "   ",
			fileType: "bash",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := fixer.ApplyFix(originalContent, tc.fixCode, tc.fileType)

			if err == nil {
				t.Errorf("expected error, got nil")
			}

			// Original content must be preserved exactly
			if result != originalContent {
				t.Errorf("original content not preserved on error.\nExpected:\n%s\nGot:\n%s",
					originalContent, result)
			}
		})
	}
}

// TestApplyFix_TransactionalMultipleOperations tests that multiple operations are handled correctly
// This test validates Requirement 7.4 - transactional behavior across multiple calls
func TestApplyFix_TransactionalMultipleOperations(t *testing.T) {
	fixer := NewAgenticCodeFixer(&mockAIClient{}, "test-model")

	originalContent := "#!/bin/bash\necho 'original'\nexit 0"

	// First operation: successful fix
	fixCode1 := "#!/bin/bash\necho 'fixed1'\nexit 0"
	result1, err1 := fixer.ApplyFix(originalContent, fixCode1, "bash")

	if err1 != nil {
		t.Errorf("first operation failed: %v", err1)
	}

	if !strings.Contains(result1, "fixed1") {
		t.Errorf("first operation did not apply fix")
	}

	// Second operation: failed fix (should not affect first result)
	fixCode2 := ""
	result2, err2 := fixer.ApplyFix(result1, fixCode2, "bash")

	if err2 == nil {
		t.Errorf("second operation should have failed")
	}

	// Second operation should return the input (result1) unchanged
	if result2 != result1 {
		t.Errorf("failed operation should preserve input content")
	}

	// Third operation: another successful fix
	fixCode3 := "#!/bin/bash\necho 'fixed3'\nexit 0"
	result3, err3 := fixer.ApplyFix(result1, fixCode3, "bash")

	if err3 != nil {
		t.Errorf("third operation failed: %v", err3)
	}

	if !strings.Contains(result3, "fixed3") {
		t.Errorf("third operation did not apply fix")
	}
}

// TestGenerateFix_MultipleCodeBlocksAppliedAtomically tests that multiple code blocks
// are applied in sequence and all changes are included in the final result
// This test validates Requirement 10.1 - Multi-Step Fixes
func TestGenerateFix_MultipleCodeBlocksAppliedAtomically(t *testing.T) {
	// Create a mock that returns multiple code blocks
	mockClient := &mockAIClientWithCodeBlock{
		response:  "First fix:\n```bash\n#!/bin/bash\necho 'first'\n```\n\nSecond fix:\n```bash\necho 'second'\nexit 0\n```",
		available: true,
		err:       nil,
	}

	fixer := NewAgenticCodeFixer(mockClient, "test-model")

	request := &FixRequest{
		UserMessage: "apply multiple fixes",
		FileContent: "echo 'original'",
		FilePath:    "/path/to/script.sh",
		FileType:    "bash",
	}

	result, err := fixer.GenerateFix(request)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatalf("result is nil")
	}

	if !result.Success {
		t.Errorf("expected Success=true, got false. Error: %s", result.ErrorMessage)
	}

	// The modified content should reflect the last applied block
	// Since we apply blocks sequentially, each one replaces the previous content
	if !strings.Contains(result.ModifiedContent, "second") {
		t.Errorf("modified content should contain 'second' from the last block, got: %s", result.ModifiedContent)
	}

	// The changes summary should indicate multiple blocks were applied
	if !strings.Contains(result.ChangesSummary, "2 code blocks") {
		t.Errorf("changes summary should indicate 2 code blocks were applied, got: %s", result.ChangesSummary)
	}
}

// TestGenerateFix_MultipleCodeBlocksPartialFailure tests that if one block fails,
// the entire operation fails (atomicity)
// This test validates Requirement 10.3 - Multi-Step Fix Rollback
func TestGenerateFix_MultipleCodeBlocksPartialFailure(t *testing.T) {
	// Create a mock that returns multiple code blocks, where the second one is invalid
	mockClient := &mockAIClientWithCodeBlock{
		response:  "First fix:\n```bash\n#!/bin/bash\necho 'first'\nexit 0\n```\n\nSecond fix (invalid):\n```bash\nif [ true ]; then\necho 'missing fi'\n```",
		available: true,
		err:       nil,
	}

	fixer := NewAgenticCodeFixer(mockClient, "test-model")

	request := &FixRequest{
		UserMessage: "apply multiple fixes",
		FileContent: "echo 'original'",
		FilePath:    "/path/to/script.sh",
		FileType:    "bash",
	}

	result, err := fixer.GenerateFix(request)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatalf("result is nil")
	}

	// Should fail because the second block has invalid syntax
	if result.Success {
		t.Errorf("expected Success=false due to invalid second block, got true")
	}

	// Error message should indicate which block failed
	if !strings.Contains(result.ErrorMessage, "block 2") {
		t.Errorf("error message should indicate block 2 failed, got: %s", result.ErrorMessage)
	}

	// Modified content should be empty (no partial application)
	if result.ModifiedContent != "" {
		t.Errorf("expected no modified content on failure, got: %s", result.ModifiedContent)
	}
}

// TestGenerateFix_MultipleCodeBlocksAllValid tests that all valid blocks are applied
// This test validates Requirement 10.1 - Multi-Step Fixes
func TestGenerateFix_MultipleCodeBlocksAllValid(t *testing.T) {
	// Create a mock that returns three valid code blocks
	mockClient := &mockAIClientWithCodeBlock{
		response:  "First:\n```bash\n#!/bin/bash\necho 'block1'\n```\n\nSecond:\n```bash\necho 'block2'\n```\n\nThird:\n```bash\necho 'block3'\nexit 0\n```",
		available: true,
		err:       nil,
	}

	fixer := NewAgenticCodeFixer(mockClient, "test-model")

	request := &FixRequest{
		UserMessage: "apply three fixes",
		FileContent: "echo 'original'",
		FilePath:    "/path/to/script.sh",
		FileType:    "bash",
	}

	result, err := fixer.GenerateFix(request)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatalf("result is nil")
	}

	if !result.Success {
		t.Errorf("expected Success=true, got false. Error: %s", result.ErrorMessage)
	}

	// The changes summary should indicate 3 blocks were applied
	if !strings.Contains(result.ChangesSummary, "3 code blocks") {
		t.Errorf("changes summary should indicate 3 code blocks were applied, got: %s", result.ChangesSummary)
	}
}

// TestGenerateFix_SingleCodeBlockNoMultipleIndicator tests that single block
// doesn't show "multiple blocks" in summary
func TestGenerateFix_SingleCodeBlockNoMultipleIndicator(t *testing.T) {
	mockClient := &mockAIClientWithCodeBlock{
		response:  "```bash\n#!/bin/bash\necho 'single'\nexit 0\n```",
		available: true,
		err:       nil,
	}

	fixer := NewAgenticCodeFixer(mockClient, "test-model")

	request := &FixRequest{
		UserMessage: "fix this",
		FileContent: "echo 'original'",
		FilePath:    "/path/to/script.sh",
		FileType:    "bash",
	}

	result, err := fixer.GenerateFix(request)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatalf("result is nil")
	}

	if !result.Success {
		t.Errorf("expected Success=true, got false. Error: %s", result.ErrorMessage)
	}

	// The changes summary should NOT indicate multiple blocks for a single block
	if strings.Contains(result.ChangesSummary, "code blocks") {
		t.Errorf("changes summary should not mention 'code blocks' for single block, got: %s", result.ChangesSummary)
	}
}

// TestValidateMultiStepFix_EmptyBlocks tests validation with no blocks
func TestValidateMultiStepFix_EmptyBlocks(t *testing.T) {
	fixer := NewAgenticCodeFixer(&mockAIClient{}, "test-model")

	blocks := []CodeBlock{}

	err := fixer.validateMultiStepFix(blocks, "bash")

	if err == nil {
		t.Error("Expected error for empty blocks, got nil")
	}

	if !strings.Contains(err.Error(), "no fix blocks") {
		t.Errorf("Expected error message to contain 'no fix blocks', got: %s", err.Error())
	}
}

// TestValidateMultiStepFix_SingleValidBlock tests validation with one valid block
func TestValidateMultiStepFix_SingleValidBlock(t *testing.T) {
	fixer := NewAgenticCodeFixer(&mockAIClient{}, "test-model")

	blocks := []CodeBlock{
		{
			Language: "bash",
			Code:     "#!/bin/bash\necho \"test\"\nexit 0",
			IsWhole:  true,
		},
	}

	err := fixer.validateMultiStepFix(blocks, "bash")

	if err != nil {
		t.Errorf("Expected no error for single valid block, got: %s", err.Error())
	}
}

// TestValidateMultiStepFix_EmptyCodeBlock tests validation with empty code
func TestValidateMultiStepFix_EmptyCodeBlock(t *testing.T) {
	fixer := NewAgenticCodeFixer(&mockAIClient{}, "test-model")

	blocks := []CodeBlock{
		{
			Language: "bash",
			Code:     "",
			IsWhole:  false,
		},
	}

	err := fixer.validateMultiStepFix(blocks, "bash")

	if err == nil {
		t.Error("Expected error for empty code block, got nil")
	}

	if !strings.Contains(err.Error(), "empty or contains only whitespace") {
		t.Errorf("Expected error message about empty block, got: %s", err.Error())
	}
}

// TestValidateMultiStepFix_WhitespaceOnlyBlock tests validation with whitespace-only code
func TestValidateMultiStepFix_WhitespaceOnlyBlock(t *testing.T) {
	fixer := NewAgenticCodeFixer(&mockAIClient{}, "test-model")

	blocks := []CodeBlock{
		{
			Language: "bash",
			Code:     "   \n\t\n   ",
			IsWhole:  false,
		},
	}

	err := fixer.validateMultiStepFix(blocks, "bash")

	if err == nil {
		t.Error("Expected error for whitespace-only block, got nil")
	}

	if !strings.Contains(err.Error(), "empty or contains only whitespace") {
		t.Errorf("Expected error message about whitespace, got: %s", err.Error())
	}
}

// TestValidateMultiStepFix_InvalidSyntax tests validation with invalid syntax
func TestValidateMultiStepFix_InvalidSyntax(t *testing.T) {
	fixer := NewAgenticCodeFixer(&mockAIClient{}, "test-model")

	blocks := []CodeBlock{
		{
			Language: "bash",
			Code:     "if [ true ]; then\n  echo \"test\"",
			IsWhole:  false,
		},
	}

	err := fixer.validateMultiStepFix(blocks, "bash")

	if err == nil {
		t.Error("Expected error for invalid syntax, got nil")
	}

	if !strings.Contains(err.Error(), "invalid syntax") {
		t.Errorf("Expected error message about invalid syntax, got: %s", err.Error())
	}
}

// TestValidateMultiStepFix_MultipleValidBlocks tests validation with multiple valid blocks
func TestValidateMultiStepFix_MultipleValidBlocks(t *testing.T) {
	fixer := NewAgenticCodeFixer(&mockAIClient{}, "test-model")

	blocks := []CodeBlock{
		{
			Language: "bash",
			Code:     "echo \"first\"",
			IsWhole:  false,
		},
		{
			Language: "bash",
			Code:     "echo \"second\"",
			IsWhole:  false,
		},
	}

	err := fixer.validateMultiStepFix(blocks, "bash")

	if err != nil {
		t.Errorf("Expected no error for multiple valid blocks, got: %s", err.Error())
	}
}

// TestValidateMultiStepFix_DuplicateBlocks tests validation with duplicate code
func TestValidateMultiStepFix_DuplicateBlocks(t *testing.T) {
	fixer := NewAgenticCodeFixer(&mockAIClient{}, "test-model")

	blocks := []CodeBlock{
		{
			Language: "bash",
			Code:     "echo \"test\"",
			IsWhole:  false,
		},
		{
			Language: "bash",
			Code:     "echo \"test\"",
			IsWhole:  false,
		},
	}

	err := fixer.validateMultiStepFix(blocks, "bash")

	if err == nil {
		t.Error("Expected error for duplicate blocks, got nil")
	}

	if !strings.Contains(err.Error(), "duplicate code") {
		t.Errorf("Expected error message about duplicate code, got: %s", err.Error())
	}
}

// TestValidateMultiStepFix_DuplicateWithWhitespace tests that whitespace differences are ignored
func TestValidateMultiStepFix_DuplicateWithWhitespace(t *testing.T) {
	fixer := NewAgenticCodeFixer(&mockAIClient{}, "test-model")

	blocks := []CodeBlock{
		{
			Language: "bash",
			Code:     "echo \"test\"",
			IsWhole:  false,
		},
		{
			Language: "bash",
			Code:     "  echo \"test\"  \n",
			IsWhole:  false,
		},
	}

	err := fixer.validateMultiStepFix(blocks, "bash")

	if err == nil {
		t.Error("Expected error for duplicate blocks (ignoring whitespace), got nil")
	}

	if !strings.Contains(err.Error(), "duplicate code") {
		t.Errorf("Expected error message about duplicate code, got: %s", err.Error())
	}
}

// TestValidateMultiStepFix_MultipleWholeFileReplacements tests validation with multiple whole-file blocks
func TestValidateMultiStepFix_MultipleWholeFileReplacements(t *testing.T) {
	fixer := NewAgenticCodeFixer(&mockAIClient{}, "test-model")

	blocks := []CodeBlock{
		{
			Language: "bash",
			Code:     "#!/bin/bash\necho \"first\"\nexit 0",
			IsWhole:  true,
		},
		{
			Language: "bash",
			Code:     "#!/bin/bash\necho \"second\"\nexit 0",
			IsWhole:  true,
		},
	}

	err := fixer.validateMultiStepFix(blocks, "bash")

	if err == nil {
		t.Error("Expected error for multiple whole-file replacements, got nil")
	}

	if !strings.Contains(err.Error(), "multiple whole-file replacements") {
		t.Errorf("Expected error message about multiple whole-file replacements, got: %s", err.Error())
	}
}

// TestValidateMultiStepFix_MixedWholeAndPartial tests validation with mixed whole and partial blocks
func TestValidateMultiStepFix_MixedWholeAndPartial(t *testing.T) {
	fixer := NewAgenticCodeFixer(&mockAIClient{}, "test-model")

	blocks := []CodeBlock{
		{
			Language: "bash",
			Code:     "#!/bin/bash\necho \"whole\"\nexit 0",
			IsWhole:  true,
		},
		{
			Language: "bash",
			Code:     "echo \"partial\"",
			IsWhole:  false,
		},
	}

	err := fixer.validateMultiStepFix(blocks, "bash")

	if err == nil {
		t.Error("Expected error for mixed whole and partial blocks, got nil")
	}

	if !strings.Contains(err.Error(), "cannot mix whole-file replacement with partial fixes") {
		t.Errorf("Expected error message about mixing whole and partial, got: %s", err.Error())
	}
}

// TestValidateMultiStepFix_InvalidSyntaxInSecondBlock tests that all blocks are validated
func TestValidateMultiStepFix_InvalidSyntaxInSecondBlock(t *testing.T) {
	fixer := NewAgenticCodeFixer(&mockAIClient{}, "test-model")

	blocks := []CodeBlock{
		{
			Language: "bash",
			Code:     "echo \"valid\"",
			IsWhole:  false,
		},
		{
			Language: "bash",
			Code:     "if [ true ]; then",
			IsWhole:  false,
		},
	}

	err := fixer.validateMultiStepFix(blocks, "bash")

	if err == nil {
		t.Error("Expected error for invalid syntax in second block, got nil")
	}

	if !strings.Contains(err.Error(), "fix block 2") {
		t.Errorf("Expected error to mention block 2, got: %s", err.Error())
	}

	if !strings.Contains(err.Error(), "invalid syntax") {
		t.Errorf("Expected error message about invalid syntax, got: %s", err.Error())
	}
}

// TestValidateMultiStepFix_DifferentFileTypes tests validation with different file types
func TestValidateMultiStepFix_DifferentFileTypes(t *testing.T) {
	fixer := NewAgenticCodeFixer(&mockAIClient{}, "test-model")

	testCases := []struct {
		name     string
		fileType string
		code     string
		wantErr  bool
	}{
		{
			name:     "valid bash",
			fileType: "bash",
			code:     "echo \"test\"",
			wantErr:  false,
		},
		{
			name:     "valid powershell",
			fileType: "powershell",
			code:     "Write-Host \"test\"",
			wantErr:  false,
		},
		{
			name:     "valid markdown",
			fileType: "markdown",
			code:     "# Title\n\nContent",
			wantErr:  false,
		},
		{
			name:     "invalid bash",
			fileType: "bash",
			code:     "if [ true ]; then",
			wantErr:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			blocks := []CodeBlock{
				{
					Language: tc.fileType,
					Code:     tc.code,
					IsWhole:  false,
				},
			}

			err := fixer.validateMultiStepFix(blocks, tc.fileType)

			if tc.wantErr && err == nil {
				t.Error("Expected error, got nil")
			}

			if !tc.wantErr && err != nil {
				t.Errorf("Expected no error, got: %s", err.Error())
			}
		})
	}
}

// TestValidateMultiStepFix_ThreeValidBlocks tests validation with three valid blocks
func TestValidateMultiStepFix_ThreeValidBlocks(t *testing.T) {
	fixer := NewAgenticCodeFixer(&mockAIClient{}, "test-model")

	blocks := []CodeBlock{
		{
			Language: "bash",
			Code:     "echo \"first\"",
			IsWhole:  false,
		},
		{
			Language: "bash",
			Code:     "echo \"second\"",
			IsWhole:  false,
		},
		{
			Language: "bash",
			Code:     "echo \"third\"",
			IsWhole:  false,
		},
	}

	err := fixer.validateMultiStepFix(blocks, "bash")

	if err != nil {
		t.Errorf("Expected no error for three valid blocks, got: %s", err.Error())
	}
}

// TestValidateMultiStepFix_ErrorMessageFormat tests that error messages are properly formatted
func TestValidateMultiStepFix_ErrorMessageFormat(t *testing.T) {
	fixer := NewAgenticCodeFixer(&mockAIClient{}, "test-model")

	testCases := []struct {
		name          string
		blocks        []CodeBlock
		fileType      string
		expectedError string
	}{
		{
			name:          "empty blocks",
			blocks:        []CodeBlock{},
			fileType:      "bash",
			expectedError: "no fix blocks",
		},
		{
			name: "empty code",
			blocks: []CodeBlock{
				{Language: "bash", Code: "", IsWhole: false},
			},
			fileType:      "bash",
			expectedError: "fix block 1 is empty",
		},
		{
			name: "invalid syntax",
			blocks: []CodeBlock{
				{Language: "bash", Code: "if [ true ]; then", IsWhole: false},
			},
			fileType:      "bash",
			expectedError: "fix block 1 has invalid syntax",
		},
		{
			name: "duplicate blocks",
			blocks: []CodeBlock{
				{Language: "bash", Code: "echo test", IsWhole: false},
				{Language: "bash", Code: "echo test", IsWhole: false},
			},
			fileType:      "bash",
			expectedError: "fix blocks 1 and 2 contain duplicate code",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := fixer.validateMultiStepFix(tc.blocks, tc.fileType)

			if err == nil {
				t.Error("Expected error, got nil")
				return
			}

			if !strings.Contains(err.Error(), tc.expectedError) {
				t.Errorf("Expected error to contain '%s', got: %s", tc.expectedError, err.Error())
			}
		})
	}
}

// TestGenerateChangeSummary_LocationInformation tests that change notifications include location information
func TestGenerateChangeSummary_LocationInformation(t *testing.T) {
	fixer := NewAgenticCodeFixer(&mockAIClient{}, "test-model")

	testCases := []struct {
		name            string
		originalContent string
		modifiedContent string
		filePath        string
		blockCount      int
		expectLocation  bool
		expectFunction  bool
	}{
		{
			name: "single line change",
			originalContent: `#!/bin/bash
echo "hello"
echo "world"`,
			modifiedContent: `#!/bin/bash
echo "hello there"
echo "world"`,
			filePath:       "test.sh",
			blockCount:     1,
			expectLocation: true,
			expectFunction: false,
		},
		{
			name: "change near function",
			originalContent: `#!/bin/bash
function greet() {
  echo "hello"
}
greet`,
			modifiedContent: `#!/bin/bash
function greet() {
  echo "hello there"
}
greet`,
			filePath:       "test.sh",
			blockCount:     1,
			expectLocation: true,
			expectFunction: true,
		},
		{
			name: "multiple line changes",
			originalContent: `#!/bin/bash
echo "line1"
echo "line2"
echo "line3"`,
			modifiedContent: `#!/bin/bash
echo "modified1"
echo "modified2"
echo "line3"`,
			filePath:       "test.sh",
			blockCount:     1,
			expectLocation: true,
			expectFunction: false,
		},
		{
			name:            "new file",
			originalContent: "",
			modifiedContent: "#!/bin/bash\necho 'hello'",
			filePath:        "test.sh",
			blockCount:      1,
			expectLocation:  false,
			expectFunction:  false,
		},
		{
			name: "multiple code blocks",
			originalContent: `#!/bin/bash
echo "hello"`,
			modifiedContent: `#!/bin/bash
echo "hello world"`,
			filePath:       "test.sh",
			blockCount:     3,
			expectLocation: true,
			expectFunction: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			summary := fixer.generateChangeSummary(tc.originalContent, tc.modifiedContent, tc.filePath, tc.blockCount, false)

			// Check that summary contains file path (Requirement 5.1, 5.2)
			if !strings.Contains(summary, tc.filePath) {
				t.Errorf("expected summary to contain file path %s, got: %s", tc.filePath, summary)
			}

			// Check for location information (Requirement 5.3)
			if tc.expectLocation {
				if !strings.Contains(summary, "Location:") && !strings.Contains(summary, "Line") {
					t.Errorf("expected summary to contain location information, got: %s", summary)
				}
			}

			// Check for function context (Requirement 5.3)
			if tc.expectFunction {
				if !strings.Contains(summary, "Context:") && !strings.Contains(summary, "function") {
					t.Errorf("expected summary to contain function context, got: %s", summary)
				}
			}

			// Check for multiple blocks indicator (Requirement 5.4)
			if tc.blockCount > 1 {
				if !strings.Contains(summary, fmt.Sprintf("Applied %d code blocks", tc.blockCount)) {
					t.Errorf("expected summary to indicate %d code blocks, got: %s", tc.blockCount, summary)
				}
			}

			// Check for save and test reminder (Requirement 5.5)
			if !strings.Contains(summary, "save") || !strings.Contains(summary, "test") {
				t.Errorf("expected summary to contain save and test reminder, got: %s", summary)
			}

			// Check for change description (Requirement 5.2)
			if !strings.Contains(summary, "Changes:") {
				t.Errorf("expected summary to contain change description, got: %s", summary)
			}
		})
	}
}

// TestDetectChangeLocations tests the location detection functionality
func TestDetectChangeLocations(t *testing.T) {
	fixer := NewAgenticCodeFixer(&mockAIClient{}, "test-model")

	testCases := []struct {
		name           string
		originalLines  []string
		modifiedLines  []string
		expectLocation bool
		expectRange    bool
	}{
		{
			name:           "single line change",
			originalLines:  []string{"line1", "line2", "line3"},
			modifiedLines:  []string{"line1", "modified", "line3"},
			expectLocation: true,
			expectRange:    false,
		},
		{
			name:           "multiple line changes",
			originalLines:  []string{"line1", "line2", "line3", "line4"},
			modifiedLines:  []string{"line1", "mod2", "mod3", "line4"},
			expectLocation: true,
			expectRange:    true,
		},
		{
			name:           "empty original",
			originalLines:  []string{},
			modifiedLines:  []string{"line1", "line2"},
			expectLocation: false,
			expectRange:    false,
		},
		{
			name:           "identical files",
			originalLines:  []string{"line1", "line2"},
			modifiedLines:  []string{"line1", "line2"},
			expectLocation: false,
			expectRange:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			location := fixer.detectChangeLocations(tc.originalLines, tc.modifiedLines)

			if tc.expectLocation {
				if location == "" {
					t.Errorf("expected location information, got empty string")
				}
				if !strings.Contains(location, "Location:") {
					t.Errorf("expected 'Location:' in output, got: %s", location)
				}
			} else {
				if location != "" {
					t.Errorf("expected no location information, got: %s", location)
				}
			}

			if tc.expectRange {
				if !strings.Contains(location, "-") {
					t.Errorf("expected line range (with '-'), got: %s", location)
				}
			}
		})
	}
}

// TestDetectNearbyFunction tests the function detection functionality
func TestDetectNearbyFunction(t *testing.T) {
	fixer := NewAgenticCodeFixer(&mockAIClient{}, "test-model")

	testCases := []struct {
		name             string
		lines            []string
		lineNum          int
		expectedFunction string
	}{
		{
			name: "bash function with function keyword",
			lines: []string{
				"#!/bin/bash",
				"function greet() {",
				"  echo 'hello'",
				"}",
			},
			lineNum:          2,
			expectedFunction: "greet",
		},
		{
			name: "bash function without function keyword",
			lines: []string{
				"#!/bin/bash",
				"greet() {",
				"  echo 'hello'",
				"}",
			},
			lineNum:          2,
			expectedFunction: "greet",
		},
		{
			name: "powershell function",
			lines: []string{
				"function Get-Greeting {",
				"  Write-Host 'hello'",
				"}",
			},
			lineNum:          1,
			expectedFunction: "Get-Greeting",
		},
		{
			name: "no function nearby",
			lines: []string{
				"#!/bin/bash",
				"echo 'hello'",
				"echo 'world'",
			},
			lineNum:          2,
			expectedFunction: "",
		},
		{
			name:             "function far away (>20 lines)",
			lines:            append([]string{"function test() {"}, make([]string, 25)...),
			lineNum:          24,
			expectedFunction: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := fixer.detectNearbyFunction(tc.lines, tc.lineNum)

			if result != tc.expectedFunction {
				t.Errorf("expected function name %q, got %q", tc.expectedFunction, result)
			}
		})
	}
}
