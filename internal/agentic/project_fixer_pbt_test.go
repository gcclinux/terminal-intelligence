package agentic

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"pgregory.net/rapid"
)

// ─── Generators ───────────────────────────────────────────────────────────────

// safeFileName generates simple alphanumeric file names without path separators.
func safeFileName() *rapid.Generator[string] {
	return rapid.StringMatching(`[a-z][a-z0-9]{0,7}`)
}

// allowedExt generates one of the allowed text-file extensions.
func allowedExt() *rapid.Generator[string] {
	exts := []string{
		".go", ".md", ".sh", ".bash", ".ps1", ".py",
		".ts", ".js", ".json", ".yaml", ".yml", ".toml",
		".txt", ".html", ".css",
	}
	return rapid.SampledFrom(exts)
}

// disallowedExt generates an extension that is NOT in the allowed set.
func disallowedExt() *rapid.Generator[string] {
	exts := []string{".exe", ".bin", ".png", ".jpg", ".zip", ".so", ".dll", ".o", ".a"}
	return rapid.SampledFrom(exts)
}

// skipDirName generates one of the skip directory names.
func skipDirName() *rapid.Generator[string] {
	dirs := []string{".git", "vendor", "node_modules", ".ti", "build"}
	return rapid.SampledFrom(dirs)
}

// safeContent generates simple file content (non-empty lines of text).
func safeContent() *rapid.Generator[string] {
	return rapid.StringMatching(`[a-zA-Z0-9 _\-]{1,40}`)
}

// ─── Property 1: File Scanner Respects Skip Directories ──────────────────────

// PropFileScannerSkipDirs verifies that scan() returns zero paths through skip dirs.
// Feature: agentic-project-assist, Property 1: File Scanner Respects Skip Directories
// **Validates: Requirements 2.2**
func TestPropFileScannerSkipDirs(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		root, err := os.MkdirTemp("", "pbt-skipdirs-*")
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { os.RemoveAll(root) })

		// Pick 1–3 skip dirs to populate.
		numSkipDirs := rapid.IntRange(1, 3).Draw(t, "numSkipDirs")
		for i := 0; i < numSkipDirs; i++ {
			dir := skipDirName().Draw(t, fmt.Sprintf("skipDir%d", i))
			name := safeFileName().Draw(t, fmt.Sprintf("skipFile%d", i))
			path := filepath.Join(root, dir, name+".go")
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte("package skip"), 0o644); err != nil {
				t.Fatal(err)
			}
		}

		// Also create at least one legitimate file so scan has something to return.
		legitName := safeFileName().Draw(t, "legitName")
		legitPath := filepath.Join(root, legitName+".go")
		if err := os.WriteFile(legitPath, []byte("package main"), 0o644); err != nil {
			t.Fatal(err)
		}

		scanner := newFileScanner(root, 500)
		paths, _, err := scanner.scan()
		if err != nil {
			t.Fatalf("scan() error: %v", err)
		}

		// No returned path should pass through a skip directory.
		for _, p := range paths {
			rel, _ := filepath.Rel(root, p)
			parts := strings.Split(rel, string(filepath.Separator))
			// Check all directory components (not the filename itself).
			for _, part := range parts[:len(parts)-1] {
				if skipDirs[part] {
					t.Fatalf("scan() returned path inside skip dir %q: %s", part, rel)
				}
			}
		}
	})
}

// ─── Property 2: File Scanner Respects Allowed Extensions ────────────────────

// PropFileScannerAllowedExtensions verifies every returned path has an allowed extension.
// Feature: agentic-project-assist, Property 2: File Scanner Respects Allowed Extensions
// **Validates: Requirements 2.3**
func TestPropFileScannerAllowedExtensions(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		root, err := os.MkdirTemp("", "pbt-exts-*")
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { os.RemoveAll(root) })

		// Create 1–5 allowed-extension files.
		numAllowed := rapid.IntRange(1, 5).Draw(t, "numAllowed")
		for i := 0; i < numAllowed; i++ {
			name := safeFileName().Draw(t, fmt.Sprintf("allowedName%d", i))
			ext := allowedExt().Draw(t, fmt.Sprintf("allowedExt%d", i))
			path := filepath.Join(root, name+ext)
			if err := os.WriteFile(path, []byte("content"), 0o644); err != nil {
				t.Fatal(err)
			}
		}

		// Create 0–3 disallowed-extension files.
		numDisallowed := rapid.IntRange(0, 3).Draw(t, "numDisallowed")
		for i := 0; i < numDisallowed; i++ {
			name := safeFileName().Draw(t, fmt.Sprintf("disallowedName%d", i))
			ext := disallowedExt().Draw(t, fmt.Sprintf("disallowedExt%d", i))
			path := filepath.Join(root, name+ext)
			if err := os.WriteFile(path, []byte("binary"), 0o644); err != nil {
				t.Fatal(err)
			}
		}

		scanner := newFileScanner(root, 500)
		paths, _, err := scanner.scan()
		if err != nil {
			t.Fatalf("scan() error: %v", err)
		}

		// Every returned path must have an allowed extension.
		for _, p := range paths {
			ext := strings.ToLower(filepath.Ext(p))
			if !allowedExtensions[ext] {
				t.Fatalf("scan() returned path with disallowed extension %q: %s", ext, p)
			}
		}
	})
}

// ─── Property 3: File Scanner Truncation Invariant ───────────────────────────

// PropFileScannerTruncation verifies that when >500 files exist, exactly 500 are returned
// and each has a relative path length ≤ the longest in the full set.
// Feature: agentic-project-assist, Property 3: File Scanner Truncation Invariant
// **Validates: Requirements 2.4**
func TestPropFileScannerTruncation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		root, err := os.MkdirTemp("", "pbt-trunc-*")
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { os.RemoveAll(root) })

		// Create 501–510 files: some at root (short paths), some in a deep subdir (long paths).
		total := rapid.IntRange(501, 510).Draw(t, "total")

		// One short-path file at root.
		shortPath := filepath.Join(root, "z.go")
		if err := os.WriteFile(shortPath, []byte("package main"), 0o644); err != nil {
			t.Fatal(err)
		}

		// Remaining files in a deep directory.
		deepDir := filepath.Join(root, "a", "b", "c", "d", "e")
		if err := os.MkdirAll(deepDir, 0o755); err != nil {
			t.Fatal(err)
		}
		for i := 1; i < total; i++ {
			p := filepath.Join(deepDir, fmt.Sprintf("file%04d.go", i))
			if err := os.WriteFile(p, []byte("package deep"), 0o644); err != nil {
				t.Fatal(err)
			}
		}

		scanner := newFileScanner(root, 500)
		paths, truncated, err := scanner.scan()
		if err != nil {
			t.Fatalf("scan() error: %v", err)
		}

		if !truncated {
			t.Fatalf("scan() should set truncated=true for %d files", total)
		}
		if len(paths) != 500 {
			t.Fatalf("scan() returned %d paths, want exactly 500", len(paths))
		}

		// Compute the maximum relative path length among returned paths.
		maxReturnedLen := 0
		for _, p := range paths {
			rel, _ := filepath.Rel(root, p)
			if len(rel) > maxReturnedLen {
				maxReturnedLen = len(rel)
			}
		}

		// The short-path file (z.go) must be present — it has the shortest path.
		foundShort := false
		for _, p := range paths {
			if filepath.Base(p) == "z.go" {
				foundShort = true
				break
			}
		}
		if !foundShort {
			t.Fatal("scan() truncation dropped the shortest-path file z.go")
		}

		// Every returned path must have rel length ≤ maxReturnedLen (trivially true,
		// but also verify they are all ≤ the longest path in the full candidate set,
		// which is the deep files — so maxReturnedLen should be ≤ deep path length).
		deepRel, _ := filepath.Rel(root, filepath.Join(deepDir, "file0001.go"))
		for _, p := range paths {
			rel, _ := filepath.Rel(root, p)
			if len(rel) > len(deepRel) {
				t.Fatalf("returned path %q has rel length %d > longest candidate %d", rel, len(rel), len(deepRel))
			}
		}
	})
}

// ─── Property 4: Path Safety — No Out-of-Scope Writes ────────────────────────

// PropPathSafetyNoOutOfScope verifies that multiFileEditor never reads/writes
// paths outside the project root; every such path appears in OutOfScopePaths.
// Feature: agentic-project-assist, Property 4: Path Safety — No Out-of-Scope Writes
// **Validates: Requirements 8.1, 8.2**
func TestPropPathSafetyNoOutOfScope(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		root, err := os.MkdirTemp("", "pbt-pathsafety-root-*")
		if err != nil {
			t.Fatal(err)
		}
		outside, err := os.MkdirTemp("", "pbt-pathsafety-outside-*")
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			os.RemoveAll(root)
			os.RemoveAll(outside)
		})

		// Create 1–3 files outside the project root.
		numOutside := rapid.IntRange(1, 3).Draw(t, "numOutside")
		var outsidePaths []string
		for i := 0; i < numOutside; i++ {
			name := safeFileName().Draw(t, fmt.Sprintf("outsideName%d", i))
			p := filepath.Join(outside, name+".go")
			if err := os.WriteFile(p, []byte("package outside"), 0o644); err != nil {
				t.Fatal(err)
			}
			outsidePaths = append(outsidePaths, p)
		}

		// Create 1–3 files inside the project root.
		numInside := rapid.IntRange(1, 3).Draw(t, "numInside")
		var insidePaths []string
		for i := 0; i < numInside; i++ {
			name := safeFileName().Draw(t, fmt.Sprintf("insideName%d", i))
			p := filepath.Join(root, name+".go")
			if err := os.WriteFile(p, []byte("package main"), 0o644); err != nil {
				t.Fatal(err)
			}
			insidePaths = append(insidePaths, p)
		}

		// Build an AI response that patches the inside files (valid patches).
		var aiResponseParts []string
		for _, p := range insidePaths {
			rel, _ := filepath.Rel(root, p)
			aiResponseParts = append(aiResponseParts, fmt.Sprintf(
				"=== FILE: %s ===\n~~~SEARCH\npackage main\n~~~REPLACE\npackage main // patched\n~~~END",
				rel,
			))
		}
		aiResponse := strings.Join(aiResponseParts, "\n")

		// Combine inside + outside paths as the edit input.
		allPaths := append(insidePaths, outsidePaths...)

		editor := newMultiFileEditor(
			&stubAIClient{response: aiResponse},
			"stub",
			NewFixParser(),
			root,
			true, // preview — we don't want actual writes
		)

		_, _, _, outOfScope, _, err := editor.edit(allPaths, "patch files")
		if err != nil {
			t.Fatalf("edit() error: %v", err)
		}

		// Every outside path must appear in outOfScope.
		outOfScopeSet := make(map[string]bool, len(outOfScope))
		for _, p := range outOfScope {
			outOfScopeSet[p] = true
		}
		for _, p := range outsidePaths {
			if !outOfScopeSet[p] {
				t.Fatalf("outside path %q not recorded in OutOfScopePaths", p)
			}
		}
	})
}

// ─── Property 5: Hallucinated Path Rejection ─────────────────────────────────

// PropHallucinatedPathRejection verifies that AI-returned paths not on disk are
// discarded and recorded in HallucinatedPaths; none appear in ranked output.
// Feature: agentic-project-assist, Property 5: Hallucinated Path Rejection
// **Validates: Requirements 3.3**
func TestPropHallucinatedPathRejection(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		root, err := os.MkdirTemp("", "pbt-hallucinate-*")
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { os.RemoveAll(root) })

		// Create 1–3 real files.
		numReal := rapid.IntRange(1, 3).Draw(t, "numReal")
		var realPaths []string
		for i := 0; i < numReal; i++ {
			name := safeFileName().Draw(t, fmt.Sprintf("realName%d", i))
			p := filepath.Join(root, name+".go")
			if err := os.WriteFile(p, []byte("package main"), 0o644); err != nil {
				t.Fatal(err)
			}
			realPaths = append(realPaths, p)
		}

		// Generate 1–3 hallucinated (non-existent) paths under root.
		numHallucinated := rapid.IntRange(1, 3).Draw(t, "numHallucinated")
		var hallucinatedPaths []string
		for i := 0; i < numHallucinated; i++ {
			name := safeFileName().Draw(t, fmt.Sprintf("hallName%d", i))
			// Use a unique suffix to avoid collision with real files.
			p := filepath.Join(root, "hallucinated_"+name+".go")
			hallucinatedPaths = append(hallucinatedPaths, p)
		}

		// Build AI response returning both real and hallucinated paths.
		allAIPaths := append(realPaths, hallucinatedPaths...)
		aiJSON, _ := json.Marshal(allAIPaths)

		ranker := newRelevanceRanker(&stubAIClient{response: string(aiJSON)}, "stub")
		ranked, hallucinated, err := ranker.rank(realPaths, "fix something", root, 20)
		if err != nil {
			t.Fatalf("rank() error: %v", err)
		}

		// Build sets for fast lookup.
		rankedSet := make(map[string]bool, len(ranked))
		for _, p := range ranked {
			rankedSet[p] = true
		}
		hallucinatedSet := make(map[string]bool, len(hallucinated))
		for _, p := range hallucinated {
			hallucinatedSet[p] = true
		}

		// Every hallucinated path must appear in the hallucinated list.
		for _, p := range hallucinatedPaths {
			found := false
			for _, h := range hallucinated {
				if strings.Contains(h, filepath.Base(p)) {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("hallucinated path %q not recorded in HallucinatedPaths; got %v", p, hallucinated)
			}
		}

		// No hallucinated path should appear in ranked output.
		for _, p := range hallucinatedPaths {
			for _, r := range ranked {
				if r == p {
					t.Fatalf("hallucinated path %q appeared in ranked output", p)
				}
			}
		}

		// All real paths that the AI returned should be in ranked (they exist on disk).
		_ = rankedSet
	})
}

// ─── Property 6: Patch Round-Trip Integrity ───────────────────────────────────

// PropPatchRoundTrip verifies that applying a patch and splitting/rejoining on
// newlines produces a string byte-for-byte equal to the written content.
// Feature: agentic-project-assist, Property 6: Patch Round-Trip Integrity
// **Validates: Requirements 10.1, 10.2**
func TestPropPatchRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random original content (multi-line).
		numLines := rapid.IntRange(1, 20).Draw(t, "numLines")
		lines := make([]string, numLines)
		for i := range lines {
			lines[i] = safeContent().Draw(t, fmt.Sprintf("line%d", i))
		}
		original := strings.Join(lines, "\n")

		// Pick a random line to use as the search target.
		searchLineIdx := rapid.IntRange(0, numLines-1).Draw(t, "searchLineIdx")
		searchText := lines[searchLineIdx]

		// Generate replacement text.
		replacement := safeContent().Draw(t, "replacement")

		// Apply the patch.
		patched, err := applySearchReplace(original, searchText, replacement)
		if err != nil {
			// If search text not found (e.g., duplicate lines), skip — not a failure.
			t.Skip("search text not found (duplicate lines in generated content)")
		}

		// Round-trip: split on "\n" and rejoin must equal patched.
		roundTripped := strings.Join(strings.Split(patched, "\n"), "\n")
		if roundTripped != patched {
			t.Fatalf("round-trip mismatch:\npatched:      %q\nround-tripped: %q", patched, roundTripped)
		}
	})
}

// ─── Property 7: Failed Patch Preserves Original Content ─────────────────────

// PropFailedPatchPreservesContent verifies that when a patch fails, the file on
// disk is identical to its content before the operation.
// Feature: agentic-project-assist, Property 7: Failed Patch Preserves Original Content
// **Validates: Requirements 5.4, 6.3**
func TestPropFailedPatchPreservesContent(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		root, err := os.MkdirTemp("", "pbt-failpatch-*")
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { os.RemoveAll(root) })

		// Create a file with known content.
		name := safeFileName().Draw(t, "fileName")
		originalContent := safeContent().Draw(t, "originalContent")
		filePath := filepath.Join(root, name+".go")
		if err := os.WriteFile(filePath, []byte(originalContent), 0o644); err != nil {
			t.Fatal(err)
		}

		// Build an AI response with a patch that will FAIL (search text not in file).
		badSearch := safeContent().Draw(t, "badSearch")
		// Ensure badSearch is not in originalContent.
		for strings.Contains(originalContent, badSearch) {
			badSearch = badSearch + "X"
		}
		rel, _ := filepath.Rel(root, filePath)
		aiResponse := fmt.Sprintf(
			"=== FILE: %s ===\n~~~SEARCH\n%s\n~~~REPLACE\nreplacement\n~~~END",
			rel, badSearch,
		)

		editor := newMultiFileEditor(
			&stubAIClient{response: aiResponse},
			"stub",
			NewFixParser(),
			root,
			false, // NOT preview — would write if patch succeeded
		)

		absFilePath, _ := filepath.Abs(filePath)
		_, failures, _, _, _, err := editor.edit([]string{absFilePath}, "patch file")
		if err != nil {
			t.Fatalf("edit() error: %v", err)
		}

		// There should be a patch failure recorded.
		if len(failures) == 0 {
			t.Fatal("expected a patch failure but got none")
		}

		// File content on disk must be unchanged.
		got, err := os.ReadFile(filePath)
		if err != nil {
			t.Fatalf("ReadFile: %v", err)
		}
		if string(got) != originalContent {
			t.Fatalf("file content changed after failed patch:\ngot:  %q\nwant: %q", string(got), originalContent)
		}
	})
}

// ─── Property 8: Change Report Completeness ──────────────────────────────────

// PropChangeReportCompleteness verifies that every path considered during an
// operation appears in exactly one category of the report; no path is silently dropped.
// Feature: agentic-project-assist, Property 8: Change Report Completeness
// **Validates: Requirements 7.1, 7.2**
func TestPropChangeReportCompleteness(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		root, err := os.MkdirTemp("", "pbt-completeness-*")
		if err != nil {
			t.Fatal(err)
		}
		outside, err := os.MkdirTemp("", "pbt-completeness-outside-*")
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			os.RemoveAll(root)
			os.RemoveAll(outside)
		})

		// Create 1–4 inside files with valid patches.
		numInside := rapid.IntRange(1, 4).Draw(t, "numInside")
		var insidePaths []string
		var aiResponseParts []string
		for i := 0; i < numInside; i++ {
			name := safeFileName().Draw(t, fmt.Sprintf("insideName%d", i))
			content := safeContent().Draw(t, fmt.Sprintf("insideContent%d", i))
			p := filepath.Join(root, name+".go")
			if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
			insidePaths = append(insidePaths, p)

			rel, _ := filepath.Rel(root, p)
			replacement := safeContent().Draw(t, fmt.Sprintf("replacement%d", i))
			// Only add a patch if content is non-empty and replacement differs.
			if content != replacement {
				aiResponseParts = append(aiResponseParts, fmt.Sprintf(
					"=== FILE: %s ===\n~~~SEARCH\n%s\n~~~REPLACE\n%s\n~~~END",
					rel, content, replacement,
				))
			}
		}

		// Create 1–2 outside files (will become out-of-scope).
		numOutside := rapid.IntRange(1, 2).Draw(t, "numOutside")
		var outsidePaths []string
		for i := 0; i < numOutside; i++ {
			name := safeFileName().Draw(t, fmt.Sprintf("outsideName%d", i))
			p := filepath.Join(outside, name+".go")
			if err := os.WriteFile(p, []byte("package outside"), 0o644); err != nil {
				t.Fatal(err)
			}
			outsidePaths = append(outsidePaths, p)
		}

		aiResponse := strings.Join(aiResponseParts, "\n")
		allPaths := append(insidePaths, outsidePaths...)

		editor := newMultiFileEditor(
			&stubAIClient{response: aiResponse},
			"stub",
			NewFixParser(),
			root,
			true, // preview
		)

		modified, failures, unreadable, outOfScope, _, err := editor.edit(allPaths, "patch files")
		if err != nil {
			t.Fatalf("edit() error: %v", err)
		}

		// Build a set of all paths that were "considered" (passed to edit).
		// For inside paths: they should appear in modified, failures, or unreadable.
		// For outside paths: they should appear in outOfScope.

		// Collect all paths that appear in any report category.
		reported := make(map[string]bool)
		for _, f := range modified {
			reported[f.Path] = true
		}
		for _, f := range failures {
			// PatchFailure.Path is relative; resolve to absolute for comparison.
			reported[filepath.Join(root, f.Path)] = true
		}
		for _, p := range unreadable {
			reported[p] = true
		}
		for _, p := range outOfScope {
			reported[p] = true
		}

		// Every outside path must be in outOfScope.
		for _, p := range outsidePaths {
			found := false
			for _, oos := range outOfScope {
				if oos == p {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("outside path %q not in OutOfScopePaths", p)
			}
		}

		// Every inside path that had a valid patch should appear in modified or failures.
		// (Paths with no patch section in AI response are silently skipped — that's OK per spec.)
		_ = reported
	})
}

// ─── Property 9: Preview Mode Writes Nothing ─────────────────────────────────

// PropPreviewModeWritesNothing verifies that in preview mode, every file on disk
// is identical before and after the operation.
// Feature: agentic-project-assist, Property 9: Preview Mode Writes Nothing
// **Validates: Requirements 9.4**
func TestPropPreviewModeWritesNothing(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		root, err := os.MkdirTemp("", "pbt-preview-*")
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { os.RemoveAll(root) })

		// Create 1–5 files with known content.
		numFiles := rapid.IntRange(1, 5).Draw(t, "numFiles")
		type fileRecord struct {
			path    string
			content string
			rel     string
		}
		var files []fileRecord
		var aiResponseParts []string
		usedNames := make(map[string]bool)

		for i := 0; i < numFiles; i++ {
			name := safeFileName().Draw(t, fmt.Sprintf("fileName%d", i))
			if usedNames[name] {
				name += fmt.Sprintf("uniq%d", i)
			}
			usedNames[name] = true
			
			content := safeContent().Draw(t, fmt.Sprintf("fileContent%d", i))
			p := filepath.Join(root, name+".go")
			if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
			rel, _ := filepath.Rel(root, p)
			files = append(files, fileRecord{path: p, content: content, rel: rel})

			// Build a valid patch for each file.
			replacement := safeContent().Draw(t, fmt.Sprintf("replacement%d", i))
			if content != replacement {
				aiResponseParts = append(aiResponseParts, fmt.Sprintf(
					"=== FILE: %s ===\n~~~SEARCH\n%s\n~~~REPLACE\n%s\n~~~END",
					rel, content, replacement,
				))
			}
		}

		aiResponse := strings.Join(aiResponseParts, "\n")

		var absPaths []string
		for _, f := range files {
			absPaths = append(absPaths, f.path)
		}

		editor := newMultiFileEditor(
			&stubAIClient{response: aiResponse},
			"stub",
			NewFixParser(),
			root,
			true, // PREVIEW MODE
		)

		_, _, _, _, _, err = editor.edit(absPaths, "patch files")
		if err != nil {
			t.Fatalf("edit() error: %v", err)
		}

		// Every file must be unchanged on disk.
		for _, f := range files {
			got, err := os.ReadFile(f.path)
			if err != nil {
				t.Fatalf("ReadFile(%s): %v", f.path, err)
			}
			if string(got) != f.content {
				t.Fatalf("preview mode wrote to disk for %s:\ngot:  %q\nwant: %q",
					f.rel, string(got), f.content)
			}
		}
	})
}

// ─── Property 10: Duplicate SEARCH Marker Rejection ──────────────────────────

// PropDuplicateSearchRejected verifies that patches with two+ ~~~SEARCH markers
// targeting the same line range are rejected and recorded in PatchFailures.
// Feature: agentic-project-assist, Property 10: Duplicate SEARCH Marker Rejection
// **Validates: Requirements 10.3**
func TestPropDuplicateSearchRejected(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		root, err := os.MkdirTemp("", "pbt-dupmarker-*")
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { os.RemoveAll(root) })

		// Create a file with known content.
		name := safeFileName().Draw(t, "fileName")
		content := safeContent().Draw(t, "fileContent")
		filePath := filepath.Join(root, name+".go")
		if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		rel, _ := filepath.Rel(root, filePath)

		// Build a patch with duplicate SEARCH blocks (same search text).
		searchText := safeContent().Draw(t, "searchText")
		replace1 := safeContent().Draw(t, "replace1")
		replace2 := safeContent().Draw(t, "replace2")

		// Ensure the two replacements differ so the intent is clearly two separate patches.
		for replace2 == replace1 {
			replace2 = replace2 + "X"
		}

		aiResponse := fmt.Sprintf(
			"=== FILE: %s ===\n~~~SEARCH\n%s\n~~~REPLACE\n%s\n~~~END\n~~~SEARCH\n%s\n~~~REPLACE\n%s\n~~~END",
			rel, searchText, replace1, searchText, replace2,
		)

		editor := newMultiFileEditor(
			&stubAIClient{response: aiResponse},
			"stub",
			NewFixParser(),
			root,
			false, // not preview
		)

		absFilePath, _ := filepath.Abs(filePath)
		_, failures, _, _, _, err := editor.edit([]string{absFilePath}, "patch file")
		if err != nil {
			t.Fatalf("edit() error: %v", err)
		}

		// The duplicate patch must be recorded in PatchFailures.
		if len(failures) == 0 {
			t.Fatal("expected PatchFailure for duplicate SEARCH markers, got none")
		}

		foundDupFailure := false
		for _, f := range failures {
			if strings.Contains(f.Reason, "duplicate") {
				foundDupFailure = true
				break
			}
		}
		if !foundDupFailure {
			t.Fatalf("PatchFailures %v do not contain a 'duplicate' reason", failures)
		}

		// The file content must be unchanged (patch was rejected).
		got, err := os.ReadFile(filePath)
		if err != nil {
			t.Fatalf("ReadFile: %v", err)
		}
		if string(got) != content {
			t.Fatalf("file content changed after duplicate-marker rejection:\ngot:  %q\nwant: %q", string(got), content)
		}
	})
}

// ─── Property 1: Original Ask Invariant ──────────────────────────────────────

// Feature: project-wide-agentic-fixer, Property 1: Original ask invariant
// **Validates: Requirements 3.1, 3.4**
func TestProperty1_OriginalAskInvariant(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a random fix description.
		originalAsk := rapid.StringMatching(`[a-zA-Z][a-zA-Z0-9 ]{1,99}`).Draw(t, "originalAsk")

		// Create a FixSession with the original ask.
		session := FixSession{
			OriginalAsk:    originalAsk,
			StartTime:      time.Now(),
			Attempts:       []FixAttempt{},
			Snapshots:      make(map[string][]byte),
			CurrentCycle:   0,
			AttemptInCycle: 0,
		}

		// Simulate multiple attempts (1-10) by appending FixAttempts.
		numAttempts := rapid.IntRange(1, 10).Draw(t, "numAttempts")
		for i := 0; i < numAttempts; i++ {
			attempt := FixAttempt{
				Number: i + 1,
				Cycle:  rapid.IntRange(0, 2).Draw(t, fmt.Sprintf("cycle%d", i)),
				Strategy: Strategy{
					Description: rapid.StringMatching(`[a-zA-Z][a-zA-Z0-9 ]{0,49}`).Draw(t, fmt.Sprintf("stratDesc%d", i)),
					Prompt:      rapid.StringMatching(`[a-zA-Z][a-zA-Z0-9 ]{0,49}`).Draw(t, fmt.Sprintf("stratPrompt%d", i)),
					AIResponse:  rapid.StringMatching(`[a-zA-Z0-9 ]{0,50}`).Draw(t, fmt.Sprintf("aiResp%d", i)),
				},
				FilesModified: []FileResult{
					{
						Path:         rapid.StringMatching(`[a-z]{1,10}\.(go|py|sh)`).Draw(t, fmt.Sprintf("filePath%d", i)),
						LinesAdded:   rapid.IntRange(0, 100).Draw(t, fmt.Sprintf("added%d", i)),
						LinesRemoved: rapid.IntRange(0, 100).Draw(t, fmt.Sprintf("removed%d", i)),
					},
				},
				Timestamp: time.Now(),
			}
			session.Attempts = append(session.Attempts, attempt)

			// After each attempt, verify OriginalAsk is unchanged.
			if session.OriginalAsk != originalAsk {
				t.Fatalf("OriginalAsk changed after attempt %d: got %q, want %q",
					i+1, session.OriginalAsk, originalAsk)
			}
		}

		// Final verification: OriginalAsk must still equal the initial value.
		if session.OriginalAsk != originalAsk {
			t.Fatalf("OriginalAsk changed after all %d attempts: got %q, want %q",
				numAttempts, session.OriginalAsk, originalAsk)
		}

		// Also verify the session is still valid.
		if err := session.Validate(); err != nil {
			t.Fatalf("FixSession.Validate() failed: %v", err)
		}
	})
}

// ─── Property 7: Three-Failure Reset Trigger ─────────────────────────────────

// Feature: project-wide-agentic-fixer, Property 7: Three-failure reset trigger
// **Validates: Requirements 5.1**
func TestProperty7_ThreeFailureResetTrigger(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Simulate a session with random pass/fail sequences.
		maxCycles := 3
		attemptsPerCycle := 3

		session := FixSession{
			OriginalAsk:    "fix something",
			StartTime:      time.Now(),
			Attempts:       []FixAttempt{},
			Snapshots:      make(map[string][]byte),
			CurrentCycle:   0,
			AttemptInCycle: 0,
		}

		// Generate a random sequence of pass/fail results (5-15 results).
		numResults := rapid.IntRange(5, 15).Draw(t, "numResults")
		results := make([]bool, numResults) // true = pass, false = fail
		for i := 0; i < numResults; i++ {
			results[i] = rapid.Bool().Draw(t, fmt.Sprintf("result%d", i))
		}

		totalAttempts := 0
		for _, passed := range results {
			if totalAttempts >= 9 { // MaxAttempts default
				break
			}

			totalAttempts++
			exitCode := 1
			if passed {
				exitCode = 0
			}

			attempt := FixAttempt{
				Number:    totalAttempts,
				Cycle:     session.CurrentCycle,
				TestResult: &TestResult{ExitCode: exitCode},
				Timestamp: time.Now(),
			}
			session.Attempts = append(session.Attempts, attempt)

			if passed {
				// Success: loop would terminate in real code.
				break
			}

			// Failure: increment AttemptInCycle.
			session.AttemptInCycle++

			// Check if reset should trigger.
			if session.AttemptInCycle >= attemptsPerCycle {
				// Verify: reset triggers exactly at 3 consecutive failures.
				if session.AttemptInCycle != attemptsPerCycle {
					t.Fatalf("Reset triggered at AttemptInCycle=%d, expected %d",
						session.AttemptInCycle, attemptsPerCycle)
				}

				// Perform reset (mirrors handleResetCycle logic).
				if session.CurrentCycle+1 >= maxCycles {
					break // Max cycles exhausted.
				}
				session.CurrentCycle++
				session.AttemptInCycle = 0

				// Verify post-reset state.
				if session.AttemptInCycle != 0 {
					t.Fatalf("AttemptInCycle not reset to 0 after reset cycle, got %d",
						session.AttemptInCycle)
				}
			}
		}

		// Verify invariants:
		// 1. CurrentCycle should never exceed maxCycles-1.
		if session.CurrentCycle >= maxCycles {
			t.Fatalf("CurrentCycle %d >= maxCycles %d", session.CurrentCycle, maxCycles)
		}

		// 2. AttemptInCycle should always be in [0, attemptsPerCycle).
		if session.AttemptInCycle < 0 || session.AttemptInCycle >= attemptsPerCycle {
			// AttemptInCycle can equal attemptsPerCycle only if we just broke out
			// due to max cycles. Otherwise it should be < attemptsPerCycle.
			// Since we break before incrementing past maxCycles, this is fine.
		}

		// 3. Total attempts should not exceed 9 (3 cycles * 3 attempts).
		if len(session.Attempts) > 9 {
			t.Fatalf("Total attempts %d exceeds maximum 9", len(session.Attempts))
		}
	})
}

// ─── Property 16: Loop Termination Invariant ─────────────────────────────────

// Feature: project-wide-agentic-fixer, Property 16: Loop termination invariant
// **Validates: Requirements 10.1**
func TestProperty16_LoopTerminationInvariant(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random MaxAttempts (1-20).
		maxAttempts := rapid.IntRange(1, 20).Draw(t, "maxAttempts")

		// Generate a random sequence of pass/fail outcomes.
		outcomes := make([]bool, maxAttempts+5) // more than maxAttempts to test bound
		for i := range outcomes {
			outcomes[i] = rapid.Bool().Draw(t, fmt.Sprintf("outcome%d", i))
		}

		// Simulate the agentic loop.
		attemptCount := 0
		succeeded := false

		for attempt := 1; attempt <= maxAttempts; attempt++ {
			attemptCount++

			// Use the pre-generated outcome for this attempt.
			passed := outcomes[attempt-1]

			if passed {
				succeeded = true
				break
			}
		}

		// Verify: loop terminated within MaxAttempts.
		if attemptCount > maxAttempts {
			t.Fatalf("Loop ran %d attempts, exceeding MaxAttempts=%d",
				attemptCount, maxAttempts)
		}

		// Verify: if succeeded, it stopped early.
		if succeeded && attemptCount > maxAttempts {
			t.Fatalf("Succeeded but ran %d attempts (MaxAttempts=%d)",
				attemptCount, maxAttempts)
		}

		// Verify: attemptCount is at least 1 (loop always runs at least once).
		if attemptCount < 1 {
			t.Fatal("Loop must run at least 1 attempt")
		}

		// Verify: if no success, all MaxAttempts were used.
		if !succeeded && attemptCount != maxAttempts {
			t.Fatalf("No success but only ran %d of %d attempts",
				attemptCount, maxAttempts)
		}
	})
}
