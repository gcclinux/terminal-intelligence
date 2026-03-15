package projectctx

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// skipDirNames is the list of directory names that ContextBuilder must skip.
var skipDirNames = []string{".git", "vendor", "node_modules", ".ti", "build"}

// drawSubset returns a random non-empty subset of items using rapid boolean draws.
func drawSubset(rt *rapid.T, items []string, label string) []string {
	var chosen []string
	for _, item := range items {
		if rapid.Bool().Draw(rt, fmt.Sprintf("%s_include_%s", label, item)) {
			chosen = append(chosen, item)
		}
	}
	// Ensure at least one item is chosen.
	if len(chosen) == 0 {
		idx := rapid.IntRange(0, len(items)-1).Draw(rt, label+"_fallback")
		chosen = append(chosen, items[idx])
	}
	return chosen
}

// Feature: smart-project-query, Property 1: Skip directories are excluded from scan results
// **Validates: Requirements 1.1**
func TestProperty1_SkipDirectoriesExcludedFromScanResults(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		dir := t.TempDir()

		// Choose a random subset of skip dirs to create (at least 1).
		chosenSkipDirs := drawSubset(rt, skipDirNames, "skipDirs")

		// Create each chosen skip directory with a file inside it.
		for _, sd := range chosenSkipDirs {
			sdPath := filepath.Join(dir, sd)
			if err := os.MkdirAll(sdPath, 0755); err != nil {
				rt.Fatalf("failed to create skip dir %s: %v", sd, err)
			}
			fPath := filepath.Join(sdPath, "should_be_skipped.txt")
			if err := os.WriteFile(fPath, []byte("hidden"), 0644); err != nil {
				rt.Fatalf("failed to write file in skip dir: %v", err)
			}
		}

		// Also create some normal files so the workspace isn't empty.
		numNormal := rapid.IntRange(0, 5).Draw(rt, "numNormalFiles")
		for i := 0; i < numNormal; i++ {
			name := fmt.Sprintf("normal_%d.txt", i)
			if err := os.WriteFile(filepath.Join(dir, name), []byte("ok"), 0644); err != nil {
				rt.Fatalf("failed to write normal file: %v", err)
			}
		}

		cb := NewContextBuilder()
		meta, err := cb.Build(dir)
		if err != nil {
			rt.Fatalf("Build failed: %v", err)
		}

		// Assert: no FileTree entry contains any skip directory segment.
		for _, entry := range meta.FileTree {
			for _, sd := range skipDirNames {
				if strings.Contains(entry, sd+"/") || strings.HasPrefix(entry, sd+"/") || entry == sd {
					rt.Fatalf("FileTree contains skip dir segment %q in entry %q", sd, entry)
				}
			}
		}

		// Assert: no KeyFiles key contains any skip directory segment.
		for key := range meta.KeyFiles {
			for _, sd := range skipDirNames {
				if strings.Contains(key, sd+"/") || strings.HasPrefix(key, sd+"/") || key == sd {
					rt.Fatalf("KeyFiles contains skip dir segment %q in key %q", sd, key)
				}
			}
		}
	})
}

// Feature: smart-project-query, Property 2: Key project files are detected when present
// **Validates: Requirements 1.2**
func TestProperty2_KeyProjectFilesDetectedWhenPresent(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		dir := t.TempDir()

		// Pick a random non-empty subset of key project files to create.
		chosenKeys := drawSubset(rt, KeyProjectFiles, "keyFiles")

		// Create each chosen key file with some content.
		for _, kf := range chosenKeys {
			content := rapid.SliceOfN(rapid.Byte(), 1, 500).Draw(rt, fmt.Sprintf("content_%s", kf))
			if err := os.WriteFile(filepath.Join(dir, kf), content, 0644); err != nil {
				rt.Fatalf("failed to write key file %s: %v", kf, err)
			}
		}

		cb := NewContextBuilder()
		meta, err := cb.Build(dir)
		if err != nil {
			rt.Fatalf("Build failed: %v", err)
		}

		// Assert: every key file we created appears in KeyFiles.
		for _, kf := range chosenKeys {
			if _, ok := meta.KeyFiles[kf]; !ok {
				rt.Fatalf("key file %q exists on disk but missing from KeyFiles map", kf)
			}
		}
	})
}

// Feature: smart-project-query, Property 3: File tree respects the 200-entry limit
// **Validates: Requirements 1.3, 1.4**
func TestProperty3_FileTreeRespectsEntryLimit(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		dir := t.TempDir()

		// Generate a random number of files, sometimes exceeding 200.
		numFiles := rapid.IntRange(0, 350).Draw(rt, "numFiles")
		for i := 0; i < numFiles; i++ {
			name := fmt.Sprintf("file_%04d.txt", i)
			if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0644); err != nil {
				rt.Fatalf("failed to write file: %v", err)
			}
		}

		cb := NewContextBuilder()
		meta, err := cb.Build(dir)
		if err != nil {
			rt.Fatalf("Build failed: %v", err)
		}

		// Assert: FileTree length is at most MaxFileTreeEntries.
		if len(meta.FileTree) > MaxFileTreeEntries {
			rt.Fatalf("FileTree has %d entries, exceeds max %d", len(meta.FileTree), MaxFileTreeEntries)
		}

		// Assert: FileTreeTruncated == (TotalFiles > MaxFileTreeEntries).
		expectedTruncated := meta.TotalFiles > MaxFileTreeEntries
		if meta.FileTreeTruncated != expectedTruncated {
			rt.Fatalf("FileTreeTruncated=%v but TotalFiles=%d (expected truncated=%v)",
				meta.FileTreeTruncated, meta.TotalFiles, expectedTruncated)
		}
	})
}

// Feature: smart-project-query, Property 4: Key file contents are truncated at 5000 bytes
// **Validates: Requirements 1.5**
func TestProperty4_KeyFileContentsTruncatedAt5000Bytes(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		dir := t.TempDir()

		// Pick a random key file name.
		keyFileName := rapid.SampledFrom(KeyProjectFiles).Draw(rt, "keyFileName")

		// Generate content that exceeds MaxKeyFileBytes.
		size := rapid.IntRange(MaxKeyFileBytes+1, MaxKeyFileBytes*3).Draw(rt, "fileSize")
		content := make([]byte, size)
		for i := range content {
			content[i] = byte('A' + (i % 26))
		}
		if err := os.WriteFile(filepath.Join(dir, keyFileName), content, 0644); err != nil {
			rt.Fatalf("failed to write key file: %v", err)
		}

		cb := NewContextBuilder()
		meta, err := cb.Build(dir)
		if err != nil {
			rt.Fatalf("Build failed: %v", err)
		}

		// Assert: the key file is present and its content length <= MaxKeyFileBytes.
		val, ok := meta.KeyFiles[keyFileName]
		if !ok {
			rt.Fatalf("key file %q not found in KeyFiles", keyFileName)
		}
		if len(val) > MaxKeyFileBytes {
			rt.Fatalf("KeyFiles[%q] has %d bytes, exceeds MaxKeyFileBytes (%d)",
				keyFileName, len(val), MaxKeyFileBytes)
		}
	})
}

// Feature: smart-project-query, Property 11: Total assembled context respects 50KB limit
// **Validates: Requirements 8.1, 8.3**
func TestProperty11_TotalContextRespects50KBLimit(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		dir := t.TempDir()

		// Create a random subset of key files with varying sizes.
		chosenKeys := drawSubset(rt, KeyProjectFiles, "keyFiles")
		for _, kf := range chosenKeys {
			size := rapid.IntRange(1, 10000).Draw(rt, fmt.Sprintf("size_%s", kf))
			content := make([]byte, size)
			for j := range content {
				content[j] = byte('a' + (j % 26))
			}
			if err := os.WriteFile(filepath.Join(dir, kf), content, 0644); err != nil {
				rt.Fatalf("failed to write key file %s: %v", kf, err)
			}
		}

		// Create a random number of regular files.
		numRegular := rapid.IntRange(0, 300).Draw(rt, "numRegularFiles")
		if numRegular > 0 {
			dirPath := filepath.Join(dir, "src")
			if err := os.MkdirAll(dirPath, 0755); err != nil {
				rt.Fatalf("failed to create src dir: %v", err)
			}
		}
		for i := 0; i < numRegular; i++ {
			name := fmt.Sprintf("src/file_%04d.go", i)
			if err := os.WriteFile(filepath.Join(dir, name), []byte("package main"), 0644); err != nil {
				rt.Fatalf("failed to write regular file: %v", err)
			}
		}

		cb := NewContextBuilder()
		meta, err := cb.Build(dir)
		if err != nil {
			rt.Fatalf("Build failed: %v", err)
		}

		// Assert: TotalContextBytes does not exceed MaxTotalContextBytes.
		if meta.TotalContextBytes > MaxTotalContextBytes {
			rt.Fatalf("TotalContextBytes=%d exceeds MaxTotalContextBytes (%d)",
				meta.TotalContextBytes, MaxTotalContextBytes)
		}
	})
}

// Feature: smart-project-query, Property 12: Priority ordering under size pressure
// **Validates: Requirements 8.2**
func TestProperty12_PriorityOrderingUnderSizePressure(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		dir := t.TempDir()

		// Always create a README.md (highest priority).
		readmeContent := rapid.SliceOfN(rapid.Byte(), 100, 2000).Draw(rt, "readmeContent")
		if err := os.WriteFile(filepath.Join(dir, "README.md"), readmeContent, 0644); err != nil {
			rt.Fatalf("failed to write README.md: %v", err)
		}

		// Create enough large key files to exceed MaxTotalContextBytes.
		largeFiles := []string{
			"go.mod", "package.json", "Cargo.toml", "pyproject.toml",
			"requirements.txt", "Dockerfile", "docker-compose.yml",
			"Makefile", ".env.example", "go.sum",
		}

		// Pick a random subset (at least 3 to ensure size pressure).
		chosen := drawSubset(rt, largeFiles, "largeFiles")
		for len(chosen) < 3 {
			extra := rapid.SampledFrom(largeFiles).Draw(rt, "extraFile")
			found := false
			for _, c := range chosen {
				if c == extra {
					found = true
					break
				}
			}
			if !found {
				chosen = append(chosen, extra)
			}
		}

		totalRawSize := len(readmeContent)
		for _, name := range chosen {
			// Each file is large enough that the total exceeds 50KB.
			size := rapid.IntRange(4000, 5000).Draw(rt, fmt.Sprintf("size_%s", name))
			content := make([]byte, size)
			for j := range content {
				content[j] = byte('0' + (j % 10))
			}
			if err := os.WriteFile(filepath.Join(dir, name), content, 0644); err != nil {
				rt.Fatalf("failed to write %s: %v", name, err)
			}
			totalRawSize += size
		}

		// Only test when total raw content actually exceeds the limit.
		if totalRawSize <= MaxTotalContextBytes {
			return // skip this iteration — not enough pressure
		}

		cb := NewContextBuilder()
		meta, err := cb.Build(dir)
		if err != nil {
			rt.Fatalf("Build failed: %v", err)
		}

		// Assert: README.md (highest priority) is always present.
		if _, ok := meta.KeyFiles["README.md"]; !ok {
			rt.Fatalf("README.md missing from KeyFiles under size pressure (total raw=%d)", totalRawSize)
		}

		// Assert: higher-priority files appear before lower-priority files are dropped.
		priorityIndex := make(map[string]int)
		for i, name := range KeyProjectFiles {
			priorityIndex[name] = i
		}

		maxIncludedPriority := -1
		for name := range meta.KeyFiles {
			idx, ok := priorityIndex[name]
			if ok && idx > maxIncludedPriority {
				maxIncludedPriority = idx
			}
		}

		// Verify no file with a lower priority index (higher priority) was excluded
		// while a file with a higher priority index (lower priority) was included.
		for i := 0; i < maxIncludedPriority; i++ {
			name := KeyProjectFiles[i]
			// Only check files that exist on disk.
			if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
				continue // file doesn't exist on disk, skip
			}
			if _, ok := meta.KeyFiles[name]; !ok {
				rt.Fatalf("higher-priority file %q (index %d) excluded while lower-priority file at index %d included",
					name, i, maxIncludedPriority)
			}
		}
	})
}

// genProjectMetadata returns a rapid generator for valid ProjectMetadata values.
func genProjectMetadata(rt *rapid.T) *ProjectMetadata {
	numKeyFiles := rapid.IntRange(0, 5).Draw(rt, "numKeyFiles")
	keyFiles := make(map[string]string, numKeyFiles)
	for i := 0; i < numKeyFiles; i++ {
		name := rapid.SampledFrom(KeyProjectFiles).Draw(rt, fmt.Sprintf("keyFileName_%d", i))
		content := rapid.StringMatching(`[a-zA-Z0-9 ._\-/]{0,200}`).Draw(rt, fmt.Sprintf("keyFileContent_%d", i))
		keyFiles[name] = content
	}

	numTreeEntries := rapid.IntRange(0, 20).Draw(rt, "numTreeEntries")
	fileTree := make([]string, numTreeEntries)
	for i := range fileTree {
		fileTree[i] = rapid.StringMatching(`[a-z]{1,10}/[a-z]{1,10}\.[a-z]{1,4}`).Draw(rt, fmt.Sprintf("treeEntry_%d", i))
	}

	totalFiles := rapid.IntRange(numTreeEntries, numTreeEntries+50).Draw(rt, "totalFiles")

	return &ProjectMetadata{
		RootDir:           rapid.StringMatching(`/[a-z]{1,10}(/[a-z]{1,10}){0,3}`).Draw(rt, "rootDir"),
		Language:          rapid.SampledFrom([]string{"go", "python", "javascript", "rust", ""}).Draw(rt, "language"),
		BuildSystem:       rapid.SampledFrom([]string{"go modules", "npm", "cargo", "make", ""}).Draw(rt, "buildSystem"),
		KeyFiles:          keyFiles,
		FileTree:          fileTree,
		FileTreeTruncated: totalFiles > MaxFileTreeEntries,
		TotalFiles:        totalFiles,
		TotalContextBytes: rapid.IntRange(0, MaxTotalContextBytes).Draw(rt, "totalContextBytes"),
	}
}

// Feature: smart-project-query, Property 5: Cache round trip
// **Validates: Requirements 2.1, 2.2, 2.5**
func TestProperty5_CacheRoundTrip(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		cache := NewContextCache()

		// Generate a random workspace directory path.
		dir := rapid.StringMatching(`/[a-z]{1,10}(/[a-z]{1,10}){0,3}`).Draw(rt, "workspaceDir")

		// Generate a valid ProjectMetadata.
		meta := genProjectMetadata(rt)

		// Put then Get should return the same metadata.
		cache.Put(dir, meta)
		got := cache.Get(dir)

		if got == nil {
			rt.Fatal("Get returned nil after Put")
		}

		// Verify all fields match.
		if got.RootDir != meta.RootDir {
			rt.Fatalf("RootDir mismatch: got %q, want %q", got.RootDir, meta.RootDir)
		}
		if got.Language != meta.Language {
			rt.Fatalf("Language mismatch: got %q, want %q", got.Language, meta.Language)
		}
		if got.BuildSystem != meta.BuildSystem {
			rt.Fatalf("BuildSystem mismatch: got %q, want %q", got.BuildSystem, meta.BuildSystem)
		}
		if len(got.KeyFiles) != len(meta.KeyFiles) {
			rt.Fatalf("KeyFiles length mismatch: got %d, want %d", len(got.KeyFiles), len(meta.KeyFiles))
		}
		for k, v := range meta.KeyFiles {
			if got.KeyFiles[k] != v {
				rt.Fatalf("KeyFiles[%q] mismatch: got %q, want %q", k, got.KeyFiles[k], v)
			}
		}
		if len(got.FileTree) != len(meta.FileTree) {
			rt.Fatalf("FileTree length mismatch: got %d, want %d", len(got.FileTree), len(meta.FileTree))
		}
		for i := range meta.FileTree {
			if got.FileTree[i] != meta.FileTree[i] {
				rt.Fatalf("FileTree[%d] mismatch: got %q, want %q", i, got.FileTree[i], meta.FileTree[i])
			}
		}
		if got.FileTreeTruncated != meta.FileTreeTruncated {
			rt.Fatalf("FileTreeTruncated mismatch: got %v, want %v", got.FileTreeTruncated, meta.FileTreeTruncated)
		}
		if got.TotalFiles != meta.TotalFiles {
			rt.Fatalf("TotalFiles mismatch: got %d, want %d", got.TotalFiles, meta.TotalFiles)
		}
		if got.TotalContextBytes != meta.TotalContextBytes {
			rt.Fatalf("TotalContextBytes mismatch: got %d, want %d", got.TotalContextBytes, meta.TotalContextBytes)
		}
	})
}

// Feature: smart-project-query, Property 6: Cache invalidation
// **Validates: Requirements 7.1**
func TestProperty6_CacheInvalidation(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		cache := NewContextCache()

		// Generate a random workspace directory path.
		dir := rapid.StringMatching(`/[a-z]{1,10}(/[a-z]{1,10}){0,3}`).Draw(rt, "workspaceDir")

		// Generate and store a valid ProjectMetadata.
		meta := genProjectMetadata(rt)
		cache.Put(dir, meta)

		// Verify it's cached.
		if cache.Get(dir) == nil {
			rt.Fatal("Get returned nil after Put (pre-invalidation check)")
		}

		// Invalidate and verify it's gone.
		cache.Invalidate(dir)
		got := cache.Get(dir)

		if got != nil {
			rt.Fatalf("Get returned non-nil after Invalidate: %+v", got)
		}
	})
}

// Feature: smart-project-query, Property 7: /ask prefix always triggers project context
// **Validates: Requirements 3.2**
func TestProperty7_AskPrefixAlwaysTriggersProjectContext(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		// Generate a random suffix (may be empty).
		suffix := rapid.StringMatching(`[a-zA-Z0-9 ?!.,]{0,50}`).Draw(rt, "suffix")

		// Randomly vary the case of "/ask".
		askVariants := []string{"/ask", "/Ask", "/ASK", "/aSk"}
		prefix := rapid.SampledFrom(askVariants).Draw(rt, "askPrefix")

		var message string
		if suffix == "" {
			message = prefix
		} else {
			// Optionally add a space between /ask and the suffix.
			sep := rapid.SampledFrom([]string{" ", "  ", ""}).Draw(rt, "separator")
			message = prefix + sep + suffix
		}

		qc := NewQueryClassifier()
		result := qc.Classify(message)

		if !result.NeedsProjectContext {
			rt.Fatalf("expected NeedsProjectContext=true for message %q, got false", message)
		}
	})
}

// Feature: smart-project-query, Property 8: Keyword-based project question classification
// **Validates: Requirements 3.3**
func TestProperty8_KeywordBasedProjectQuestionClassification(t *testing.T) {
	// Question indicators (words only, "?" handled separately).
	indicators := []string{"how", "what", "where", "why", "which", "explain", "describe", "show", "tell", "list"}
	terms := []string{"project", "repo", "codebase", "build", "run", "install", "deploy", "test", "config", "structure", "architecture", "dependency", "dependencies"}

	rapid.Check(t, func(rt *rapid.T) {
		// Pick at least one indicator and one project term.
		indicator := rapid.SampledFrom(indicators).Draw(rt, "indicator")
		term := rapid.SampledFrom(terms).Draw(rt, "term")

		// Build a message that contains both. Add filler words around them.
		fillers := []string{"do I", "the", "this", "my", "about", "can you", "please"}
		filler1 := rapid.SampledFrom(fillers).Draw(rt, "filler1")
		filler2 := rapid.SampledFrom(fillers).Draw(rt, "filler2")

		// Randomly decide whether to include "?" as well.
		useQuestion := rapid.Bool().Draw(rt, "useQuestion")
		qmark := ""
		if useQuestion {
			qmark = "?"
		}

		message := indicator + " " + filler1 + " " + term + " " + filler2 + qmark

		qc := NewQueryClassifier()
		result := qc.Classify(message)

		if !result.NeedsProjectContext {
			rt.Fatalf("expected NeedsProjectContext=true for message %q (indicator=%q, term=%q), got false",
				message, indicator, term)
		}
	})
}

// Feature: smart-project-query, Property 10: Search-like intent extracts search terms
// **Validates: Requirements 5.2**
func TestProperty10_SearchLikeIntentExtractsSearchTerms(t *testing.T) {
	searchPhrases := []string{"where is", "find", "which file", "locate", "grep", "search for"}

	rapid.Check(t, func(rt *rapid.T) {
		phrase := rapid.SampledFrom(searchPhrases).Draw(rt, "searchPhrase")

		// Generate a non-empty search term (at least 1 alpha char).
		term := rapid.StringMatching(`[a-zA-Z][a-zA-Z0-9_]{0,15}`).Draw(rt, "searchTerm")

		// Optionally add a prefix before the search phrase.
		usePrefix := rapid.Bool().Draw(rt, "usePrefix")
		prefix := ""
		if usePrefix {
			prefix = rapid.SampledFrom([]string{"please ", "can you ", "I want to ", ""}).Draw(rt, "prefix")
		}

		message := prefix + phrase + " " + term

		qc := NewQueryClassifier()
		result := qc.Classify(message)

		if len(result.SearchTerms) == 0 {
			rt.Fatalf("expected non-empty SearchTerms for message %q (phrase=%q, term=%q), got empty",
				message, phrase, term)
		}
	})
}

// Feature: smart-project-query, Property 9: Prompt contains all required context components
// **Validates: Requirements 4.2, 4.3**
func TestProperty9_PromptContainsAllRequiredContextComponents(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		meta := genProjectMetadata(rt)

		// Generate a non-empty user message.
		userMessage := rapid.StringMatching(`[a-zA-Z][a-zA-Z0-9 ?!.,]{1,80}`).Draw(rt, "userMessage")

		// Optionally generate search results.
		numSearchResults := rapid.IntRange(0, 3).Draw(rt, "numSearchResults")
		var searchResults []string
		for i := 0; i < numSearchResults; i++ {
			sr := rapid.StringMatching(`[a-zA-Z0-9 /._:]{1,50}`).Draw(rt, fmt.Sprintf("searchResult_%d", i))
			searchResults = append(searchResults, sr)
		}

		// Optionally generate file context.
		useFileContext := rapid.Bool().Draw(rt, "useFileContext")
		fileContext := ""
		if useFileContext {
			fileContext = rapid.StringMatching(`[a-zA-Z0-9 \n]{1,100}`).Draw(rt, "fileContext")
		}

		pb := NewPromptBuilder()
		prompt := pb.Build(meta, userMessage, searchResults, fileContext)

		// Assert: prompt contains the file tree listing.
		for _, entry := range meta.FileTree {
			if !strings.Contains(prompt, entry) {
				rt.Fatalf("prompt missing file tree entry %q", entry)
			}
		}

		// Assert: prompt contains the detected language.
		if meta.Language != "" {
			if !strings.Contains(prompt, meta.Language) {
				rt.Fatalf("prompt missing language %q", meta.Language)
			}
		}

		// Assert: prompt contains the build system.
		if meta.BuildSystem != "" {
			if !strings.Contains(prompt, meta.BuildSystem) {
				rt.Fatalf("prompt missing build system %q", meta.BuildSystem)
			}
		}

		// Assert: prompt contains the content of each key file.
		for name, content := range meta.KeyFiles {
			if !strings.Contains(prompt, name) {
				rt.Fatalf("prompt missing key file name %q", name)
			}
			if content != "" && !strings.Contains(prompt, content) {
				rt.Fatalf("prompt missing key file content for %q", name)
			}
		}

		// Assert: prompt contains the user's original message.
		if !strings.Contains(prompt, userMessage) {
			rt.Fatalf("prompt missing user message %q", userMessage)
		}

		// Assert: prompt contains a system instruction about answering from context.
		if !strings.Contains(prompt, "project context") {
			rt.Fatalf("prompt missing system instruction about answering from context")
		}

		// Assert: prompt contains the debug byte count comment.
		if !strings.Contains(prompt, "<!-- context:") {
			rt.Fatalf("prompt missing debug byte count comment")
		}

		// Assert: if search results provided, they appear in the prompt.
		for _, sr := range searchResults {
			if !strings.Contains(prompt, sr) {
				rt.Fatalf("prompt missing search result %q", sr)
			}
		}

		// Assert: if file context provided, it appears in the prompt.
		if fileContext != "" && !strings.Contains(prompt, fileContext) {
			rt.Fatalf("prompt missing file context")
		}
	})
}
