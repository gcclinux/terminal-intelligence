package gemini

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/user/terminal-intelligence/internal/types"
)

// writeSSEChunk writes a single SSE data event to the response writer.
func writeSSEChunk(w http.ResponseWriter, data interface{}) {
	jsonBytes, _ := json.Marshal(data)
	fmt.Fprintf(w, "data: %s\n\n", jsonBytes)
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

func TestGenerate_StreamingTokenUsage(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")

		// First chunk: text content
		writeSSEChunk(w, map[string]interface{}{
			"candidates": []map[string]interface{}{
				{
					"content": map[string]interface{}{
						"parts": []map[string]interface{}{
							{"text": "Hello "},
						},
					},
				},
			},
		})

		// Second chunk: more text
		writeSSEChunk(w, map[string]interface{}{
			"candidates": []map[string]interface{}{
				{
					"content": map[string]interface{}{
						"parts": []map[string]interface{}{
							{"text": "world"},
						},
					},
				},
			},
		})

		// Final chunk: usageMetadata with token counts
		writeSSEChunk(w, map[string]interface{}{
			"candidates": []map[string]interface{}{
				{
					"content": map[string]interface{}{
						"parts": []map[string]interface{}{
							{"text": ""},
						},
					},
				},
			},
			"usageMetadata": map[string]interface{}{
				"promptTokenCount":     42,
				"candidatesTokenCount": 10,
				"totalTokenCount":      52,
			},
		})
	}))
	defer mockServer.Close()

	client := NewGeminiClientWithURL("test-key", mockServer.URL)

	var capturedUsage types.TokenUsage
	onTokenUsage := func(usage types.TokenUsage) {
		capturedUsage = usage
	}

	responseChan, err := client.Generate("test prompt", "gemini-2.0-flash-exp", nil, onTokenUsage)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	var fullResponse string
	for chunk := range responseChan {
		fullResponse += chunk
	}

	if fullResponse != "Hello world" {
		t.Errorf("expected response 'Hello world', got %q", fullResponse)
	}

	if capturedUsage.InputTokens != 42 {
		t.Errorf("expected InputTokens=42, got %d", capturedUsage.InputTokens)
	}
	if capturedUsage.OutputTokens != 10 {
		t.Errorf("expected OutputTokens=10, got %d", capturedUsage.OutputTokens)
	}
	if capturedUsage.TotalTokens != 52 {
		t.Errorf("expected TotalTokens=52, got %d", capturedUsage.TotalTokens)
	}
}

func TestGenerate_StreamingWithThinkingTokens(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")

		writeSSEChunk(w, map[string]interface{}{
			"candidates": []map[string]interface{}{
				{
					"content": map[string]interface{}{
						"parts": []map[string]interface{}{
							{"text": "Thinking model response"},
						},
					},
				},
			},
			"usageMetadata": map[string]interface{}{
				"promptTokenCount":     100,
				"candidatesTokenCount": 50,
				"totalTokenCount":      250,
				"thoughtsTokenCount":   100,
			},
		})
	}))
	defer mockServer.Close()

	client := NewGeminiClientWithURL("test-key", mockServer.URL)

	var capturedUsage types.TokenUsage
	onTokenUsage := func(usage types.TokenUsage) {
		capturedUsage = usage
	}

	responseChan, err := client.Generate("test", "gemini-3-flash-preview", nil, onTokenUsage)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	for range responseChan {
	}

	if capturedUsage.InputTokens != 100 {
		t.Errorf("expected InputTokens=100, got %d", capturedUsage.InputTokens)
	}
	if capturedUsage.OutputTokens != 50 {
		t.Errorf("expected OutputTokens=50, got %d", capturedUsage.OutputTokens)
	}
	if capturedUsage.TotalTokens != 250 {
		t.Errorf("expected TotalTokens=250, got %d", capturedUsage.TotalTokens)
	}
}

func TestGenerate_StreamingNoUsageMetadata(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")

		writeSSEChunk(w, map[string]interface{}{
			"candidates": []map[string]interface{}{
				{
					"content": map[string]interface{}{
						"parts": []map[string]interface{}{
							{"text": "response without metadata"},
						},
					},
				},
			},
		})
	}))
	defer mockServer.Close()

	client := NewGeminiClientWithURL("test-key", mockServer.URL)

	callbackCalled := false
	onTokenUsage := func(usage types.TokenUsage) {
		callbackCalled = true
	}

	responseChan, err := client.Generate("test", "model", nil, onTokenUsage)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	for range responseChan {
	}

	if !callbackCalled {
		t.Fatal("onTokenUsage callback was not called")
	}
}

func TestGenerate_StreamingErrorInResponse(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")

		writeSSEChunk(w, map[string]interface{}{
			"error": map[string]interface{}{
				"code":    400,
				"message": "Invalid request",
				"status":  "INVALID_ARGUMENT",
			},
		})
	}))
	defer mockServer.Close()

	client := NewGeminiClientWithURL("test-key", mockServer.URL)

	responseChan, err := client.Generate("test", "model", nil, nil)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	var received string
	for chunk := range responseChan {
		received += chunk
	}

	if received == "" {
		t.Fatal("expected error message in channel")
	}
}

func TestGenerate_HTTPError(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("forbidden"))
	}))
	defer mockServer.Close()

	client := NewGeminiClientWithURL("test-key", mockServer.URL)

	_, err := client.Generate("test", "model", nil, nil)
	if err == nil {
		t.Fatal("expected error for HTTP 403")
	}
}

func TestGenerate_MultipleChunksTokenUsage(t *testing.T) {
	// Verify token usage is captured from the LAST chunk with usageMetadata
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")

		// Chunk with partial usage
		writeSSEChunk(w, map[string]interface{}{
			"candidates": []map[string]interface{}{
				{"content": map[string]interface{}{
					"parts": []map[string]interface{}{{"text": "chunk1 "}},
				}},
			},
			"usageMetadata": map[string]interface{}{
				"promptTokenCount":     10,
				"candidatesTokenCount": 5,
				"totalTokenCount":      15,
			},
		})

		// Chunk with updated usage (final)
		writeSSEChunk(w, map[string]interface{}{
			"candidates": []map[string]interface{}{
				{"content": map[string]interface{}{
					"parts": []map[string]interface{}{{"text": "chunk2"}},
				}},
			},
			"usageMetadata": map[string]interface{}{
				"promptTokenCount":     10,
				"candidatesTokenCount": 20,
				"totalTokenCount":      30,
			},
		})
	}))
	defer mockServer.Close()

	client := NewGeminiClientWithURL("test-key", mockServer.URL)

	var capturedUsage types.TokenUsage
	onTokenUsage := func(usage types.TokenUsage) {
		capturedUsage = usage
	}

	responseChan, err := client.Generate("test", "model", nil, onTokenUsage)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	var fullResponse string
	for chunk := range responseChan {
		fullResponse += chunk
	}

	if fullResponse != "chunk1 chunk2" {
		t.Errorf("expected 'chunk1 chunk2', got %q", fullResponse)
	}

	// Should have the LAST usage metadata values
	if capturedUsage.InputTokens != 10 {
		t.Errorf("expected InputTokens=10, got %d", capturedUsage.InputTokens)
	}
	if capturedUsage.OutputTokens != 20 {
		t.Errorf("expected OutputTokens=20, got %d", capturedUsage.OutputTokens)
	}
	if capturedUsage.TotalTokens != 30 {
		t.Errorf("expected TotalTokens=30, got %d", capturedUsage.TotalTokens)
	}
}

func TestGenerate_StreamingUsesCorrectEndpoint(t *testing.T) {
	var capturedPath string
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		w.Header().Set("Content-Type", "text/event-stream")
		writeSSEChunk(w, map[string]interface{}{
			"candidates": []map[string]interface{}{
				{"content": map[string]interface{}{
					"parts": []map[string]interface{}{{"text": "ok"}},
				}},
			},
		})
	}))
	defer mockServer.Close()

	client := NewGeminiClientWithURL("test-key", mockServer.URL)

	responseChan, err := client.Generate("test", "gemini-3-flash-preview", nil, nil)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	for range responseChan {
	}

	expected := "/models/gemini-3-flash-preview:streamGenerateContent"
	if capturedPath != expected {
		t.Errorf("expected path %q, got %q", expected, capturedPath)
	}
}
