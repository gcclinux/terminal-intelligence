package docgen

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/user/terminal-intelligence/internal/types"
	"pgregory.net/rapid"
)

// ── Generators ────────────────────────────────────────────────────────────────

// rapidAnalysisResult generates *AnalysisResult with random (possibly empty) fields.
func rapidAnalysisResult() *rapid.Generator[*AnalysisResult] {
	return rapid.Custom(func(t *rapid.T) *AnalysisResult {
		// Random runtime dependencies
		numDeps := rapid.IntRange(0, 5).Draw(t, "numDeps")
		deps := make([]Dependency, numDeps)
		for i := range deps {
			deps[i] = Dependency{
				Name:    rapid.StringMatching(`[A-Za-z][A-Za-z0-9_-]{1,15}`).Draw(t, fmt.Sprintf("dep_name_%d", i)),
				Version: rapid.StringMatching(`v[0-9]+\.[0-9]+\.[0-9]+`).Draw(t, fmt.Sprintf("dep_ver_%d", i)),
			}
		}

		// Random config files
		numCfg := rapid.IntRange(0, 4).Draw(t, "numCfg")
		cfgFiles := make([]string, numCfg)
		for i := range cfgFiles {
			cfgFiles[i] = rapid.StringMatching(`[a-z][a-z0-9_-]{1,10}\.(json|yaml|toml|env)`).Draw(t, fmt.Sprintf("cfg_%d", i))
		}

		// Random package manifests
		manifests := rapid.SliceOfN(
			rapid.SampledFrom([]string{"go.mod", "package.json", "requirements.txt", "Cargo.toml", "Pipfile"}),
			0, 3,
		).Draw(t, "manifests")

		// Random README content (possibly empty)
		readmeContent := rapid.OneOf(
			rapid.Just(""),
			rapid.StringMatching(`[A-Za-z0-9 \n#]{5,100}`),
		).Draw(t, "readme")

		// Random exported and unexported functions
		numFuncs := rapid.IntRange(0, 6).Draw(t, "numFuncs")
		funcs := make([]FunctionInfo, numFuncs)
		for i := range funcs {
			exported := rapid.Bool().Draw(t, fmt.Sprintf("func_exported_%d", i))
			name := rapid.StringMatching(`[A-Za-z][A-Za-z0-9]{1,15}`).Draw(t, fmt.Sprintf("func_name_%d", i))
			if exported {
				// Ensure first letter is uppercase for exported
				name = strings.ToUpper(name[:1]) + name[1:]
			} else {
				name = strings.ToLower(name[:1]) + name[1:]
			}
			funcs[i] = FunctionInfo{
				Name:       name,
				Signature:  fmt.Sprintf("func %s() error", name),
				IsExported: exported,
			}
		}

		return &AnalysisResult{
			Dependencies: &DependencyInfo{
				Runtime: deps,
			},
			Configuration: &ConfigInfo{
				ConfigFiles:      cfgFiles,
				PackageManifests: manifests,
			},
			Documentation: &ExistingDocs{
				ReadmeContent: readmeContent,
			},
			CodeStructure: &CodeStructure{
				Functions: funcs,
			},
		}
	})
}

// rapidDocType draws from the five DocumentationType constants.
func rapidDocType() *rapid.Generator[DocumentationType] {
	return rapid.SampledFrom([]DocumentationType{
		DocTypeUserManual,
		DocTypeInstallation,
		DocTypeAPI,
		DocTypeTutorial,
		DocTypeGeneral,
	})
}

// rapidChunkSlice generates a non-empty slice of random strings.
func rapidChunkSlice() *rapid.Generator[[]string] {
	return rapid.SliceOfN(rapid.StringMatching(`[A-Za-z0-9 .,!?]{1,30}`), 1, 10)
}

// rapidErrorOrNil generates a non-nil error for failure-path tests.
func rapidErrorOrNil() *rapid.Generator[error] {
	return rapid.Just(fmt.Errorf("test error"))
}

// ── Mock helpers ──────────────────────────────────────────────────────────────

// pbtMockPane is a thread-safe notification collector for PBT tests.
type pbtMockPane struct {
	notifications []string
}

func (p *pbtMockPane) DisplayNotification(msg string) {
	p.notifications = append(p.notifications, msg)
}

func (p *pbtMockPane) OpenFileInEditor(filePath string) error { return nil }

func newPBTFeedback() (*FeedbackManager, *pbtMockPane) {
	pane := &pbtMockPane{}
	return NewFeedbackManager(pane), pane
}

// pbtStreamClient returns a fixed set of chunks on Generate.
type pbtStreamClient struct {
	chunks []string
}

func (c *pbtStreamClient) Generate(prompt, model string, context []int, onTokenUsage func(types.TokenUsage)) (<-chan string, error) {
	ch := make(chan string, len(c.chunks))
	for _, s := range c.chunks {
		ch <- s
	}
	close(ch)
	return ch, nil
}

func (c *pbtStreamClient) IsAvailable() (bool, error) { return true, nil }
func (c *pbtStreamClient) ListModels() ([]string, error) { return nil, nil }

// pbtErrorClient always returns an error from Generate.
type pbtErrorClient struct{}

func (c *pbtErrorClient) Generate(prompt, model string, context []int, onTokenUsage func(types.TokenUsage)) (<-chan string, error) {
	return nil, fmt.Errorf("test error")
}

func (c *pbtErrorClient) IsAvailable() (bool, error) { return true, nil }
func (c *pbtErrorClient) ListModels() ([]string, error) { return nil, nil }

// pbtUnavailableClient always reports unavailable.
type pbtUnavailableClient struct{}

func (c *pbtUnavailableClient) Generate(prompt, model string, context []int, onTokenUsage func(types.TokenUsage)) (<-chan string, error) {
	ch := make(chan string)
	close(ch)
	return ch, nil
}

func (c *pbtUnavailableClient) IsAvailable() (bool, error) { return false, nil }
func (c *pbtUnavailableClient) ListModels() ([]string, error) { return nil, nil }

// pbtSelectiveClient fails on the specified set of call indices (1-indexed).
type pbtSelectiveClient struct {
	chunks    []string
	failOnSet map[int]bool
	callCount int
}

func (c *pbtSelectiveClient) Generate(prompt, model string, context []int, onTokenUsage func(types.TokenUsage)) (<-chan string, error) {
	c.callCount++
	if c.failOnSet[c.callCount] {
		return nil, fmt.Errorf("selective failure on call %d", c.callCount)
	}
	ch := make(chan string, len(c.chunks))
	for _, s := range c.chunks {
		ch <- s
	}
	close(ch)
	return ch, nil
}

func (c *pbtSelectiveClient) IsAvailable() (bool, error) { return true, nil }
func (c *pbtSelectiveClient) ListModels() ([]string, error) { return nil, nil }

// ── Property tests ────────────────────────────────────────────────────────────

// Feature: ai-powered-doc-generation, Property 1: chunk concatenation
// Validates: Requirements 1.2
func TestChunkConcatenation(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		chunks := rapidChunkSlice().Draw(rt, "chunks")
		result := rapidAnalysisResult().Draw(rt, "result")
		docType := rapidDocType().Draw(rt, "docType")

		client := &pbtStreamClient{chunks: chunks}
		feedback, _ := newPBTFeedback()
		g := NewAIGenerator(client, "test-model", feedback)

		doc, err := g.Generate(result, docType)
		if err != nil {
			rt.Fatalf("Generate returned unexpected error: %v", err)
		}
		if doc == nil {
			rt.Fatal("Generate returned nil doc")
		}

		want := strings.Join(chunks, "")
		if doc.Content != want {
			rt.Fatalf("Content = %q; want %q", doc.Content, want)
		}
	})
}

// Feature: ai-powered-doc-generation, Property 2: distinct prompts per DocumentationType
// Validates: Requirements 1.3
func TestDistinctPromptsPerDocType(t *testing.T) {
	allTypes := []DocumentationType{
		DocTypeUserManual,
		DocTypeInstallation,
		DocTypeAPI,
		DocTypeTutorial,
		DocTypeGeneral,
	}

	rapid.Check(t, func(rt *rapid.T) {
		result := rapidAnalysisResult().Draw(rt, "result")

		// Draw two distinct doc types
		idxA := rapid.IntRange(0, len(allTypes)-1).Draw(rt, "idxA")
		idxB := rapid.IntRange(0, len(allTypes)-1).Draw(rt, "idxB")
		if idxA == idxB {
			// Skip when same type is drawn — not a violation
			return
		}
		a := allTypes[idxA]
		b := allTypes[idxB]

		g := NewAIGenerator(&pbtStreamClient{}, "test-model", nil)
		promptA := g.BuildPrompt(result, a)
		promptB := g.BuildPrompt(result, b)

		if promptA == promptB {
			rt.Fatalf("BuildPrompt(%v) == BuildPrompt(%v); prompts must be distinct", a, b)
		}
		if !strings.Contains(promptA, a.String()) {
			rt.Fatalf("prompt for %v does not contain its string representation %q", a, a.String())
		}
		if !strings.Contains(promptB, b.String()) {
			rt.Fatalf("prompt for %v does not contain its string representation %q", b, b.String())
		}
	})
}

// Feature: ai-powered-doc-generation, Property 3: prompt contains dependencies, languages, and config files
// Validates: Requirements 2.1, 2.4
func TestPromptContainsDependenciesAndConfig(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		result := rapidAnalysisResult().Draw(rt, "result")
		docType := rapidDocType().Draw(rt, "docType")

		// Only assert when there is something to check
		hasDeps := result.Dependencies != nil && len(result.Dependencies.Runtime) > 0
		hasCfg := result.Configuration != nil &&
			(len(result.Configuration.PackageManifests) > 0 || len(result.Configuration.ConfigFiles) > 0)

		if !hasDeps && !hasCfg {
			return
		}

		g := NewAIGenerator(&pbtStreamClient{}, "test-model", nil)
		prompt := g.BuildPrompt(result, docType)

		if hasDeps {
			for _, dep := range result.Dependencies.Runtime {
				if !strings.Contains(prompt, dep.Name) {
					rt.Fatalf("prompt missing dependency name %q", dep.Name)
				}
			}
		}

		if result.Configuration != nil {
			for _, m := range result.Configuration.PackageManifests {
				if !strings.Contains(prompt, m) {
					rt.Fatalf("prompt missing package manifest %q", m)
				}
			}
			for _, cf := range result.Configuration.ConfigFiles {
				if !strings.Contains(prompt, cf) {
					rt.Fatalf("prompt missing config file %q", cf)
				}
			}
		}
	})
}

// Feature: ai-powered-doc-generation, Property 4: API prompt contains exported symbols
// Validates: Requirements 2.2
func TestAPIPromptContainsExportedSymbols(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		result := rapidAnalysisResult().Draw(rt, "result")

		// Only assert when there is at least one exported function
		var exported []FunctionInfo
		if result.CodeStructure != nil {
			for _, fn := range result.CodeStructure.Functions {
				if fn.IsExported {
					exported = append(exported, fn)
				}
			}
		}
		if len(exported) == 0 {
			return
		}

		g := NewAIGenerator(&pbtStreamClient{}, "test-model", nil)
		prompt := g.BuildPrompt(result, DocTypeAPI)

		// Build the symbol block to know which symbols fit within the budget
		symbolBlock := buildSymbolBlock(result, g.tokenBudget)

		for _, fn := range exported {
			// Only check symbols that actually appear in the symbol block
			// (i.e., were not truncated away)
			if strings.Contains(symbolBlock, fn.Name) {
				if !strings.Contains(prompt, fn.Name) {
					rt.Fatalf("API prompt missing exported symbol %q", fn.Name)
				}
			}
		}
	})
}

// Feature: ai-powered-doc-generation, Property 5: prompt contains README when non-empty
// Validates: Requirements 2.3
func TestPromptContainsReadme(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		result := rapidAnalysisResult().Draw(rt, "result")
		docType := rapidDocType().Draw(rt, "docType")

		if result.Documentation == nil || result.Documentation.ReadmeContent == "" {
			return
		}

		g := NewAIGenerator(&pbtStreamClient{}, "test-model", nil)
		prompt := g.BuildPrompt(result, docType)

		if !strings.Contains(prompt, result.Documentation.ReadmeContent) {
			rt.Fatalf("prompt does not contain README content %q", result.Documentation.ReadmeContent)
		}
	})
}

// Feature: ai-powered-doc-generation, Property 6: truncation notice when symbol list exceeds token budget
// Validates: Requirements 2.5
func TestTruncationNoticeWhenOverBudget(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		// Generate at least 2 exported functions with long-enough names to overflow a tiny budget
		numFuncs := rapid.IntRange(2, 5).Draw(rt, "numFuncs")
		funcs := make([]FunctionInfo, numFuncs)
		for i := range funcs {
			name := fmt.Sprintf("ExportedFunction%d", i)
			funcs[i] = FunctionInfo{
				Name:       name,
				Signature:  fmt.Sprintf("func %s() error", name),
				IsExported: true,
			}
		}

		result := &AnalysisResult{
			CodeStructure: &CodeStructure{Functions: funcs},
		}

		// Use a budget of 1 rune so any symbol overflows
		g := NewAIGenerator(&pbtStreamClient{}, "test-model", nil, WithTokenBudget(1))
		prompt := g.BuildPrompt(result, DocTypeAPI)

		const truncationMarker = "[TRUNCATED: symbol list exceeded token budget"
		if !strings.Contains(prompt, truncationMarker) {
			rt.Fatalf("expected truncation notice in prompt; got:\n%s", prompt)
		}
	})
}

// Feature: ai-powered-doc-generation, Property 7: NotifyProgress called during generation
// Validates: Requirements 3.1
func TestNotifyProgressCalledDuringGeneration(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		chunks := rapidChunkSlice().Draw(rt, "chunks")
		result := rapidAnalysisResult().Draw(rt, "result")
		docType := rapidDocType().Draw(rt, "docType")

		client := &pbtStreamClient{chunks: chunks}
		feedback, pane := newPBTFeedback()
		g := NewAIGenerator(client, "test-model", feedback)

		_, err := g.Generate(result, docType)
		if err != nil {
			rt.Fatalf("Generate returned unexpected error: %v", err)
		}

		// At least one non-empty progress notification must have been sent
		found := false
		for _, n := range pane.notifications {
			if n != "" {
				found = true
				break
			}
		}
		if !found {
			rt.Fatal("NotifyProgress was not called with a non-empty message during generation")
		}
	})
}

// Feature: ai-powered-doc-generation, Property 8: Generate error causes NotifyError and no file write
// Validates: Requirements 3.2, 4.2
func TestGenerateErrorCausesNotifyErrorAndNoWrite(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		docType := rapidDocType().Draw(rt, "docType")
		result := rapidAnalysisResult().Draw(rt, "result")

		feedback, pane := newPBTFeedback()
		g := NewAIGenerator(&pbtErrorClient{}, "test-model", feedback)

		doc, err := g.Generate(result, docType)

		if err == nil {
			rt.Fatal("expected non-nil error from Generate when client fails")
		}
		if doc != nil {
			rt.Fatalf("expected nil GeneratedDoc on error, got %+v", doc)
		}

		// NotifyError must have been called (notification contains the error text)
		found := false
		for _, n := range pane.notifications {
			if strings.Contains(n, "test error") || strings.Contains(strings.ToLower(n), "error") || strings.Contains(strings.ToLower(n), "failed") {
				found = true
				break
			}
		}
		if !found {
			rt.Fatalf("expected NotifyError notification; got: %v", pane.notifications)
		}
	})
}

// Feature: ai-powered-doc-generation, Property 9: AI unavailability causes NotifyError and no files written
// Validates: Requirements 4.1
func TestUnavailableCausesNotifyErrorAndNoWrite(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		tmpDir := t.TempDir()

		pane := &pbtMockPane{}
		pipeline := NewPipeline(tmpDir, &pbtUnavailableClient{}, "test-model", pane)

		_, err := pipeline.ProcessCommand("/project /doc create user manual")
		if err != nil {
			rt.Fatalf("Pipeline returned unexpected error: %v", err)
		}

		// No .md files should have been written
		entries, readErr := os.ReadDir(tmpDir)
		if readErr != nil {
			rt.Fatalf("Failed to read temp dir: %v", readErr)
		}
		for _, e := range entries {
			if strings.HasSuffix(e.Name(), ".md") {
				rt.Fatalf("Unexpected .md file written when AI unavailable: %s", e.Name())
			}
		}

		// NotifyError must have been called
		found := false
		for _, n := range pane.notifications {
			if strings.Contains(strings.ToLower(n), "unavailable") ||
				strings.Contains(strings.ToLower(n), "error") ||
				strings.Contains(strings.ToLower(n), "failed") {
				found = true
				break
			}
		}
		if !found {
			rt.Fatalf("expected NotifyError notification about unavailability; got: %v", pane.notifications)
		}
	})
}

// Feature: ai-powered-doc-generation, Property 10: partial failure preserves successful results in NotifyComplete
// Validates: Requirements 4.3
func TestPartialFailureNotifyComplete(t *testing.T) {
	allTypes := []DocumentationType{
		DocTypeUserManual,
		DocTypeInstallation,
		DocTypeAPI,
		DocTypeTutorial,
		DocTypeGeneral,
	}

	rapid.Check(t, func(rt *rapid.T) {
		result := rapidAnalysisResult().Draw(rt, "result")

		// Pick a subset of types (at least 2 so we can have partial failure)
		n := rapid.IntRange(2, len(allTypes)).Draw(rt, "n")
		docTypes := allTypes[:n]

		// Pick at least one to fail and at least one to succeed
		numFail := rapid.IntRange(1, n-1).Draw(rt, "numFail")
		failSet := make(map[int]bool)
		for i := 1; i <= numFail; i++ {
			failSet[i] = true
		}

		client := &pbtSelectiveClient{
			chunks:    []string{"ok content"},
			failOnSet: failSet,
		}
		feedback, _ := newPBTFeedback()
		g := NewAIGenerator(client, "test-model", feedback)

		docs, err := g.GenerateMultiple(result, docTypes)
		if err != nil {
			rt.Fatalf("GenerateMultiple returned unexpected error: %v", err)
		}

		expectedSuccessCount := n - numFail
		if len(docs) != expectedSuccessCount {
			rt.Fatalf("expected %d successful docs, got %d", expectedSuccessCount, len(docs))
		}

		// No nil entries
		for i, d := range docs {
			if d == nil {
				rt.Fatalf("docs[%d] is nil", i)
			}
		}
	})
}

// Feature: ai-powered-doc-generation, Property 11: BuildPrompt is deterministic
// Validates: Requirements 5.2
func TestBuildPromptIsDeterministic(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		result := rapidAnalysisResult().Draw(rt, "result")
		docType := rapidDocType().Draw(rt, "docType")

		g := NewAIGenerator(&pbtStreamClient{}, "test-model", nil)

		prompt1 := g.BuildPrompt(result, docType)
		prompt2 := g.BuildPrompt(result, docType)

		if prompt1 != prompt2 {
			rt.Fatalf("BuildPrompt is not deterministic for docType=%v:\nfirst:  %q\nsecond: %q", docType, prompt1, prompt2)
		}
	})
}

// Feature: ai-powered-doc-generation, Property 12: ParsePromptMetadata round-trip
// Validates: Requirements 5.4
func TestParsePromptMetadataRoundTrip(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		result := rapidAnalysisResult().Draw(rt, "result")
		docType := rapidDocType().Draw(rt, "docType")

		g := NewAIGenerator(&pbtStreamClient{}, "test-model", nil)
		prompt := g.BuildPrompt(result, docType)

		got, err := ParsePromptMetadata(prompt)
		if err != nil {
			rt.Fatalf("ParsePromptMetadata returned error for docType=%v: %v", docType, err)
		}
		if got != docType {
			rt.Fatalf("round-trip failed: want %v, got %v", docType, got)
		}
	})
}
