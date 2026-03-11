package agentic

import (
	"bufio"
	"encoding/json"
	"fmt"
	iofs "io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/user/terminal-intelligence/internal/executor"
)

// skipDirs is the set of directory names that fileScanner will never descend into.
var skipDirs = map[string]bool{
	".git":         true,
	"vendor":       true,
	"node_modules": true,
	".ti":          true,
	"build":        true,
}

// allowedExtensions is the set of file extensions that fileScanner considers text files.
var allowedExtensions = map[string]bool{
	".go":   true,
	".md":   true,
	".sh":   true,
	".bash": true,
	".ps1":  true,
	".py":   true,
	".ts":   true,
	".js":   true,
	".json": true,
	".yaml": true,
	".yml":  true,
	".toml": true,
	".txt":  true,
	".html": true,
	".css":  true,
}

// gitIgnoreMatcher provides basic .gitignore matching.
type gitIgnoreMatcher struct {
	rules []string
}

func newGitIgnoreMatcher(root string) *gitIgnoreMatcher {
	m := &gitIgnoreMatcher{}
	data, err := os.ReadFile(filepath.Join(root, ".gitignore"))
	if err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "#") {
				m.rules = append(m.rules, line)
			}
		}
	}
	return m
}

func (m *gitIgnoreMatcher) matches(relPath string) bool {
	if len(m.rules) == 0 {
		return false
	}
	
	normalizedPath := filepath.ToSlash(relPath)
	parts := strings.Split(normalizedPath, "/")
	
	for _, rule := range m.rules {
		isRootOnly := strings.HasPrefix(rule, "/")
		isDirOnly := strings.HasSuffix(rule, "/")
		
		cleanRule := strings.Trim(rule, "/")
		
		// If rule starts with /, it must match at the root level of the project
		if isRootOnly {
			if normalizedPath == cleanRule || strings.HasPrefix(normalizedPath, cleanRule+"/") {
				return true
			}
			continue
		}
		
		// If it doesn't start with /, it can match any level
		for _, part := range parts {
			if part == cleanRule {
				return true
			}
			if matched, err := filepath.Match(cleanRule, part); err == nil && matched {
				return true
			}
		}
		
		if !isDirOnly {
			if normalizedPath == cleanRule {
				return true
			}
			if matched, err := filepath.Match(cleanRule, normalizedPath); err == nil && matched {
				return true
			}
		}
	}
	return false
}

// fileScanner recursively enumerates text files under a root directory.
type fileScanner struct {
	root     string
	maxDepth int // default 20
	maxFiles int // default 2000
	ignore   *gitIgnoreMatcher
}

// scan returns all text-file paths under root, respecting skip rules.
// Returns (paths, truncated, error) where truncated is true when the raw
// candidate count exceeded maxFiles and the list was shortened.
func (fs *fileScanner) scan() ([]string, bool, error) {
	var collected []string

	err := filepath.WalkDir(fs.root, func(path string, d iofs.DirEntry, err error) error {
		if err != nil {
			// Surface permission / I/O errors with path context (Requirement 2.5).
			return &scanError{path: path, cause: err}
		}

		// Compute depth relative to root.
		rel, relErr := filepath.Rel(fs.root, path)
		if relErr != nil {
			return relErr
		}

		// root itself — skip depth check, just continue.
		if rel == "." {
			return nil
		}

		depth := strings.Count(rel, string(filepath.Separator)) + 1

		// Check .gitignore
		if fs.ignore.matches(rel) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if d.IsDir() {
			// Skip banned directories regardless of depth.
			if skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			// Prune beyond max depth.
			if depth >= fs.maxDepth {
				return filepath.SkipDir
			}
			return nil
		}

		// Regular file: apply extension filter.
		if !allowedExtensions[strings.ToLower(filepath.Ext(d.Name()))] {
			return nil
		}

		collected = append(collected, rel)
		return nil
	})

	if err != nil {
		return nil, false, err
	}

	// Truncation: keep the closest files.
	truncated := false
	if len(collected) > fs.maxFiles {
		sort.Slice(collected, func(i, j int) bool {
			return len(collected[i]) < len(collected[j])
		})
		collected = collected[:fs.maxFiles]
		truncated = true
	}

	// Convert relative paths back to absolute paths for callers.
	abs := make([]string, len(collected))
	for i, rel := range collected {
		abs[i] = filepath.Join(fs.root, rel)
	}

	return abs, truncated, nil
}

// scanError wraps a filesystem error with the offending path.
type scanError struct {
	path  string
	cause error
}

func (e *scanError) Error() string {
	return "scan error at " + e.path + ": " + e.cause.Error()
}

func (e *scanError) Unwrap() error { return e.cause }

// newFileScanner creates a fileScanner with the default limits.
func newFileScanner(root string, maxFiles int) *fileScanner {
	if maxFiles <= 0 {
		maxFiles = 2000
	}
	return &fileScanner{
		root:     root,
		maxDepth: 20,
		maxFiles: maxFiles,
		ignore:   newGitIgnoreMatcher(root),
	}
}

// relevanceRanker uses the AI model to rank candidate files by relevance to a request.
type relevanceRanker struct {
	aiClient AIClient
	model    string
}

// newRelevanceRanker creates a relevanceRanker with the given AI client and model.
func newRelevanceRanker(aiClient AIClient, model string) *relevanceRanker {
	return &relevanceRanker{
		aiClient: aiClient,
		model:    model,
	}
}

// rank returns up to maxResults file paths most relevant to the request.
// It reads the first 100 lines of each candidate to include in the prompt.
// Returns (ranked, hallucinated, error) where hallucinated contains AI-returned
// paths that do not exist under root.
func (rr *relevanceRanker) rank(
	paths []string,
	request string,
	root string,
	maxResults int,
) ([]string, []string, error) {
	if len(paths) == 0 {
		return nil, nil, nil
	}

	// Build a set of known absolute paths for fast lookup.
	knownPaths := make(map[string]bool, len(paths))
	for _, p := range paths {
		abs, err := filepath.Abs(p)
		if err == nil {
			knownPaths[abs] = true
		}
	}

	// Resolve root to absolute for prefix checks.
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot resolve project root %q: %w", root, err)
	}
	// Ensure root ends with separator so prefix check is unambiguous.
	if !strings.HasSuffix(absRoot, string(filepath.Separator)) {
		absRoot += string(filepath.Separator)
	}

	// Build the ranking prompt.
	prompt := rr.buildRankingPrompt(paths, request, maxResults)

	// Call the AI.
	responseChan, err := rr.aiClient.Generate(prompt, rr.model, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("relevance ranker AI call failed: %w", err)
	}

	var sb strings.Builder
	for chunk := range responseChan {
		sb.WriteString(chunk)
	}
	rawResponse := strings.TrimSpace(sb.String())

	// Parse the AI response into a list of paths.
	aiPaths := parseRankedPaths(rawResponse)

	// Validate each returned path: must exist under root.
	var ranked []string
	var hallucinated []string

	for _, p := range aiPaths {
		if len(ranked) >= maxResults {
			break
		}

		// Normalise: if the AI returned a relative path, make it absolute.
		candidate := p
		if !filepath.IsAbs(candidate) {
			candidate = filepath.Join(root, candidate)
		}
		absCandidate, err := filepath.Abs(candidate)
		if err != nil {
			hallucinated = append(hallucinated, p)
			continue
		}

		// Must be under project root.
		if !strings.HasPrefix(absCandidate+string(filepath.Separator), absRoot) &&
			absCandidate != strings.TrimSuffix(absRoot, string(filepath.Separator)) {
			hallucinated = append(hallucinated, p)
			continue
		}

		// Must actually exist on disk.
		if _, statErr := os.Stat(absCandidate); statErr != nil {
			hallucinated = append(hallucinated, p)
			continue
		}

		// Must be in the original candidate set.
		if !knownPaths[absCandidate] {
			hallucinated = append(hallucinated, p)
			continue
		}

		ranked = append(ranked, absCandidate)
	}

	return ranked, hallucinated, nil
}

// buildRankingPrompt constructs the prompt sent to the AI for relevance ranking.
// It includes the first 100 lines of each candidate file.
func (rr *relevanceRanker) buildRankingPrompt(paths []string, request string, maxResults int) string {
	var sb strings.Builder

	sb.WriteString("You are a code assistant helping to identify which files need to be modified.\n\n")
	sb.WriteString("User request: ")
	sb.WriteString(request)
	sb.WriteString("\n\n")
	sb.WriteString(fmt.Sprintf("From the files listed below, return the %d most relevant file paths that are likely to require modification to fulfil the user request.\n", maxResults))
	sb.WriteString("Respond with ONLY a JSON array of file paths, e.g.: [\"path/to/file.go\", \"other/file.md\"]\n")
	sb.WriteString("If no files are relevant, respond with an empty array: []\n\n")
	sb.WriteString("=== CANDIDATE FILES ===\n\n")

	for _, p := range paths {
		sb.WriteString("--- FILE: ")
		sb.WriteString(p)
		sb.WriteString(" ---\n")
		preview := readFirstNLines(p, 100)
		if preview != "" {
			sb.WriteString(preview)
			if !strings.HasSuffix(preview, "\n") {
				sb.WriteString("\n")
			}
		} else {
			sb.WriteString("(empty or unreadable)\n")
		}
		sb.WriteString("\n")
	}

	sb.WriteString("=== END OF FILES ===\n\n")
	sb.WriteString(fmt.Sprintf("Return a JSON array of at most %d file paths most relevant to: %s\n", maxResults, request))

	return sb.String()
}

// readFirstNLines reads up to n lines from the file at path and returns them joined.
// Returns an empty string if the file cannot be opened.
func readFirstNLines(path string, n int) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() && len(lines) < n {
		lines = append(lines, scanner.Text())
	}
	return strings.Join(lines, "\n")
}

// parseRankedPaths extracts a list of file paths from the AI response.
// It first tries to parse a JSON array; if that fails it falls back to
// treating each non-empty line as a path.
func parseRankedPaths(response string) []string {
	// Strip markdown code fences if present.
	cleaned := strings.TrimSpace(response)
	if strings.HasPrefix(cleaned, "```") {
		lines := strings.SplitN(cleaned, "\n", 2)
		if len(lines) == 2 {
			rest := lines[1]
			if idx := strings.LastIndex(rest, "```"); idx >= 0 {
				rest = rest[:idx]
			}
			cleaned = strings.TrimSpace(rest)
		}
	}

	// Try JSON array first.
	if strings.HasPrefix(cleaned, "[") {
		var paths []string
		if err := json.Unmarshal([]byte(cleaned), &paths); err == nil {
			var result []string
			for _, p := range paths {
				p = strings.TrimSpace(p)
				if p != "" {
					result = append(result, p)
				}
			}
			return result
		}
	}

	// Fallback: one path per line.
	var result []string
	for _, line := range strings.Split(cleaned, "\n") {
		line = strings.TrimSpace(line)
		// Skip lines that look like prose (contain spaces and no path separator).
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
			continue
		}
		result = append(result, line)
	}
	return result
}

// ─── MultiFileEditor ──────────────────────────────────────────────────────────

// multiFileEditor reads multiple files, assembles a consolidated AI prompt,
// parses the AI response into per-file patches, validates and applies them.
type multiFileEditor struct {
	aiClient  AIClient
	model     string
	fixParser *FixParser
	root      string // absolute project root
	preview   bool
}

// newMultiFileEditor creates a multiFileEditor with the given dependencies.
func newMultiFileEditor(aiClient AIClient, model string, fixParser *FixParser, root string, preview bool) *multiFileEditor {
	return &multiFileEditor{
		aiClient:  aiClient,
		model:     model,
		fixParser: fixParser,
		root:      root,
		preview:   preview,
	}
}

// edit reads each file (up to 2000 lines), builds a consolidated prompt, calls
// the AI, splits the response on "=== FILE:" boundaries, validates and applies
// patches, and returns per-file results alongside any execution command.
//
// Returns: (modified []FileResult, failures []PatchFailure, unreadable []string, outOfScope []string, execCmd string, error)
func (me *multiFileEditor) edit(
	paths []string,
	request string,
) ([]FileResult, []PatchFailure, []string, []string, string, error) {
	var modified []FileResult
	var failures []PatchFailure
	var unreadable []string
	var outOfScope []string
	var execCmd string

	// ── Step 1: Resolve root to absolute path ────────────────────────────────
	absRoot, err := filepath.Abs(me.root)
	if err != nil {
		return nil, nil, nil, nil, "", fmt.Errorf("cannot resolve project root %q: %w", me.root, err)
	}
	// Ensure root ends with separator for unambiguous prefix checks.
	rootPrefix := absRoot
	if !strings.HasSuffix(rootPrefix, string(filepath.Separator)) {
		rootPrefix += string(filepath.Separator)
	}

	// ── Step 2: Read each file (up to 2000 lines) ────────────────────────────
	var entries []mfeFileEntry

	for _, p := range paths {
		// 4.6 Path safety check.
		safe, absP, reason := me.checkPathSafety(p, absRoot, rootPrefix)
		if !safe {
			outOfScope = append(outOfScope, p)
			_ = reason
			_ = absP
			continue
		}

		// Stat to get original permissions.
		info, statErr := os.Stat(absP)
		if statErr != nil {
			unreadable = append(unreadable, p)
			continue
		}

		// Read up to 2000 lines.
		content, truncated, readErr := readUpToNLines(absP, 2000)
		if readErr != nil {
			unreadable = append(unreadable, p)
			continue
		}
		if truncated {
			content += "\n[TRUNCATED: file exceeds 2000 lines; only the first 2000 lines are shown]"
		}

		relPath, _ := filepath.Rel(absRoot, absP)
		entries = append(entries, mfeFileEntry{
			absPath:  absP,
			relPath:  relPath,
			content:  content,
			origPerm: info.Mode().Perm(),
		})
	}

	if len(entries) == 0 {
		// Nothing to edit — all paths were out-of-scope or unreadable.
		return modified, failures, unreadable, outOfScope, "", nil
	}

	// ── Step 3: Build consolidated prompt ────────────────────────────────────
	prompt := me.buildEditPrompt(entries, request)

	// ── Step 4: Call AI ───────────────────────────────────────────────────────
	responseChan, err := me.aiClient.Generate(prompt, me.model, nil)
	if err != nil {
		return nil, nil, unreadable, outOfScope, "", fmt.Errorf("AI generate call failed: %w", err)
	}
	var sb strings.Builder
	for chunk := range responseChan {
		sb.WriteString(chunk)
	}
	aiResponse := sb.String()

	execCmd = extractExecuteCommand(aiResponse)

	// ── Step 5: Split response on "=== FILE:" boundaries ─────────────────────
	fileSections := splitOnFileHeaders(aiResponse)

	// Build a lookup from relPath → entry for fast matching.
	entryByRel := make(map[string]mfeFileEntry, len(entries))
	for _, e := range entries {
		entryByRel[e.relPath] = e
		// Also index by absolute path in case AI returns absolute paths.
		entryByRel[e.absPath] = e
	}

	// ── Step 6: Per-file: validate + apply patches ────────────────────────────
	for filePath, patchText := range fileSections {
		// Normalise the path the AI returned.
		filePath = strings.TrimSpace(filePath)

		// Parse patches from the section text.
		patches := parseSearchReplace(patchText)
		if len(patches) == 0 {
			// No patches found for this file — not an error, just nothing to do.
			continue
		}

		// Try to find the matching entry.
		entry, ok := entryByRel[filePath]
		if !ok {
			// Try joining with root in case AI returned a relative path.
			candidate := filepath.Join(absRoot, filePath)
			absCandidate, absErr := filepath.Abs(candidate)
			if absErr == nil {
				relCandidate, _ := filepath.Rel(absRoot, absCandidate)
				entry, ok = entryByRel[relCandidate]
				if !ok {
					entry, ok = entryByRel[absCandidate]
				}
			}
			
			// If not found, check if it's a new file
			if !ok {
				hasNewFile := false
				for _, p := range patches {
					if p.isNewFile {
						hasNewFile = true
						break
					}
				}
				if hasNewFile {
					safe, absP, _ := me.checkPathSafety(filePath, absRoot, rootPrefix)
					if safe {
						relCandidate, _ := filepath.Rel(absRoot, absP)
						entry = mfeFileEntry{
							absPath:  absP,
							relPath:  relCandidate,
							content:  "",
							origPerm: 0644,
						}
						ok = true
					}
				}
			}
		}
		if !ok {
			// AI hallucinated a path not in our entry set — skip silently.
			continue
		}

		// 4.5 Duplicate ~~~SEARCH marker detection.
		if hasDuplicateSearchMarkers(patchText) {
			failures = append(failures, PatchFailure{
				Path:   entry.relPath,
				Reason: "duplicate ~~~SEARCH markers detected in patch",
			})
			continue
		}

		// Apply patches sequentially to the file content.
		currentContent := entry.content
		// Strip any truncation notice before applying patches.
		currentContent = strings.TrimSuffix(currentContent, "\n[TRUNCATED: file exceeds 2000 lines; only the first 2000 lines are shown]")

		applyFailed := false
		for _, patch := range patches {
			var newContent string
			var applyErr error
			if patch.isNewFile {
				newContent = patch.replace
				if !strings.HasSuffix(newContent, "\n") {
					newContent += "\n"
				}
			} else {
				newContent, applyErr = applySearchReplace(currentContent, patch.search, patch.replace)
			}
			if applyErr != nil {
				failures = append(failures, PatchFailure{
					Path:   entry.relPath,
					Reason: fmt.Sprintf("patch apply failed: %s", applyErr.Error()),
				})
				applyFailed = true
				break
			}
			currentContent = newContent
		}
		if applyFailed {
			continue
		}

		// Count lines added/removed.
		origLines := strings.Split(entry.content, "\n")
		newLines := strings.Split(currentContent, "\n")
		linesAdded := 0
		linesRemoved := 0
		if len(newLines) > len(origLines) {
			linesAdded = len(newLines) - len(origLines)
		} else {
			linesRemoved = len(origLines) - len(newLines)
		}

		// 4.7 / 4.8 Write file (or skip in preview mode).
		if !me.preview {
			if err := os.MkdirAll(filepath.Dir(entry.absPath), 0755); err != nil {
				failures = append(failures, PatchFailure{
					Path:   entry.relPath,
					Reason: fmt.Sprintf("mkdir failed: %s", err.Error()),
				})
				continue
			}

			writeErr := os.WriteFile(entry.absPath, []byte(currentContent), entry.origPerm)
			if writeErr != nil {
				failures = append(failures, PatchFailure{
					Path:   entry.relPath,
					Reason: fmt.Sprintf("write failed: %s", writeErr.Error()),
				})
				continue
			}
		}

		modified = append(modified, FileResult{
			Path:         entry.absPath,
			RelPath:      entry.relPath,
			LinesAdded:   linesAdded,
			LinesRemoved: linesRemoved,
		})
	}

	return modified, failures, unreadable, outOfScope, execCmd, nil
}

// mfeFileEntry holds the data for a single file being edited.
type mfeFileEntry struct {
	absPath  string
	relPath  string
	content  string
	origPerm os.FileMode
}

// buildEditPrompt assembles the consolidated prompt for the AI.
func (me *multiFileEditor) buildEditPrompt(entries []mfeFileEntry, request string) string {
	var sb strings.Builder

	sb.WriteString("You are a code assistant. Apply the following changes to the files listed below.\n\n")
	sb.WriteString("User request: ")
	sb.WriteString(request)
	sb.WriteString("\n\n")
	sb.WriteString("For each file that needs modification, output a section in this exact format:\n\n")
	sb.WriteString("=== FILE: <relative/path/to/file> ===\n")
	sb.WriteString("~~~SEARCH\n")
	sb.WriteString("<exact lines to find>\n")
	sb.WriteString("~~~REPLACE\n")
	sb.WriteString("<replacement lines>\n")
	sb.WriteString("~~~END\n\n")
	sb.WriteString("To create a completely new file, use this format instead:\n")
	sb.WriteString("=== FILE: <relative/path/to/new_file> ===\n")
	sb.WriteString("~~~NEWFILE\n")
	sb.WriteString("<entire file content>\n")
	sb.WriteString("~~~END\n\n")
	sb.WriteString("If you want to run a command to verify your changes (e.g., go test, python tests), output it OUTSIDE of any FILE blocks as:\n")
	sb.WriteString("~~~EXECUTE\n")
	sb.WriteString("<command>\n")
	sb.WriteString("~~~END\n\n")
	sb.WriteString("You may include multiple SEARCH/REPLACE blocks per file. ")
	sb.WriteString("Only output sections for files that need changes or creation. ")
	sb.WriteString("Do not output anything outside the === FILE: === sections, EXCEPT for an optional ~~~EXECUTE block at the end.\n\n")
	sb.WriteString("=== FILES TO CONSIDER ===\n\n")

	for _, e := range entries {
		sb.WriteString("=== FILE: ")
		sb.WriteString(e.relPath)
		sb.WriteString(" ===\n")
		sb.WriteString(e.content)
		if !strings.HasSuffix(e.content, "\n") {
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	sb.WriteString("=== END OF FILES ===\n\n")
	sb.WriteString("Now output the modifications using the === FILE: === / ~~~SEARCH / ~~~REPLACE / ~~~END or ~~~NEWFILE format described above.\n")

	return sb.String()
}

// checkPathSafety resolves a path to absolute form, resolves symlinks, and
// verifies it is under the project root. Returns (safe, absPath, reason).
func (me *multiFileEditor) checkPathSafety(p, absRoot, rootPrefix string) (bool, string, string) {
	// Make absolute.
	absP, err := filepath.Abs(p)
	if err != nil {
		return false, "", fmt.Sprintf("cannot make absolute: %s", err)
	}

	// Resolve symlinks.
	resolved, err := filepath.EvalSymlinks(absP)
	if err != nil {
		// If the file doesn't exist yet, EvalSymlinks fails; fall back to absP.
		resolved = absP
	}

	// Prefix check.
	if !strings.HasPrefix(resolved+string(filepath.Separator), rootPrefix) &&
		resolved != strings.TrimSuffix(rootPrefix, string(filepath.Separator)) {
		return false, resolved, fmt.Sprintf("path %q is outside project root %q", resolved, absRoot)
	}

	return true, resolved, ""
}

// extractExecuteCommand finds a ~~~EXECUTE block in the AI response.
func extractExecuteCommand(response string) string {
	lines := strings.Split(response, "\n")
	inExecute := false
	var cmd []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "~~~EXECUTE" {
			inExecute = true
			cmd = nil
			continue
		}
		if inExecute && trimmed == "~~~END" {
			break
		}
		if inExecute {
			cmd = append(cmd, line)
		}
	}
	return strings.TrimSpace(strings.Join(cmd, "\n"))
}

// ─── Patch helpers ────────────────────────────────────────────────────────────

// splitOnFileHeaders splits an AI response into a map of filePath → patchText.
// It looks for lines matching "=== FILE: <path> ===" as section delimiters.
func splitOnFileHeaders(response string) map[string]string {
	result := make(map[string]string)
	lines := strings.Split(response, "\n")

	var currentPath string
	var currentLines []string

	flush := func() {
		if currentPath != "" && len(currentLines) > 0 {
			result[currentPath] = strings.Join(currentLines, "\n")
		}
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "=== FILE:") && strings.HasSuffix(trimmed, "===") {
			flush()
			// Extract path: "=== FILE: path/to/file.go ==="
			inner := strings.TrimPrefix(trimmed, "=== FILE:")
			inner = strings.TrimSuffix(inner, "===")
			currentPath = strings.TrimSpace(inner)
			currentLines = nil
		} else if currentPath != "" {
			currentLines = append(currentLines, line)
		}
	}
	flush()

	return result
}

// searchReplacePatch holds a single SEARCH/REPLACE pair.
type searchReplacePatch struct {
	search    string
	replace   string
	isNewFile bool
}

// parseSearchReplace extracts all ~~~SEARCH / ~~~REPLACE / ~~~END blocks from text.
func parseSearchReplace(text string) []searchReplacePatch {
	var patches []searchReplacePatch
	lines := strings.Split(text, "\n")

	const (
		stateIdle    = 0
		stateSearch  = 1
		stateReplace = 2
		stateNewFile = 3
	)

	state := stateIdle
	var searchLines []string
	var replaceLines []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		switch state {
		case stateIdle:
			if trimmed == "~~~SEARCH" {
				state = stateSearch
				searchLines = nil
				replaceLines = nil
			} else if trimmed == "~~~NEWFILE" {
				state = stateNewFile
				replaceLines = nil
			}
		case stateSearch:
			if trimmed == "~~~REPLACE" {
				state = stateReplace
			} else {
				searchLines = append(searchLines, line)
			}
		case stateReplace:
			if trimmed == "~~~END" {
				patches = append(patches, searchReplacePatch{
					search:  strings.Join(searchLines, "\n"),
					replace: strings.Join(replaceLines, "\n"),
				})
				state = stateIdle
				searchLines = nil
				replaceLines = nil
			} else {
				replaceLines = append(replaceLines, line)
			}
		case stateNewFile:
			if trimmed == "~~~END" {
				patches = append(patches, searchReplacePatch{
					replace:   strings.Join(replaceLines, "\n"),
					isNewFile: true,
				})
				state = stateIdle
				replaceLines = nil
			} else {
				replaceLines = append(replaceLines, line)
			}
		}
	}

	return patches
}

// hasDuplicateSearchMarkers returns true when the patch text contains two or
// more ~~~SEARCH blocks whose search content is identical (same line range).
func hasDuplicateSearchMarkers(text string) bool {
	patches := parseSearchReplace(text)
	if len(patches) < 2 {
		return false
	}
	seen := make(map[string]bool, len(patches))
	for _, p := range patches {
		key := p.search
		if seen[key] {
			return true
		}
		seen[key] = true
	}
	return false
}

// applySearchReplace replaces the first occurrence of search in content with
// replace. Returns an error if search is not found.
func applySearchReplace(content, search, replace string) (string, error) {
	if search == "" {
		// Empty search means prepend the replacement (new-file-like insertion).
		return replace + "\n" + content, nil
	}
	idx := strings.Index(content, search)
	if idx == -1 {
		return content, fmt.Errorf("search text not found in file content")
	}
	return content[:idx] + replace + content[idx+len(search):], nil
}

// readUpToNLines reads up to n lines from the file at path.
// Returns (content, truncated, error).
func readUpToNLines(path string, n int) (string, bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", false, err
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
		if len(lines) == n {
			// Check if there are more lines.
			if scanner.Scan() {
				return strings.Join(lines, "\n"), true, nil
			}
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return "", false, err
	}
	return strings.Join(lines, "\n"), false, nil
}

// ─── ProjectFixer ─────────────────────────────────────────────────────────────

// ProjectFixer orchestrates project-wide agentic operations.
// It is the single entry point called by AIChatPane for /project commands.
type ProjectFixer struct {
	aiClient  AIClient
	model     string
	fixParser *FixParser
	executor  *executor.CommandExecutor
}

// NewProjectFixer creates a new ProjectFixer with the given AI client and model.
func NewProjectFixer(aiClient AIClient, model string) *ProjectFixer {
	return &ProjectFixer{
		aiClient:  aiClient,
		model:     model,
		fixParser: NewFixParser(),
		executor:  executor.NewCommandExecutor(),
	}
}

// ProcessProjectMessage is the single entry point called by AIChatPane.
// It parses the /preview and /project prefixes, validates inputs, and runs the
// scan → rank → edit pipeline, returning a ChangeReport.
//
// Requirements: 1.1, 1.2, 1.3, 1.4, 1.5, 9.4
func (pf *ProjectFixer) ProcessProjectMessage(
	message string,
	projectRoot string,
	statusUpdate func(phase string),
) (*ChangeReport, error) {
	// ── Step 1: Parse /preview and /project prefixes ──────────────────────────
	previewMode := false
	remaining := strings.TrimSpace(message)

	// Check for /preview prefix (Req 9.4)
	lowerRemaining := strings.ToLower(remaining)
	if strings.HasPrefix(lowerRemaining, "/preview") {
		previewMode = true
		remaining = strings.TrimSpace(remaining[len("/preview"):])
		lowerRemaining = strings.ToLower(remaining)
	}

	// Strip /project prefix (Req 1.1)
	if strings.HasPrefix(lowerRemaining, "/project") {
		remaining = strings.TrimSpace(remaining[len("/project"):])
	}

	// Req 1.5: /project with no additional text
	if strings.TrimSpace(remaining) == "" {
		return nil, fmt.Errorf("Please provide a description of the change you want to make.")
	}

	requestText := strings.TrimSpace(remaining)

	// ── Step 2: Validate projectRoot (Req 1.2) ───────────────────────────────
	if strings.TrimSpace(projectRoot) == "" {
		return nil, fmt.Errorf("No project root is set. Please open a project folder first.")
	}
	if _, err := os.Stat(projectRoot); err != nil {
		return nil, fmt.Errorf("No project root is set. Please open a project folder first.")
	}

	report := &ChangeReport{
		PreviewMode: previewMode,
	}

	// ── Step 3: Scan (Req 1.4) ────────────────────────────────────────────────
	callStatus(statusUpdate, "scanning")
	scanner := newFileScanner(projectRoot, 0)
	scannedPaths, truncated, scanErr := scanner.scan()
	if scanErr != nil {
		return nil, fmt.Errorf("file scan failed: %w", scanErr)
	}
	if truncated {
		report.TruncationWarning = fmt.Sprintf("More than %d files found; list was truncated to the %d shortest paths.", scanner.maxFiles, scanner.maxFiles)
	}
	report.FilesRead = scannedPaths

	// ── Step 4: Rank (Req 1.4) ────────────────────────────────────────────────
	callStatus(statusUpdate, "ranking")
	ranker := newRelevanceRanker(pf.aiClient, pf.model)
	ranked, hallucinated, rankErr := ranker.rank(scannedPaths, requestText, projectRoot, 20)
	if rankErr != nil {
		return nil, fmt.Errorf("relevance ranking failed: %w", rankErr)
	}
	report.HallucinatedPaths = hallucinated

	// Req 3.4: no relevant files found
	if len(ranked) == 0 {
		return report, nil
	}

	// ── Step 5: Edit and Execute Loop (Req 1.4) ────────────────────────────────────────────────
	var modified []FileResult
	var failures []PatchFailure
	var unreadable []string
	var outOfScope []string

	maxIterations := 3
	for i := 0; i < maxIterations; i++ {
		callStatus(statusUpdate, fmt.Sprintf("modifying (iteration %d/%d)", i+1, maxIterations))
		editor := newMultiFileEditor(pf.aiClient, pf.model, pf.fixParser, projectRoot, previewMode)
		
		mod, fail, unread, outScope, execCmd, editErr := editor.edit(ranked, requestText)
		if editErr != nil {
			return nil, fmt.Errorf("multi-file edit failed: %w", editErr)
		}

		// Accumulate results
		modified = append(modified, mod...)
		failures = append(failures, fail...)
		unreadable = append(unreadable, unread...)
		outOfScope = append(outOfScope, outScope...)

		// Execute if requested
		if execCmd != "" && !previewMode {
			callStatus(statusUpdate, fmt.Sprintf("executing verification: %s", execCmd))
			cmdResult, _ := pf.executor.ExecuteCommand(execCmd, projectRoot)
			
			if cmdResult != nil && cmdResult.ExitCode != 0 {
				// Execution failed, feedback to AI
				errorFeedback := fmt.Sprintf("\n\nI ran your verification command `%s` but it failed with exit code %d.\nStdout:\n%s\nStderr:\n%s\nPlease fix the issue and try again.", 
					execCmd, cmdResult.ExitCode, cmdResult.Stdout, cmdResult.Stderr)
				requestText += errorFeedback
				continue // loop to try again
			}
		}
		
		// If no execution command or execution succeeded, we are done
		break
	}

	report.FilesModified = modified
	report.PatchFailures = failures
	report.FilesUnreadable = unreadable
	report.OutOfScopePaths = outOfScope

	return report, nil
}

// callStatus invokes the statusUpdate callback if it is non-nil.
func callStatus(statusUpdate func(string), phase string) {
	if statusUpdate != nil {
		statusUpdate(phase)
	}
}

// FormatChangeReport produces a human-readable summary of a ChangeReport.
//
// Example output:
//
//	Project-wide operation complete (PREVIEW MODE)
//
//	Files scanned: 42
//	Files modified: 3
//	  - internal/ui/aichat.go (+5 -2)
//	  ...
//
// Req 7.4: "No files were modified." is shown when FilesModified is empty.
func FormatChangeReport(cr *ChangeReport) string {
	var sb strings.Builder

	// Header
	if cr.PreviewMode {
		sb.WriteString("Project-wide operation complete (PREVIEW MODE)\n")
	} else {
		sb.WriteString("Project-wide operation complete\n")
	}
	sb.WriteString("\n")

	// Files scanned
	sb.WriteString(fmt.Sprintf("Files scanned: %d\n", len(cr.FilesRead)))

	// Truncation warning
	if cr.TruncationWarning != "" {
		sb.WriteString(fmt.Sprintf("Warning: %s\n", cr.TruncationWarning))
	}

	// Files modified
	sb.WriteString(fmt.Sprintf("Files modified: %d\n", len(cr.FilesModified)))
	for _, f := range cr.FilesModified {
		sb.WriteString(fmt.Sprintf("  - %s (+%d -%d)\n", f.RelPath, f.LinesAdded, f.LinesRemoved))
	}

	// Patch failures
	if len(cr.PatchFailures) > 0 {
		sb.WriteString(fmt.Sprintf("\nPatch failures: %d\n", len(cr.PatchFailures)))
		for _, pf := range cr.PatchFailures {
			sb.WriteString(fmt.Sprintf("  - %s: %s\n", pf.Path, pf.Reason))
		}
	}

	// Hallucinated paths
	if len(cr.HallucinatedPaths) > 0 {
		sb.WriteString(fmt.Sprintf("\nHallucinated paths: %d\n", len(cr.HallucinatedPaths)))
		for _, p := range cr.HallucinatedPaths {
			sb.WriteString(fmt.Sprintf("  - %s\n", p))
		}
	}

	// Out-of-scope paths
	if len(cr.OutOfScopePaths) > 0 {
		sb.WriteString(fmt.Sprintf("\nOut-of-scope paths: %d\n", len(cr.OutOfScopePaths)))
		for _, p := range cr.OutOfScopePaths {
			sb.WriteString(fmt.Sprintf("  - %s\n", p))
		}
	}

	// Unreadable files
	if len(cr.FilesUnreadable) > 0 {
		sb.WriteString(fmt.Sprintf("\nUnreadable files: %d\n", len(cr.FilesUnreadable)))
		for _, p := range cr.FilesUnreadable {
			sb.WriteString(fmt.Sprintf("  - %s\n", p))
		}
	}

	// Req 7.4: no files modified
	if len(cr.FilesModified) == 0 {
		sb.WriteString("\nNo files were modified.\n")
	}

	return sb.String()
}
