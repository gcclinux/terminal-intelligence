package dirtracker

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"pgregory.net/rapid"
)

var workspaceRoot = filepath.Join("/", "workspace")

// safeDirName generates simple alphanumeric directory names that won't break parsing.
func safeDirName() *rapid.Generator[string] {
	return rapid.StringMatching("[a-z][a-z0-9]{0,7}")
}

// TestProperty1_CdUpdatesEffectiveDirectory verifies that cd commands update
// the effective directory via sequential filepath.Join application.
// **Validates: Requirements 1.1, 1.3, 2.2, 5.1**
func TestProperty1_CdUpdatesEffectiveDirectory(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a random sequence of relative directory names for cd commands
		numCds := rapid.IntRange(1, 10).Draw(t, "numCds")
		dirs := make([]string, numCds)
		for i := range dirs {
			dirs[i] = safeDirName().Draw(t, fmt.Sprintf("dir%d", i))
		}

		// Build bash content with cd commands
		var lines []string
		for _, d := range dirs {
			lines = append(lines, "cd "+d)
		}
		bashContent := strings.Join(lines, "\n")

		// Also intersperse some non-bash blocks
		numNonBash := rapid.IntRange(0, 3).Draw(t, "numNonBash")
		blocks := []CodeBlockInfo{
			{Language: "bash", Content: bashContent},
		}
		for i := 0; i < numNonBash; i++ {
			blocks = append(blocks, CodeBlockInfo{Language: "go", Content: "package main"})
		}

		dt := New(workspaceRoot)
		mappings := dt.ComputeMappings(blocks)

		// Compute expected effective directory by applying filepath.Join sequentially
		expected := workspaceRoot
		for _, d := range dirs {
			expected = filepath.Join(expected, d)
		}

		// The first block (bash) should have the final effective dir after all cds
		if mappings[0] != expected {
			t.Fatalf("block 0: got %q, want %q", mappings[0], expected)
		}

		// All subsequent non-bash blocks should inherit the same directory
		for i := 1; i < len(mappings); i++ {
			if mappings[i] != expected {
				t.Fatalf("block %d: got %q, want %q (should inherit)", i, mappings[i], expected)
			}
		}
	})
}

// TestProperty2_MkdirParsedCorrectly verifies that mkdir commands are parsed
// with correct Path and MkdirP flag.
// **Validates: Requirements 1.2, 1.4**
func TestProperty2_MkdirParsedCorrectly(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		dirname := safeDirName().Draw(t, "dirname")
		useP := rapid.Bool().Draw(t, "useP")

		var content string
		if useP {
			content = "mkdir -p " + dirname
		} else {
			content = "mkdir " + dirname
		}

		dt := New(workspaceRoot)
		cmds := dt.ParseBashCommands(content)

		if len(cmds) != 1 {
			t.Fatalf("expected 1 command, got %d: %+v", len(cmds), cmds)
		}
		if cmds[0].Type != "mkdir" {
			t.Fatalf("expected type mkdir, got %q", cmds[0].Type)
		}
		if cmds[0].Path != dirname {
			t.Fatalf("expected path %q, got %q", dirname, cmds[0].Path)
		}
		if cmds[0].MkdirP != useP {
			t.Fatalf("expected MkdirP=%v, got %v", useP, cmds[0].MkdirP)
		}
	})
}

// TestProperty3_CompoundMkdirCdProcessedLeftToRight verifies that
// "mkdir X && cd X" results in effective dir including X.
// **Validates: Requirements 1.5**
func TestProperty3_CompoundMkdirCdProcessedLeftToRight(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		dirname := safeDirName().Draw(t, "dirname")

		content := fmt.Sprintf("mkdir %s && cd %s", dirname, dirname)
		blocks := []CodeBlockInfo{
			{Language: "bash", Content: content},
		}

		dt := New(workspaceRoot)
		mappings := dt.ComputeMappings(blocks)

		expected := filepath.Join(workspaceRoot, dirname)
		if mappings[0] != expected {
			t.Fatalf("got %q, want %q", mappings[0], expected)
		}
	})
}

// TestProperty4_CdDotDotClampedAtRoot verifies that excessive cd .. never
// goes above workspace root.
// **Validates: Requirements 1.6**
func TestProperty4_CdDotDotClampedAtRoot(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// First cd into some subdirectories
		depth := rapid.IntRange(0, 5).Draw(t, "depth")
		var cdInLines []string
		for i := 0; i < depth; i++ {
			d := safeDirName().Draw(t, fmt.Sprintf("subdir%d", i))
			cdInLines = append(cdInLines, "cd "+d)
		}

		// Then cd .. more times than we went in
		excessUps := rapid.IntRange(1, 20).Draw(t, "excessUps")
		totalUps := depth + excessUps
		var cdOutLines []string
		for i := 0; i < totalUps; i++ {
			cdOutLines = append(cdOutLines, "cd ..")
		}

		allLines := append(cdInLines, cdOutLines...)
		content := strings.Join(allLines, "\n")

		blocks := []CodeBlockInfo{
			{Language: "bash", Content: content},
		}

		dt := New(workspaceRoot)
		mappings := dt.ComputeMappings(blocks)

		if !strings.HasPrefix(mappings[0], workspaceRoot) {
			t.Fatalf("effective dir %q is not under workspace root %q", mappings[0], workspaceRoot)
		}
	})
}

// TestProperty5_CommentLinesIgnored verifies that commented mkdir/cd lines
// cause no directory change.
// **Validates: Requirements 1.7**
func TestProperty5_CommentLinesIgnored(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		numLines := rapid.IntRange(1, 10).Draw(t, "numLines")
		var lines []string
		for i := 0; i < numLines; i++ {
			dirname := safeDirName().Draw(t, fmt.Sprintf("dir%d", i))
			cmdType := rapid.SampledFrom([]string{"mkdir", "cd"}).Draw(t, fmt.Sprintf("cmd%d", i))
			lines = append(lines, "# "+cmdType+" "+dirname)
		}
		content := strings.Join(lines, "\n")

		blocks := []CodeBlockInfo{
			{Language: "bash", Content: content},
		}

		dt := New(workspaceRoot)
		mappings := dt.ComputeMappings(blocks)

		if mappings[0] != workspaceRoot {
			t.Fatalf("expected %q (no change), got %q", workspaceRoot, mappings[0])
		}
	})
}

// TestProperty6_InitializationToWorkspaceRoot verifies that blocks with no
// directory commands all map to workspace root.
// **Validates: Requirements 2.1, 2.5**
func TestProperty6_InitializationToWorkspaceRoot(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		numBlocks := rapid.IntRange(1, 10).Draw(t, "numBlocks")
		blocks := make([]CodeBlockInfo, numBlocks)
		for i := range blocks {
			// Use non-bash languages or bash with non-dir commands
			lang := rapid.SampledFrom([]string{"bash", "go", "python", ""}).Draw(t, fmt.Sprintf("lang%d", i))
			// Content has no mkdir or cd commands
			content := rapid.SampledFrom([]string{
				"echo hello",
				"ls -la",
				"cat file.txt",
				"go build .",
				"python main.py",
				"npm install",
			}).Draw(t, fmt.Sprintf("content%d", i))
			blocks[i] = CodeBlockInfo{Language: lang, Content: content}
		}

		dt := New(workspaceRoot)
		mappings := dt.ComputeMappings(blocks)

		for i, m := range mappings {
			if m != workspaceRoot {
				t.Fatalf("block %d: got %q, want %q", i, m, workspaceRoot)
			}
		}
	})
}

// TestProperty7_NonBashBlocksInheritAndMappingLengthPreserved verifies that
// non-bash blocks inherit the current directory and mapping length equals input length.
// **Validates: Requirements 2.3, 2.4**
func TestProperty7_NonBashBlocksInheritAndMappingLengthPreserved(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		numBlocks := rapid.IntRange(1, 15).Draw(t, "numBlocks")
		blocks := make([]CodeBlockInfo, numBlocks)

		for i := range blocks {
			isBash := rapid.Bool().Draw(t, fmt.Sprintf("isBash%d", i))
			if isBash {
				// Bash block with a random cd command
				dirname := safeDirName().Draw(t, fmt.Sprintf("dir%d", i))
				blocks[i] = CodeBlockInfo{Language: "bash", Content: "cd " + dirname}
			} else {
				lang := rapid.SampledFrom([]string{"go", "python", "javascript", ""}).Draw(t, fmt.Sprintf("lang%d", i))
				blocks[i] = CodeBlockInfo{Language: lang, Content: "some code"}
			}
		}

		dt := New(workspaceRoot)
		mappings := dt.ComputeMappings(blocks)

		// Mapping length must equal input length
		if len(mappings) != len(blocks) {
			t.Fatalf("mapping length %d != block count %d", len(mappings), len(blocks))
		}

		// Verify non-bash blocks inherit from the most recent effective dir
		currentDir := workspaceRoot
		for i, block := range blocks {
			if isBashBlock(block.Language) {
				// Recompute what the effective dir should be after this bash block
				cmds := dt.ParseBashCommands(block.Content)
				for _, cmd := range cmds {
					if cmd.Type == "cd" {
						currentDir = filepath.Join(currentDir, cmd.Path)
						currentDir = filepath.Clean(currentDir)
						if !strings.HasPrefix(currentDir, workspaceRoot) {
							currentDir = workspaceRoot
						}
					}
				}
			}
			if mappings[i] != currentDir {
				t.Fatalf("block %d: got %q, want %q", i, mappings[i], currentDir)
			}
		}
	})
}

// TestProperty8_NonRelativePathsIgnored verifies that cd with absolute, ~, $HOME,
// or $VAR paths causes no directory change.
// **Validates: Requirements 5.2, 5.3, 5.5**
func TestProperty8_NonRelativePathsIgnored(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a non-relative path
		pathType := rapid.SampledFrom([]string{"absolute", "tilde", "home", "var"}).Draw(t, "pathType")
		dirname := safeDirName().Draw(t, "dirname")

		var target string
		switch pathType {
		case "absolute":
			target = "/" + dirname
		case "tilde":
			target = "~"
		case "home":
			target = "$HOME"
		case "var":
			target = "$" + strings.ToUpper(dirname)
		}

		content := "cd " + target
		blocks := []CodeBlockInfo{
			{Language: "bash", Content: content},
		}

		dt := New(workspaceRoot)
		mappings := dt.ComputeMappings(blocks)

		if mappings[0] != workspaceRoot {
			t.Fatalf("cd %q should be ignored, got %q, want %q", target, mappings[0], workspaceRoot)
		}
	})
}

// TestProperty9_InputBlocksNotModified verifies that ComputeMappings does not
// modify the input blocks.
// **Validates: Requirements 6.2**
func TestProperty9_InputBlocksNotModified(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		numBlocks := rapid.IntRange(1, 10).Draw(t, "numBlocks")
		blocks := make([]CodeBlockInfo, numBlocks)

		// Store original values for comparison
		origLanguages := make([]string, numBlocks)
		origContents := make([]string, numBlocks)

		for i := range blocks {
			lang := rapid.SampledFrom([]string{"bash", "go", "python", "sh", ""}).Draw(t, fmt.Sprintf("lang%d", i))
			dirname := safeDirName().Draw(t, fmt.Sprintf("dir%d", i))
			content := rapid.SampledFrom([]string{
				"cd " + dirname,
				"mkdir " + dirname,
				"mkdir " + dirname + " && cd " + dirname,
				"echo hello",
				"# cd " + dirname,
			}).Draw(t, fmt.Sprintf("content%d", i))

			blocks[i] = CodeBlockInfo{Language: lang, Content: content}
			origLanguages[i] = lang
			origContents[i] = content
		}

		dt := New(workspaceRoot)
		_ = dt.ComputeMappings(blocks)

		// Verify blocks are unchanged
		for i := range blocks {
			if blocks[i].Language != origLanguages[i] {
				t.Fatalf("block %d Language changed: %q -> %q", i, origLanguages[i], blocks[i].Language)
			}
			if blocks[i].Content != origContents[i] {
				t.Fatalf("block %d Content changed: %q -> %q", i, origContents[i], blocks[i].Content)
			}
		}
	})
}
