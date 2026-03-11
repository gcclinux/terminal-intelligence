package agentic

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// stubAIClient is a configurable stub for AIClient used in project_fixer tests.
type stubAIClient struct {
	response string
	err      error
}

func (s *stubAIClient) IsAvailable() (bool, error) { return true, nil }

func (s *stubAIClient) Generate(prompt string, model string, context []int) (<-chan string, error) {
	if s.err != nil {
		return nil, s.err
	}
	ch := make(chan string, 1)
	ch <- s.response
	close(ch)
	return ch, nil
}

func (s *stubAIClient) ListModels() ([]string, error) { return []string{"stub-model"}, nil }

// ─── helpers ──────────────────────────────────────────────────────────────────

// createFile creates a file at path with the given content, creating parent dirs as needed.
func createFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%s): %v", path, err)
	}
}

// ─── TestFileScannerSkipDirs ──────────────────────────────────────────────────

// TestFileScannerSkipDirs verifies that skip directories are excluded from scan results.
func TestFileScannerSkipDirs(t *testing.T) {
	root := t.TempDir()

	// Create .go files inside each skip directory.
	skipDirNames := []string{".git", "vendor", "node_modules", ".ti", "build"}
	for _, dir := range skipDirNames {
		createFile(t, filepath.Join(root, dir, "file.go"), "package skip")
	}

	// Create a legitimate file at the root level.
	createFile(t, filepath.Join(root, "main.go"), "package main")

	scanner := newFileScanner(root, 500)
	paths, _, err := scanner.scan()
	if err != nil {
		t.Fatalf("scan() error: %v", err)
	}

	// Verify no returned path passes through a skip directory.
	for _, p := range paths {
		rel, _ := filepath.Rel(root, p)
		parts := strings.Split(rel, string(filepath.Separator))
		for _, part := range parts[:len(parts)-1] { // exclude the filename itself
			if skipDirs[part] {
				t.Errorf("scan() returned path inside skip dir %q: %s", part, rel)
			}
		}
	}

	// The legitimate file should be present.
	found := false
	for _, p := range paths {
		if filepath.Base(p) == "main.go" {
			found = true
			break
		}
	}
	if !found {
		t.Error("scan() did not return the legitimate main.go file")
	}
}

// ─── TestFileScannerExtensionFilter ──────────────────────────────────────────

// TestFileScannerExtensionFilter verifies that only allowed extensions are returned.
func TestFileScannerExtensionFilter(t *testing.T) {
	root := t.TempDir()

	// Allowed extensions.
	allowed := []string{
		"a.go", "b.md", "c.sh", "d.bash", "e.ps1",
		"f.py", "g.ts", "h.js", "i.json", "j.yaml",
		"k.yml", "l.toml", "m.txt", "n.html", "o.css",
	}
	for _, name := range allowed {
		createFile(t, filepath.Join(root, name), "content")
	}

	// Disallowed extensions.
	disallowed := []string{"a.exe", "b.bin", "c.png", "d.jpg", "e.zip", "f.so"}
	for _, name := range disallowed {
		createFile(t, filepath.Join(root, name), "content")
	}

	scanner := newFileScanner(root, 500)
	paths, _, err := scanner.scan()
	if err != nil {
		t.Fatalf("scan() error: %v", err)
	}

	// Build a set of returned basenames.
	returned := make(map[string]bool, len(paths))
	for _, p := range paths {
		returned[filepath.Base(p)] = true
	}

	// Every allowed file must be present.
	for _, name := range allowed {
		if !returned[name] {
			t.Errorf("scan() missing allowed file: %s", name)
		}
	}

	// No disallowed file must be present.
	for _, name := range disallowed {
		if returned[name] {
			t.Errorf("scan() returned disallowed file: %s", name)
		}
	}
}

// ─── TestFileScannerTruncation ────────────────────────────────────────────────

// TestFileScannerTruncation verifies the 500-file cap with shortest-path selection.
func TestFileScannerTruncation(t *testing.T) {
	root := t.TempDir()

	// Create 501 .go files: 500 in a deep subdirectory (long paths) and 1 at root (short path).
	deepDir := filepath.Join(root, "a", "b", "c", "d")
	for i := 0; i < 500; i++ {
		createFile(t, filepath.Join(deepDir, fmt.Sprintf("file%04d.go", i)), "package deep")
	}
	// One short-path file at root level.
	createFile(t, filepath.Join(root, "short.go"), "package main")

	scanner := newFileScanner(root, 500)
	paths, truncated, err := scanner.scan()
	if err != nil {
		t.Fatalf("scan() error: %v", err)
	}

	if !truncated {
		t.Error("scan() should have set truncated=true for 501 files")
	}

	if len(paths) != 500 {
		t.Errorf("scan() returned %d paths, want exactly 500", len(paths))
	}

	// The short-path file should be in the result (it has the shortest relative path).
	foundShort := false
	for _, p := range paths {
		if filepath.Base(p) == "short.go" {
			foundShort = true
			break
		}
	}
	if !foundShort {
		t.Error("scan() truncation should keep shortest-path files; short.go was dropped")
	}
}

// ─── TestPathSafetyRejectsOutOfScope ─────────────────────────────────────────

// TestPathSafetyRejectsOutOfScope verifies that symlink and ../ traversal is blocked.
func TestPathSafetyRejectsOutOfScope(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()

	// Create a file outside the project root.
	outsideFile := filepath.Join(outside, "secret.txt")
	createFile(t, outsideFile, "secret content")

	editor := newMultiFileEditor(&stubAIClient{}, "stub", NewFixParser(), root, true /* preview */)

	absRoot, _ := filepath.Abs(root)
	rootPrefix := absRoot + string(filepath.Separator)

	// Test 1: path with ../ traversal.
	traversalPath := filepath.Join(root, "..", filepath.Base(outside), "secret.txt")
	safe, _, _ := editor.checkPathSafety(traversalPath, absRoot, rootPrefix)
	if safe {
		t.Error("checkPathSafety should reject ../ traversal path")
	}

	// Test 2: absolute path outside root.
	safe, _, _ = editor.checkPathSafety(outsideFile, absRoot, rootPrefix)
	if safe {
		t.Error("checkPathSafety should reject absolute path outside project root")
	}

	// Test 3: path inside root should be accepted.
	insideFile := filepath.Join(root, "inside.go")
	createFile(t, insideFile, "package main")
	safe, _, _ = editor.checkPathSafety(insideFile, absRoot, rootPrefix)
	if !safe {
		t.Error("checkPathSafety should accept path inside project root")
	}
}

// ─── TestHallucinatedPathsDiscarded ──────────────────────────────────────────

// TestHallucinatedPathsDiscarded verifies that non-existent AI paths are rejected.
func TestHallucinatedPathsDiscarded(t *testing.T) {
	root := t.TempDir()

	// Create one real file.
	realFile := filepath.Join(root, "real.go")
	createFile(t, realFile, "package main")

	// The stub AI returns one real path and one hallucinated path.
	absReal, _ := filepath.Abs(realFile)
	hallucinatedPath := filepath.Join(root, "does_not_exist.go")
	aiResponse := fmt.Sprintf(`[%q, %q]`, absReal, hallucinatedPath)

	ranker := newRelevanceRanker(&stubAIClient{response: aiResponse}, "stub")
	ranked, hallucinated, err := ranker.rank([]string{absReal}, "fix something", root, 20)
	if err != nil {
		t.Fatalf("rank() error: %v", err)
	}

	// The real file should be ranked.
	if len(ranked) != 1 {
		t.Errorf("rank() returned %d ranked paths, want 1", len(ranked))
	}

	// The hallucinated path should be in the hallucinated list.
	if len(hallucinated) == 0 {
		t.Error("rank() should have recorded the hallucinated path")
	}

	foundHallucinated := false
	for _, h := range hallucinated {
		if strings.Contains(h, "does_not_exist.go") {
			foundHallucinated = true
			break
		}
	}
	if !foundHallucinated {
		t.Errorf("hallucinated list %v does not contain does_not_exist.go", hallucinated)
	}
}

// ─── TestPreviewModeNoWrites ──────────────────────────────────────────────────

// TestPreviewModeNoWrites verifies that no disk writes occur in preview mode.
func TestPreviewModeNoWrites(t *testing.T) {
	root := t.TempDir()

	originalContent := "package main\n\nfunc hello() string {\n\treturn \"hello\"\n}\n"
	filePath := filepath.Join(root, "hello.go")
	createFile(t, filePath, originalContent)

	// Build a valid patch that would change "hello" to "world".
	aiResponse := fmt.Sprintf(`=== FILE: hello.go ===
~~~SEARCH
return "hello"
~~~REPLACE
return "world"
~~~END
`)

	editor := newMultiFileEditor(
		&stubAIClient{response: aiResponse},
		"stub",
		NewFixParser(),
		root,
		true, // preview mode
	)

	absFilePath, _ := filepath.Abs(filePath)
	_, _, _, _, _, err := editor.edit([]string{absFilePath}, "change hello to world")
	if err != nil {
		t.Fatalf("edit() error: %v", err)
	}

	// File content must be unchanged.
	got, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != originalContent {
		t.Errorf("preview mode wrote to disk; content changed:\ngot:  %q\nwant: %q", string(got), originalContent)
	}
}

// ─── TestChangeReportNoModifications ─────────────────────────────────────────

// TestChangeReportNoModifications verifies "No files were modified." message.
func TestChangeReportNoModifications(t *testing.T) {
	cr := &ChangeReport{
		FilesRead:     []string{"a.go", "b.go"},
		FilesModified: nil, // empty
	}

	report := FormatChangeReport(cr)

	if !strings.Contains(report, "No files were modified.") {
		t.Errorf("FormatChangeReport should contain 'No files were modified.' when FilesModified is empty; got:\n%s", report)
	}
}

// ─── TestDuplicateSearchMarkerRejected ───────────────────────────────────────

// TestDuplicateSearchMarkerRejected verifies duplicate ~~~SEARCH detection.
func TestDuplicateSearchMarkerRejected(t *testing.T) {
	// Two blocks with identical search content — should be detected as duplicate.
	duplicatePatch := `~~~SEARCH
func hello() string {
~~~REPLACE
func hello() string { // changed
~~~END
~~~SEARCH
func hello() string {
~~~REPLACE
func hello() string { // changed again
~~~END
`
	if !hasDuplicateSearchMarkers(duplicatePatch) {
		t.Error("hasDuplicateSearchMarkers should return true for duplicate SEARCH blocks")
	}

	// Two blocks with different search content — should NOT be detected as duplicate.
	uniquePatch := `~~~SEARCH
func hello() string {
~~~REPLACE
func hello() string { // changed
~~~END
~~~SEARCH
func world() string {
~~~REPLACE
func world() string { // changed
~~~END
`
	if hasDuplicateSearchMarkers(uniquePatch) {
		t.Error("hasDuplicateSearchMarkers should return false for unique SEARCH blocks")
	}

	// Single block — never a duplicate.
	singlePatch := `~~~SEARCH
func hello() string {
~~~REPLACE
func hello() string { // changed
~~~END
`
	if hasDuplicateSearchMarkers(singlePatch) {
		t.Error("hasDuplicateSearchMarkers should return false for a single SEARCH block")
	}

	// Empty text — never a duplicate.
	if hasDuplicateSearchMarkers("") {
		t.Error("hasDuplicateSearchMarkers should return false for empty text")
	}
}

func TestGitIgnoreMatcher(t *testing.T) {
	root := t.TempDir()
	gitignoreContent := `
node_modules/
*.log
temp
# comment
/dist
`
	err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte(gitignoreContent), 0o644)
	if err != nil {
		t.Fatalf("failed to write .gitignore: %v", err)
	}

	matcher := newGitIgnoreMatcher(root)

	tests := []struct {
		path     string
		expected bool
	}{
		{"node_modules/package.json", true},
		{"src/node_modules/package.json", true},
		{"app.log", true},
		{"logs/error.log", true},
		{"temp", true},
		{"temp/file.txt", true},
		{"dist", true},
		{"src/dist", false}, // matches /dist only at root
		{"main.go", false},
	}

	for _, tt := range tests {
		actual := matcher.matches(tt.path)
		if actual != tt.expected {
			t.Errorf("expected matches(%q) to be %v, got %v", tt.path, tt.expected, actual)
		}
	}
}

func TestExtractExecuteCommand(t *testing.T) {
	response := `
Here is your complete code fix!
=== FILE: main.go ===
~~~SEARCH
func hello()
~~~REPLACE
func helloWorld()
~~~END

And here is the command to verify:
~~~EXECUTE
go test ./...
~~~END
`
	expected := "go test ./..."
	actual := extractExecuteCommand(response)
	if actual != expected {
		t.Errorf("expected %q, got %q", expected, actual)
	}
}
