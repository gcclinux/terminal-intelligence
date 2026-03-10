package docgen

import (
	"fmt"
	"strings"
	"testing"
)

// ── helpers ───────────────────────────────────────────────────────────────────

// mockChatPane captures DisplayNotification calls so tests can inspect them.
type mockChatPane struct {
	notifications []string
}

func (m *mockChatPane) DisplayNotification(msg string) {
	m.notifications = append(m.notifications, msg)
}

func (m *mockChatPane) OpenFileInEditor(filePath string) error { return nil }

// newTestFeedback returns a FeedbackManager backed by a mockChatPane.
// The pane pointer is returned so callers can inspect captured notifications.
func newTestFeedback() (*FeedbackManager, *mockChatPane) {
	pane := &mockChatPane{}
	return NewFeedbackManager(pane), pane
}

// ── BuildPrompt tests ─────────────────────────────────────────────────────────

// TestBuildPrompt_EmptyAnalysisResult verifies that an empty AnalysisResult
// still produces a non-empty prompt that contains the DOC_TYPE: header.
// Requirements: 2.2, 5.1
func TestBuildPrompt_EmptyAnalysisResult(t *testing.T) {
	g := NewAIGenerator(&MockAIClient{}, "test-model", nil)

	prompt := g.BuildPrompt(&AnalysisResult{}, DocTypeGeneral)

	if prompt == "" {
		t.Fatal("expected non-empty prompt for empty AnalysisResult")
	}
	if !strings.Contains(prompt, "DOC_TYPE:") {
		t.Errorf("prompt missing DOC_TYPE: header; got:\n%s", prompt)
	}
}

// TestBuildPrompt_NilAnalysisResult verifies that a nil AnalysisResult also
// produces a valid prompt with the DOC_TYPE: header.
// Requirements: 5.1
func TestBuildPrompt_NilAnalysisResult(t *testing.T) {
	g := NewAIGenerator(&MockAIClient{}, "test-model", nil)

	prompt := g.BuildPrompt(nil, DocTypeGeneral)

	if prompt == "" {
		t.Fatal("expected non-empty prompt for nil AnalysisResult")
	}
	if !strings.Contains(prompt, "DOC_TYPE:") {
		t.Errorf("prompt missing DOC_TYPE: header; got:\n%s", prompt)
	}
}

// TestBuildPrompt_DocTypeAPI_IncludesAPISymbolsSection verifies that a
// DocTypeAPI prompt contains the "API SYMBOLS" section header.
// Requirements: 2.2
func TestBuildPrompt_DocTypeAPI_IncludesAPISymbolsSection(t *testing.T) {
	g := NewAIGenerator(&MockAIClient{}, "test-model", nil)

	prompt := g.BuildPrompt(&AnalysisResult{}, DocTypeAPI)

	if !strings.Contains(prompt, "API SYMBOLS") {
		t.Errorf("DocTypeAPI prompt missing API SYMBOLS section; got:\n%s", prompt)
	}
}

// TestBuildPrompt_NonAPITypes_ExcludeAPISymbolsSection verifies that non-API
// doc types do NOT include the "API SYMBOLS" section.
// Requirements: 2.2
func TestBuildPrompt_NonAPITypes_ExcludeAPISymbolsSection(t *testing.T) {
	g := NewAIGenerator(&MockAIClient{}, "test-model", nil)

	nonAPITypes := []DocumentationType{
		DocTypeUserManual,
		DocTypeInstallation,
		DocTypeTutorial,
		DocTypeGeneral,
	}

	for _, dt := range nonAPITypes {
		prompt := g.BuildPrompt(&AnalysisResult{}, dt)
		if strings.Contains(prompt, "API SYMBOLS") {
			t.Errorf("doc type %q prompt unexpectedly contains API SYMBOLS section", dt.String())
		}
	}
}

// TestBuildPrompt_TruncationNotice verifies that when the symbol list exceeds
// the token budget, the truncation notice is appended.
// Requirements: 2.5
func TestBuildPrompt_TruncationNotice(t *testing.T) {
	// Use a tiny token budget so even a single short symbol overflows.
	g := NewAIGenerator(&MockAIClient{}, "test-model", nil, WithTokenBudget(1))

	result := &AnalysisResult{
		CodeStructure: &CodeStructure{
			Functions: []FunctionInfo{
				{Name: "ExportedFunc", Signature: "func ExportedFunc() error", IsExported: true},
				{Name: "AnotherFunc", Signature: "func AnotherFunc() string", IsExported: true},
			},
		},
	}

	prompt := g.BuildPrompt(result, DocTypeAPI)

	const truncationMarker = "[TRUNCATED: symbol list exceeded token budget"
	if !strings.Contains(prompt, truncationMarker) {
		t.Errorf("expected truncation notice in prompt; got:\n%s", prompt)
	}
}

// ── ParsePromptMetadata tests ─────────────────────────────────────────────────

// TestParsePromptMetadata_MissingHeader verifies that a prompt without a
// DOC_TYPE: header returns an error.
// Requirements: 5.3
func TestParsePromptMetadata_MissingHeader(t *testing.T) {
	_, err := ParsePromptMetadata("This prompt has no DOC_TYPE header at all.")
	if err == nil {
		t.Error("expected error for prompt without DOC_TYPE: header, got nil")
	}
}

// TestParsePromptMetadata_EmptyString verifies that an empty string returns an error.
// Requirements: 5.3
func TestParsePromptMetadata_EmptyString(t *testing.T) {
	_, err := ParsePromptMetadata("")
	if err == nil {
		t.Error("expected error for empty prompt, got nil")
	}
}

// TestParsePromptMetadata_RoundTrip verifies that ParsePromptMetadata correctly
// recovers the DocumentationType from a prompt produced by BuildPrompt.
// Requirements: 5.3
func TestParsePromptMetadata_RoundTrip(t *testing.T) {
	g := NewAIGenerator(&MockAIClient{}, "test-model", nil)

	docTypes := []DocumentationType{
		DocTypeUserManual,
		DocTypeInstallation,
		DocTypeAPI,
		DocTypeTutorial,
		DocTypeGeneral,
	}

	for _, dt := range docTypes {
		prompt := g.BuildPrompt(&AnalysisResult{}, dt)
		got, err := ParsePromptMetadata(prompt)
		if err != nil {
			t.Errorf("ParsePromptMetadata(%q) returned error: %v", dt.String(), err)
			continue
		}
		if got != dt {
			t.Errorf("round-trip failed: want %v, got %v", dt, got)
		}
	}
}

// ── NewAIGenerator tests ──────────────────────────────────────────────────────

// TestNewAIGenerator_EmptyModelUsesDefault verifies that when an empty model
// string is provided, DefaultModel is used and NotifyProgress is called.
// Requirements: 6.4
func TestNewAIGenerator_EmptyModelUsesDefault(t *testing.T) {
	feedback, pane := newTestFeedback()

	g := NewAIGenerator(&MockAIClient{}, "", feedback)

	if g.model != DefaultModel {
		t.Errorf("expected model %q, got %q", DefaultModel, g.model)
	}

	// NotifyProgress must have been called at least once with a non-empty message.
	if len(pane.notifications) == 0 {
		t.Error("expected NotifyProgress to be called when model is empty, but no notifications were recorded")
	}

	// At least one notification should mention the default model.
	found := false
	for _, n := range pane.notifications {
		if strings.Contains(n, DefaultModel) {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected a notification mentioning %q; got: %v", DefaultModel, pane.notifications)
	}
}

// TestNewAIGenerator_NonEmptyModelPreserved verifies that a non-empty model
// string is kept as-is and NotifyProgress is NOT called for the fallback.
// Requirements: 6.4
func TestNewAIGenerator_NonEmptyModelPreserved(t *testing.T) {
	feedback, pane := newTestFeedback()

	const customModel = "my-custom-model"
	g := NewAIGenerator(&MockAIClient{}, customModel, feedback)

	if g.model != customModel {
		t.Errorf("expected model %q, got %q", customModel, g.model)
	}

	// No fallback notification should have been emitted.
	for _, n := range pane.notifications {
		if strings.Contains(n, DefaultModel) {
			t.Errorf("unexpected notification mentioning DefaultModel when custom model was provided: %q", n)
		}
	}
}

// ── Generate tests ────────────────────────────────────────────────────────────

// mockStreamClient is an AIClient whose Generate returns a fixed slice of chunks.
type mockStreamClient struct {
	chunks []string
	err    error // returned directly from Generate (not on channel)
}

func (m *mockStreamClient) Generate(prompt string, model string, context []int) (<-chan string, error) {
	if m.err != nil {
		return nil, m.err
	}
	ch := make(chan string, len(m.chunks))
	for _, c := range m.chunks {
		ch <- c
	}
	close(ch)
	return ch, nil
}

func (m *mockStreamClient) IsAvailable() (bool, error) { return true, nil }
func (m *mockStreamClient) ListModels() ([]string, error) { return nil, nil }

// TestGenerate_TwoChunkStream verifies that Generate concatenates two chunks
// into a single Content string.
// Requirements: 1.1, 1.2
func TestGenerate_TwoChunkStream(t *testing.T) {
	client := &mockStreamClient{chunks: []string{"Hello, ", "world!"}}
	feedback, _ := newTestFeedback()
	g := NewAIGenerator(client, "test-model", feedback)

	doc, err := g.Generate(&AnalysisResult{}, DocTypeGeneral)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if doc == nil {
		t.Fatal("expected non-nil GeneratedDoc")
	}

	const want = "Hello, world!"
	if doc.Content != want {
		t.Errorf("Content = %q; want %q", doc.Content, want)
	}
	if doc.Type != DocTypeGeneral {
		t.Errorf("Type = %v; want DocTypeGeneral", doc.Type)
	}
	if doc.Filename == "" {
		t.Error("expected non-empty Filename")
	}
}

// TestGenerate_ClientError verifies that when AIClient.Generate returns an
// error, Generate propagates it and calls feedback.NotifyError.
// Requirements: 3.2, 4.2
func TestGenerate_ClientError(t *testing.T) {
	wantErr := fmt.Errorf("AI unavailable")
	client := &mockStreamClient{err: wantErr}
	feedback, pane := newTestFeedback()
	g := NewAIGenerator(client, "test-model", feedback)

	doc, err := g.Generate(&AnalysisResult{}, DocTypeGeneral)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if doc != nil {
		t.Errorf("expected nil GeneratedDoc on error, got %+v", doc)
	}

	// NotifyError must have been called.
	found := false
	for _, n := range pane.notifications {
		if strings.Contains(n, wantErr.Error()) {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected notification containing %q; got: %v", wantErr.Error(), pane.notifications)
	}
}

// ── GenerateMultiple tests ────────────────────────────────────────────────────

// mockSelectiveClient fails on the Nth call to Generate (1-indexed).
// All other calls succeed with the provided chunks.
type mockSelectiveClient struct {
	chunks   []string
	failOnNth int
	callCount int
}

func (m *mockSelectiveClient) Generate(prompt string, model string, context []int) (<-chan string, error) {
	m.callCount++
	if m.callCount == m.failOnNth {
		return nil, fmt.Errorf("selective failure on call %d", m.callCount)
	}
	ch := make(chan string, len(m.chunks))
	for _, c := range m.chunks {
		ch <- c
	}
	close(ch)
	return ch, nil
}

func (m *mockSelectiveClient) IsAvailable() (bool, error) { return true, nil }
func (m *mockSelectiveClient) ListModels() ([]string, error) { return nil, nil }

// TestGenerateMultiple_MixedSuccessFailure verifies that when one doc type
// fails and another succeeds, GenerateMultiple returns only the successful doc
// and calls NotifyError for the failed type.
// Requirements: 1.2, 4.2, 4.3
func TestGenerateMultiple_MixedSuccessFailure(t *testing.T) {
	// Fail on the first call (DocTypeGeneral), succeed on the second (DocTypeAPI).
	client := &mockSelectiveClient{
		chunks:    []string{"api content"},
		failOnNth: 1,
	}
	feedback, pane := newTestFeedback()
	g := NewAIGenerator(client, "test-model", feedback)

	docTypes := []DocumentationType{DocTypeGeneral, DocTypeAPI}
	docs, err := g.GenerateMultiple(&AnalysisResult{}, docTypes)

	// GenerateMultiple itself never returns a non-nil error.
	if err != nil {
		t.Fatalf("unexpected error from GenerateMultiple: %v", err)
	}

	// Only the successful doc (DocTypeAPI) should be returned.
	if len(docs) != 1 {
		t.Fatalf("expected 1 successful doc, got %d", len(docs))
	}
	if docs[0].Type != DocTypeAPI {
		t.Errorf("expected successful doc type DocTypeAPI, got %v", docs[0].Type)
	}
	if docs[0].Content != "api content" {
		t.Errorf("expected content %q, got %q", "api content", docs[0].Content)
	}

	// NotifyError must have been called for the failed type.
	if len(pane.notifications) == 0 {
		t.Fatal("expected NotifyError to be called for the failed doc type, but no notifications recorded")
	}
	foundError := false
	for _, n := range pane.notifications {
		if strings.Contains(n, "selective failure") {
			foundError = true
			break
		}
	}
	if !foundError {
		t.Errorf("expected a notification containing the failure message; got: %v", pane.notifications)
	}
}
