package agentic

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/user/terminal-intelligence/internal/executor"
)

// AgenticProjectFixer orchestrates the project-wide agentic fixing workflow.
// It coordinates scanning, ranking, fixing, testing, and retry logic across
// the entire workspace.
type AgenticProjectFixer struct {
	aiClient     AIClient
	model        string
	fixParser    *FixParser
	executor     *executor.CommandExecutor
	logger       *ActionLogger
	tracker      *AttemptTracker
	snapshots    *FileSnapshotManager
	testRunner   *TestRunner
	langRegistry map[string]LanguageConfig
}

// NewAgenticProjectFixer creates a new AgenticProjectFixer with all internal
// components initialised and ready for use.
func NewAgenticProjectFixer(aiClient AIClient, model string, logger *ActionLogger) *AgenticProjectFixer {
	// Copy the default language registry so mutations don't affect the global.
	registry := make(map[string]LanguageConfig, len(defaultLanguageRegistry))
	for k, v := range defaultLanguageRegistry {
		registry[k] = v
	}

	return &AgenticProjectFixer{
		aiClient:     aiClient,
		model:        model,
		fixParser:    NewFixParser(),
		executor:     executor.NewCommandExecutor(),
		logger:       logger,
		tracker:      NewAttemptTracker(),
		snapshots:    NewFileSnapshotManager(),
		testRunner:   NewTestRunner(),
		langRegistry: registry,
	}
}

// buildAgenticPrompt composes the AI prompt for a fix attempt.
// It includes system instructions, the original ask, file contents (up to 2000
// lines per file), prior attempt summaries, current test failures, an
// instruction to try a different strategy, and the expected output format.
func (apf *AgenticProjectFixer) buildAgenticPrompt(
	session *FixSession,
	rankedFiles []string,
	lastTestResult *TestResult,
) string {
	var sb strings.Builder

	// 1. System instructions
	sb.WriteString("You are an agentic code fixer. Your job is to analyze code issues and generate SEARCH/REPLACE patches to fix them.\n\n")

	// 2. Original ask (always included, never changes)
	sb.WriteString("=== ORIGINAL ASK ===\n")
	sb.WriteString(session.OriginalAsk)
	sb.WriteString("\n\n")

	// 3. File contents: read up to 2000 lines per ranked file
	if len(rankedFiles) > 0 {
		sb.WriteString("=== FILES ===\n\n")
		for _, path := range rankedFiles {
			content, truncated, err := readUpToNLines(path, 2000)
			if err != nil {
				sb.WriteString(fmt.Sprintf("=== FILE: %s ===\n(unreadable: %s)\n\n", path, err.Error()))
				continue
			}
			sb.WriteString(fmt.Sprintf("=== FILE: %s ===\n", path))
			sb.WriteString(content)
			if !strings.HasSuffix(content, "\n") {
				sb.WriteString("\n")
			}
			if truncated {
				sb.WriteString("[TRUNCATED: file exceeds 2000 lines; only the first 2000 lines are shown]\n")
			}
			sb.WriteString("\n")
		}
		sb.WriteString("=== END OF FILES ===\n\n")
	}

	// 4. Prior attempt summary (if tracker has attempts)
	if apf.tracker.AttemptCount() > 0 {
		sb.WriteString("=== PRIOR ATTEMPTS ===\n")
		sb.WriteString(apf.tracker.GenerateSummary())
		sb.WriteString("\n\n")
	}

	// 5. Current test failures (if lastTestResult is not nil and ExitCode != 0)
	if lastTestResult != nil && lastTestResult.ExitCode != 0 {
		sb.WriteString("=== CURRENT TEST FAILURES ===\n")
		if lastTestResult.Stdout != "" {
			sb.WriteString("Stdout:\n")
			sb.WriteString(lastTestResult.Stdout)
			if !strings.HasSuffix(lastTestResult.Stdout, "\n") {
				sb.WriteString("\n")
			}
		}
		if lastTestResult.Stderr != "" {
			sb.WriteString("Stderr:\n")
			sb.WriteString(lastTestResult.Stderr)
			if !strings.HasSuffix(lastTestResult.Stderr, "\n") {
				sb.WriteString("\n")
			}
		}
		sb.WriteString(fmt.Sprintf("Exit code: %d\n", lastTestResult.ExitCode))
		if lastTestResult.TimedOut {
			sb.WriteString("(test execution timed out)\n")
		}
		sb.WriteString("\n")
	}

	// 6. Instruction to try different strategy
	sb.WriteString("Try a DIFFERENT approach than any previously attempted strategies.\n\n")

	// 7. Output format instructions (SEARCH/REPLACE format)
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
	sb.WriteString("You may include multiple SEARCH/REPLACE blocks per file. ")
	sb.WriteString("Only output sections for files that need changes or creation.\n")

	return sb.String()
}

// ProcessFixCommand is the single entry point called by AIChatPane for /fix commands.
// It orchestrates the full agentic loop: scan → rank → snapshot → fix → test → retry.
//
// Requirements: 1.1, 1.2, 1.3, 1.4, 1.5, 2.1, 2.2, 2.3, 2.4, 2.5, 3.1, 3.2, 3.3, 3.4,
// 4.1, 4.2, 4.3, 5.1, 5.2, 5.3, 5.4, 5.5, 6.1, 6.2, 6.3, 6.4, 6.5, 6.6,
// 7.1, 7.3, 7.4, 7.5, 8.1, 8.2, 8.3, 8.4, 9.3, 9.4, 10.1, 10.2, 10.3, 10.4, 10.5
func (apf *AgenticProjectFixer) ProcessFixCommand(
	request *FixSessionRequest,
	statusUpdate func(phase string),
) (*FixSessionResult, error) {
	// ── Step 1: Set defaults ─────────────────────────────────────────────────
	if request.MaxAttempts <= 0 {
		request.MaxAttempts = 9
	}
	if request.MaxCycles <= 0 {
		request.MaxCycles = 3
	}

	// ── Step 2: Validate request ─────────────────────────────────────────────
	if err := request.Validate(); err != nil {
		return nil, fmt.Errorf("invalid fix request: %w", err)
	}

	// ── Step 3: Create session ───────────────────────────────────────────────
	session := &FixSession{
		OriginalAsk:    request.Message,
		StartTime:      time.Now(),
		Attempts:       []FixAttempt{},
		Snapshots:      make(map[string][]byte),
		CurrentCycle:   0,
		AttemptInCycle: 0,
	}

	// Reset tracker for this session.
	apf.tracker = NewAttemptTracker()

	// Log session start (Req 6.1).
	apf.logger.Log("Fix session started: %s", request.Message)

	// ── Step 4: SCANNING PHASE ───────────────────────────────────────────────
	callStatus(statusUpdate, "scanning")
	apf.logger.Log("Scanning project files in %s", request.ProjectRoot)

	scanner := newFileScanner(request.ProjectRoot, 0)
	scannedPaths, _, scanErr := scanner.scan()
	if scanErr != nil {
		apf.logger.Log("Scan error: %s", scanErr.Error())
		return nil, fmt.Errorf("file scan failed: %w", scanErr)
	}

	// If OpenFilePath is set, ensure it's in the scanned paths (Req 1.3).
	if request.OpenFilePath != "" {
		absOpen, err := filepath.Abs(request.OpenFilePath)
		if err == nil {
			found := false
			for _, p := range scannedPaths {
				if p == absOpen {
					found = true
					break
				}
			}
			if !found {
				if _, statErr := os.Stat(absOpen); statErr == nil {
					scannedPaths = append(scannedPaths, absOpen)
				}
			}
		}
	}

	apf.logger.Log("Scanned %d files", len(scannedPaths))

	// ── Step 5: RANKING PHASE ────────────────────────────────────────────────
	callStatus(statusUpdate, "ranking")
	apf.logger.Log("Ranking files by relevance")

	ranker := newRelevanceRanker(apf.aiClient, apf.model)
	ranked, _, rankErr := ranker.rank(scannedPaths, request.Message, request.ProjectRoot, 20)
	if rankErr != nil {
		apf.logger.Log("Ranking error: %s", rankErr.Error())
		return nil, fmt.Errorf("relevance ranking failed: %w", rankErr)
	}

	if len(ranked) == 0 {
		apf.logger.Log("No relevant files found")
		return &FixSessionResult{
			Success:      false,
			ErrorMessage: "no relevant files found for the given fix description",
		}, nil
	}

	// If OpenFilePath is set and not in ranked list, prepend it (Req 1.3).
	if request.OpenFilePath != "" {
		absOpen, err := filepath.Abs(request.OpenFilePath)
		if err == nil {
			found := false
			for _, p := range ranked {
				if p == absOpen {
					found = true
					break
				}
			}
			if !found {
				if _, statErr := os.Stat(absOpen); statErr == nil {
					ranked = append([]string{absOpen}, ranked...)
				}
			}
		}
	}

	session.RankedFiles = ranked

	for _, f := range ranked {
		apf.logger.Log("Investigating file: %s", f)
	}

	// ── Step 6: SNAPSHOT PHASE ───────────────────────────────────────────────
	if err := apf.snapshots.Capture(ranked); err != nil {
		apf.logger.Log("Snapshot capture error: %s", err.Error())
		return nil, fmt.Errorf("failed to capture file snapshots: %w", err)
	}

	// ── Step 7: AGENTIC LOOP ─────────────────────────────────────────────────
	var lastTestResult *TestResult
	var lastModified []FileResult
	success := false

	for attempt := 1; attempt <= request.MaxAttempts; attempt++ {
		// (a) Check AI availability (Req 10.4, 10.5).
		available, aiErr := apf.aiClient.IsAvailable()
		if aiErr != nil || !available {
			apf.logger.Log("AI service unavailable, restoring files and terminating")
			failed := apf.snapshots.Restore()
			for _, f := range failed {
				apf.logger.Log("Failed to restore file: %s", f)
			}
			return &FixSessionResult{
				Success:       false,
				TotalAttempts: attempt - 1,
				TotalCycles:   session.CurrentCycle + 1,
				Attempts:      session.Attempts,
				ErrorMessage:  "AI service became unavailable during fix session",
			}, nil
		}

		// (b) Log attempt start (Req 6.6).
		callStatus(statusUpdate, fmt.Sprintf("fixing (attempt %d/%d, cycle %d)", attempt, request.MaxAttempts, session.CurrentCycle+1))
		apf.logger.Log("Attempt %d (cycle %d, attempt-in-cycle %d)", attempt, session.CurrentCycle+1, session.AttemptInCycle+1)

		// (c) Build prompt.
		prompt := apf.buildAgenticPrompt(session, ranked, lastTestResult)

		// (d) Call AI to generate response.
		responseChan, genErr := apf.aiClient.Generate(prompt, apf.model, nil, nil)
		if genErr != nil {
			apf.logger.Log("AI generation failed: %s", genErr.Error())
			// Record failed attempt and continue.
			fa := FixAttempt{
				Number:    attempt,
				Cycle:     session.CurrentCycle,
				Strategy:  Strategy{Description: "AI generation failed", Prompt: prompt, AIResponse: ""},
				Timestamp: time.Now(),
			}
			session.Attempts = append(session.Attempts, fa)
			apf.tracker.Record(fa)
			session.AttemptInCycle++

			if session.AttemptInCycle >= 3 {
				apf.handleResetCycle(session, request, statusUpdate)
			}
			continue
		}

		var sb strings.Builder
		for chunk := range responseChan {
			sb.WriteString(chunk)
		}
		aiResponse := sb.String()

		// (e) Handle empty AI response.
		if strings.TrimSpace(aiResponse) == "" {
			apf.logger.Log("AI returned empty response")
			fa := FixAttempt{
				Number:    attempt,
				Cycle:     session.CurrentCycle,
				Strategy:  Strategy{Description: "AI returned empty response", Prompt: prompt, AIResponse: ""},
				Timestamp: time.Now(),
			}
			session.Attempts = append(session.Attempts, fa)
			apf.tracker.Record(fa)
			session.AttemptInCycle++

			if session.AttemptInCycle >= 3 {
				apf.handleResetCycle(session, request, statusUpdate)
			}
			continue
		}

		// (f) Parse response into file sections.
		fileSections := splitOnFileHeaders(aiResponse)
		apf.logger.Log("Attempt %d: AI proposed changes to %d file(s)", attempt, len(fileSections))

		// Log a brief summary of what the AI is trying to do (first 200 chars of response).
		responseSummary := strings.TrimSpace(aiResponse)
		if len(responseSummary) > 200 {
			responseSummary = responseSummary[:200] + "..."
		}
		apf.logger.Log("Attempt %d: AI strategy summary: %s", attempt, responseSummary)

		// Resolve project root for path operations.
		absRoot, _ := filepath.Abs(request.ProjectRoot)
		rootPrefix := absRoot
		if !strings.HasSuffix(rootPrefix, string(filepath.Separator)) {
			rootPrefix += string(filepath.Separator)
		}

		// (g) Apply patches per file.
		var modified []FileResult
		var patchesApplied []searchReplacePatch
		var failures []PatchFailure

		for filePath, patchText := range fileSections {
			filePath = strings.TrimSpace(filePath)
			patches := parseSearchReplace(patchText)
			if len(patches) == 0 {
				apf.logger.Log("Attempt %d: no valid patches found for file: %s", attempt, filePath)
				continue
			}
			apf.logger.Log("Attempt %d: applying %d patch(es) to %s", attempt, len(patches), filePath)

			// Resolve the file path.
			absP := filePath
			if !filepath.IsAbs(absP) {
				absP = filepath.Join(absRoot, absP)
			}
			absP, _ = filepath.Abs(absP)

			// Validate path is within project root (Req 2.5).
			if !strings.HasPrefix(absP+string(filepath.Separator), rootPrefix) &&
				absP != strings.TrimSuffix(rootPrefix, string(filepath.Separator)) {
				continue
			}

			// Read current file content.
			content, _, readErr := readUpToNLines(absP, 2000)
			if readErr != nil {
				apf.logger.Log("Attempt %d: could not read file %s: %s", attempt, filePath, readErr.Error())
				failures = append(failures, PatchFailure{Path: filePath, Reason: readErr.Error()})
				continue
			}

			// Capture snapshot if not already captured.
			if !apf.snapshots.HasSnapshot(absP) {
				_ = apf.snapshots.Capture([]string{absP})
			}

			// Apply patches sequentially.
			currentContent := content
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
					apf.logger.Log("Attempt %d: patch failed on %s: %s", attempt, filePath, applyErr.Error())
					failures = append(failures, PatchFailure{Path: filePath, Reason: applyErr.Error()})
					applyFailed = true
					break
				}
				currentContent = newContent
				patchesApplied = append(patchesApplied, patch)
			}
			if applyFailed {
				continue
			}

			// Write modified content to disk.
			if err := os.MkdirAll(filepath.Dir(absP), 0755); err != nil {
				failures = append(failures, PatchFailure{Path: filePath, Reason: err.Error()})
				continue
			}
			if err := os.WriteFile(absP, []byte(currentContent), 0644); err != nil {
				failures = append(failures, PatchFailure{Path: filePath, Reason: err.Error()})
				continue
			}

			relPath, _ := filepath.Rel(absRoot, absP)
			origLines := strings.Split(content, "\n")
			newLines := strings.Split(currentContent, "\n")
			linesAdded := 0
			linesRemoved := 0
			if len(newLines) > len(origLines) {
				linesAdded = len(newLines) - len(origLines)
			} else {
				linesRemoved = len(origLines) - len(newLines)
			}

			fr := FileResult{
				Path:         absP,
				RelPath:      relPath,
				LinesAdded:   linesAdded,
				LinesRemoved: linesRemoved,
			}
			modified = append(modified, fr)
			apf.logger.Log("Modified file: %s (+%d -%d)", relPath, linesAdded, linesRemoved)
		}

		lastModified = modified

		// Log summary of patch application results.
		if len(failures) > 0 {
			apf.logger.Log("Attempt %d: %d patch failure(s) encountered", attempt, len(failures))
			for _, f := range failures {
				apf.logger.Log("  Patch failure: %s — %s", f.Path, f.Reason)
			}
		}
		if len(modified) == 0 && len(failures) > 0 {
			apf.logger.Log("Attempt %d: no files were successfully modified — all patches failed", attempt)
		} else if len(modified) > 0 {
			apf.logger.Log("Attempt %d: successfully modified %d file(s)", attempt, len(modified))
		}

		// (h) Detect language and get test command (Req 9.4).
		modifiedPaths := make([]string, len(modified))
		for i, m := range modified {
			modifiedPaths[i] = m.Path
		}
		lang := detectLanguage(modifiedPaths)
		if lang == "" {
			// Fall back to detecting from all ranked files.
			lang = detectLanguage(ranked)
		}
		testCmd := getTestCommand(lang)

		// (i) Run tests (Req 7.1).
		var testResult *TestResult
		if testCmd != "" {
			callStatus(statusUpdate, fmt.Sprintf("testing (attempt %d)", attempt))
			apf.logger.Log("Running tests: %s", testCmd)
			testResult = apf.testRunner.Run(testCmd, request.ProjectRoot)
			apf.logger.Log("Test result: exit code %d (duration: %s)", testResult.ExitCode, testResult.Duration)
		} else {
			apf.logger.Log("No test command available for detected language")
			// Treat as success if no tests can be run (Req 7.5).
			testResult = &TestResult{ExitCode: 0, Stdout: "no test command available"}
		}

		lastTestResult = testResult

		// (j) Record attempt.
		fa := FixAttempt{
			Number:         attempt,
			Cycle:          session.CurrentCycle,
			Strategy:       Strategy{Description: fmt.Sprintf("Attempt %d fix", attempt), Prompt: prompt, AIResponse: aiResponse},
			FilesModified:  modified,
			PatchesApplied: patchesApplied,
			TestCommand:    testCmd,
			TestResult:     testResult,
			Timestamp:      time.Now(),
		}
		session.Attempts = append(session.Attempts, fa)
		apf.tracker.Record(fa)

		// (k) Check test result (Req 7.3, 7.4).
		if testResult.ExitCode == 0 {
			apf.logger.Log("Attempt %d: tests passed — fix successful", attempt)
			success = true
			break
		}

		// (l) Tests failed — log details so the user can see what went wrong.
		apf.logger.Log("Attempt %d: tests failed (exit code %d)", attempt, testResult.ExitCode)
		if testResult.Stderr != "" {
			stderrSnippet := testResult.Stderr
			if len(stderrSnippet) > 500 {
				stderrSnippet = stderrSnippet[:500] + "...(truncated)"
			}
			apf.logger.Log("Attempt %d: stderr: %s", attempt, stderrSnippet)
		}
		if testResult.Stdout != "" {
			stdoutSnippet := testResult.Stdout
			if len(stdoutSnippet) > 500 {
				stdoutSnippet = stdoutSnippet[:500] + "...(truncated)"
			}
			apf.logger.Log("Attempt %d: stdout: %s", attempt, stdoutSnippet)
		}
		session.AttemptInCycle++

		// (m) Check for reset cycle (Req 5.1).
		if session.AttemptInCycle >= 3 {
			if session.CurrentCycle+1 >= request.MaxCycles {
				apf.logger.Log("Maximum cycles (%d) exhausted", request.MaxCycles)
				break
			}
			apf.handleResetCycle(session, request, statusUpdate)
		}
	}

	// ── Step 8: Build result ─────────────────────────────────────────────────
	result := &FixSessionResult{
		Success:       success,
		TotalAttempts: len(session.Attempts),
		TotalCycles:   session.CurrentCycle + 1,
		Attempts:      session.Attempts,
	}

	if success {
		apf.logger.Log("Fix session completed successfully after %d attempts", len(session.Attempts))
		apf.snapshots.Discard()

		// Build final report from last successful modifications.
		result.FinalReport = &ChangeReport{
			FilesRead:     ranked,
			FilesModified: lastModified,
		}
	} else {
		// Restore files on failure (Req 8.2).
		failed := apf.snapshots.Restore()
		for _, f := range failed {
			apf.logger.Log("Failed to restore file: %s", f)
		}

		result.ErrorMessage = fmt.Sprintf("fix session exhausted after %d attempts across %d cycles; manual intervention needed",
			len(session.Attempts), session.CurrentCycle+1)
		apf.logger.Log("Fix session failed: %s", result.ErrorMessage)
	}

	return result, nil
}

// handleResetCycle performs a reset cycle: restores snapshots, increments cycle,
// resets attempt-in-cycle counter, and logs the event.
func (apf *AgenticProjectFixer) handleResetCycle(
	session *FixSession,
	request *FixSessionRequest,
	statusUpdate func(phase string),
) {
	callStatus(statusUpdate, "resetting")
	apf.logger.Log("Reset cycle triggered: 3 consecutive failures in cycle %d", session.CurrentCycle+1)

	// Restore all files to snapshot state (Req 5.2).
	failed := apf.snapshots.Restore()
	for _, f := range failed {
		apf.logger.Log("Failed to restore file during reset: %s", f)
	}

	session.CurrentCycle++
	session.AttemptInCycle = 0

	apf.logger.Log("Starting cycle %d of %d", session.CurrentCycle+1, request.MaxCycles)
}
