package projectctx

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Test 1: Empty workspace directory — Build should succeed with empty FileTree and KeyFiles.
// Validates: Requirements 1.1, 1.2
func TestBuild_EmptyWorkspace(t *testing.T) {
	dir := t.TempDir()

	cb := NewContextBuilder()
	meta, err := cb.Build(dir)
	if err != nil {
		t.Fatalf("Build failed on empty workspace: %v", err)
	}

	if len(meta.FileTree) != 0 {
		t.Errorf("expected empty FileTree, got %d entries", len(meta.FileTree))
	}
	if len(meta.KeyFiles) != 0 {
		t.Errorf("expected empty KeyFiles, got %d entries", len(meta.KeyFiles))
	}
	if meta.TotalFiles != 0 {
		t.Errorf("expected TotalFiles=0, got %d", meta.TotalFiles)
	}
	if meta.FileTreeTruncated {
		t.Error("expected FileTreeTruncated=false for empty workspace")
	}
}

// Test 2: Workspace with only skip directories — Build should succeed with empty FileTree.
// Validates: Requirements 1.1
func TestBuild_OnlySkipDirectories(t *testing.T) {
	dir := t.TempDir()

	// Create all skip directories with files inside them.
	for skipDir := range SkipDirs {
		sdPath := filepath.Join(dir, skipDir)
		if err := os.MkdirAll(sdPath, 0755); err != nil {
			t.Fatalf("failed to create skip dir %s: %v", skipDir, err)
		}
		if err := os.WriteFile(filepath.Join(sdPath, "hidden.txt"), []byte("should be skipped"), 0644); err != nil {
			t.Fatalf("failed to write file in skip dir: %v", err)
		}
	}

	cb := NewContextBuilder()
	meta, err := cb.Build(dir)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if len(meta.FileTree) != 0 {
		t.Errorf("expected empty FileTree when only skip dirs exist, got %d entries: %v", len(meta.FileTree), meta.FileTree)
	}
	if len(meta.KeyFiles) != 0 {
		t.Errorf("expected empty KeyFiles, got %d entries", len(meta.KeyFiles))
	}
}

// Test 3: Specific key file detection — go.mod and README.md should appear in KeyFiles.
// Validates: Requirements 1.2
func TestBuild_SpecificKeyFileDetection(t *testing.T) {
	dir := t.TempDir()

	goModContent := "module example.com/myproject\n\ngo 1.21\n"
	readmeContent := "# My Project\n\nThis is a test project.\n"

	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goModContent), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte(readmeContent), 0644); err != nil {
		t.Fatalf("failed to write README.md: %v", err)
	}

	cb := NewContextBuilder()
	meta, err := cb.Build(dir)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if _, ok := meta.KeyFiles["go.mod"]; !ok {
		t.Error("go.mod not found in KeyFiles")
	}
	if _, ok := meta.KeyFiles["README.md"]; !ok {
		t.Error("README.md not found in KeyFiles")
	}
	if got := meta.KeyFiles["go.mod"]; got != goModContent {
		t.Errorf("go.mod content mismatch: got %q, want %q", got, goModContent)
	}
	if got := meta.KeyFiles["README.md"]; got != readmeContent {
		t.Errorf("README.md content mismatch: got %q, want %q", got, readmeContent)
	}
	if meta.Language != "go" {
		t.Errorf("expected Language=go, got %q", meta.Language)
	}
	if meta.BuildSystem != "go modules" {
		t.Errorf("expected BuildSystem='go modules', got %q", meta.BuildSystem)
	}
}

// Test 4: Classification of specific messages.
// Validates: Requirements 3.2, 3.3, 3.5
func TestClassify_SpecificMessages(t *testing.T) {
	qc := NewQueryClassifier()

	tests := []struct {
		message             string
		wantNeedsContext    bool
		description         string
	}{
		{
			message:          "how do I run this project?",
			wantNeedsContext: true,
			description:      "question indicator 'how' + project term 'run' + '?' → needs context",
		},
		{
			message:          "hello",
			wantNeedsContext: false,
			description:      "no question indicator, no project term → no context",
		},
		{
			message:          "fix the bug",
			wantNeedsContext: false,
			description:      "no question indicator + project term combo → no context",
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			result := qc.Classify(tc.message)
			if result.NeedsProjectContext != tc.wantNeedsContext {
				t.Errorf("Classify(%q): NeedsProjectContext=%v, want %v",
					tc.message, result.NeedsProjectContext, tc.wantNeedsContext)
			}
		})
	}
}

// Test 5: Prompt output format with known inputs.
// Validates: Requirements 4.2
func TestPromptBuild_OutputFormat(t *testing.T) {
	meta := &ProjectMetadata{
		RootDir:     "/test/workspace",
		Language:    "go",
		BuildSystem: "go modules",
		KeyFiles: map[string]string{
			"README.md": "# Test Project",
			"go.mod":    "module test\n\ngo 1.21",
		},
		FileTree:          []string{"main.go", "go.mod", "README.md"},
		FileTreeTruncated: false,
		TotalFiles:        3,
		TotalContextBytes: 100,
	}

	pb := NewPromptBuilder()
	prompt := pb.Build(meta, "how do I build this?", nil, "")

	// Verify structure contains expected sections.
	expectedSections := []string{
		"## Project File Tree",
		"## Project Info",
		"## Key Project Files",
		"## User Question",
	}
	for _, section := range expectedSections {
		if !strings.Contains(prompt, section) {
			t.Errorf("prompt missing section %q", section)
		}
	}

	// Verify content is present.
	if !strings.Contains(prompt, "main.go") {
		t.Error("prompt missing file tree entry 'main.go'")
	}
	if !strings.Contains(prompt, "Language: go") {
		t.Error("prompt missing language info")
	}
	if !strings.Contains(prompt, "Build System: go modules") {
		t.Error("prompt missing build system info")
	}
	if !strings.Contains(prompt, "# Test Project") {
		t.Error("prompt missing README.md content")
	}
	if !strings.Contains(prompt, "how do I build this?") {
		t.Error("prompt missing user message")
	}
	if !strings.Contains(prompt, "project-aware") {
		t.Error("prompt missing system instruction")
	}
	if !strings.Contains(prompt, "<!-- context:") {
		t.Error("prompt missing debug byte count comment")
	}
}

// Test 6: Non-existent workspace directory returns error.
// Validates: Requirements 1.6
func TestBuild_NonExistentDirectory(t *testing.T) {
	cb := NewContextBuilder()
	_, err := cb.Build("/nonexistent/path/that/does/not/exist")
	if err == nil {
		t.Fatal("expected error for non-existent directory, got nil")
	}
	if !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("expected error message to mention 'does not exist', got: %v", err)
	}
}

// Test 7: Unreadable key file is skipped gracefully.
// Validates: Requirements 1.6
func TestBuild_UnreadableKeyFileSkipped(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping unreadable file test on Windows")
	}

	dir := t.TempDir()

	// Create a readable README.md.
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Hello"), 0644); err != nil {
		t.Fatalf("failed to write README.md: %v", err)
	}

	// Create go.mod and then remove read permissions.
	goModPath := filepath.Join(dir, "go.mod")
	if err := os.WriteFile(goModPath, []byte("module test"), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}
	if err := os.Chmod(goModPath, 0000); err != nil {
		t.Fatalf("failed to chmod go.mod: %v", err)
	}
	// Restore permissions on cleanup so TempDir can remove it.
	t.Cleanup(func() {
		os.Chmod(goModPath, 0644)
	})

	cb := NewContextBuilder()
	meta, err := cb.Build(dir)
	if err != nil {
		t.Fatalf("Build should not fail for unreadable key file, got: %v", err)
	}

	// README.md should be present.
	if _, ok := meta.KeyFiles["README.md"]; !ok {
		t.Error("README.md should be present in KeyFiles")
	}

	// go.mod should be skipped (unreadable).
	if _, ok := meta.KeyFiles["go.mod"]; ok {
		t.Error("go.mod should be skipped when unreadable, but it was found in KeyFiles")
	}
}

// Test 8: /ask message classification → NeedsProjectContext=true.
// Validates: Requirements 3.2
func TestClassify_AskPrefix(t *testing.T) {
	qc := NewQueryClassifier()

	tests := []struct {
		message string
	}{
		{"/ask what does this project do?"},
		{"/ask"},
		{"/Ask how to build"},
		{"/ASK tell me about the architecture"},
	}

	for _, tc := range tests {
		t.Run(tc.message, func(t *testing.T) {
			result := qc.Classify(tc.message)
			if !result.NeedsProjectContext {
				t.Errorf("Classify(%q): expected NeedsProjectContext=true, got false", tc.message)
			}
		})
	}
}

// Test 9: Search-like messages extract terms.
// Validates: Requirements 5.2
func TestClassify_SearchTermExtraction(t *testing.T) {
	qc := NewQueryClassifier()

	tests := []struct {
		message     string
		description string
	}{
		{
			message:     "where is the main function",
			description: "where is → should extract search terms",
		},
		{
			message:     "find database config",
			description: "find → should extract search terms",
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			result := qc.Classify(tc.message)
			if len(result.SearchTerms) == 0 {
				t.Errorf("Classify(%q): expected non-empty SearchTerms, got empty", tc.message)
			}
		})
	}
}
