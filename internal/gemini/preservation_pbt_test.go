package gemini

// Preservation Property Tests - Gemini Streaming Text and Error Behavior Unchanged
//
// **Validates: Requirements 3.3, 3.8, 3.9**
//
// These tests verify that the Gemini streaming response behavior is preserved.
// They validate that candidate text is delivered through the channel and
// errors are properly handled.
//
// Property 2: Preservation - For all valid Gemini responses with candidate content,
// the response channel receives the candidate text.
// For error scenarios, errors are returned or delivered through the channel.

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// writeSSEEvent writes a single SSE data event for PBT tests.
func writeSSEEvent(w http.ResponseWriter, data interface{}) {
	jsonBytes, _ := json.Marshal(data)
	fmt.Fprintf(w, "data: %s\n\n", jsonBytes)
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

// TestPreservation_GeminiCandidateTextDelivery verifies that for all valid Gemini
// responses with candidate content, the response channel receives the candidate text.
//
// **Validates: Requirements 3.3, 3.8**
func TestPreservation_GeminiCandidateTextDelivery(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate non-empty candidate text (ASCII to avoid JSON encoding issues)
		candidateText := rapid.StringMatching(`[a-zA-Z0-9 .,!?]{1,200}`).Draw(t, "candidateText")

		// Create mock Gemini SSE streaming server
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/event-stream")
			writeSSEEvent(w, map[string]interface{}{
				"candidates": []map[string]interface{}{
					{
						"content": map[string]interface{}{
							"parts": []map[string]interface{}{
								{"text": candidateText},
							},
						},
					},
				},
			})
		}))
		defer mockServer.Close()

		client := &GeminiClient{
			apiKey:     "test-api-key",
			baseURL:    mockServer.URL,
			httpClient: http.DefaultClient,
		}

		responseChan, err := client.Generate("test prompt", "gemini-2.0-flash-exp", nil, nil)
		if err != nil {
			t.Fatalf("Generate returned error: %v", err)
		}

		// Collect all received text
		var received []string
		for chunk := range responseChan {
			received = append(received, chunk)
		}

		// Verify: candidate text received
		if len(received) != 1 {
			t.Fatalf("expected 1 text chunk, got %d: %v", len(received), received)
		}

		if received[0] != candidateText {
			t.Fatalf("expected candidate text %q, got %q", candidateText, received[0])
		}

		// Verify: channel is closed
		_, ok := <-responseChan
		if ok {
			t.Fatal("expected channel to be closed after delivering candidate text")
		}
	})
}

// TestPreservation_GeminiMultipleStreamChunks verifies that when a Gemini streaming
// response has multiple chunks, all text parts are delivered through the channel.
//
// **Validates: Requirements 3.3**
func TestPreservation_GeminiMultipleStreamChunks(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate 2-5 text chunks
		numChunks := rapid.IntRange(2, 5).Draw(t, "numChunks")
		texts := make([]string, numChunks)
		for i := 0; i < numChunks; i++ {
			texts[i] = rapid.StringMatching(`[a-zA-Z0-9 ]{1,50}`).Draw(t, fmt.Sprintf("text_%d", i))
		}

		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/event-stream")
			for _, text := range texts {
				writeSSEEvent(w, map[string]interface{}{
					"candidates": []map[string]interface{}{
						{
							"content": map[string]interface{}{
								"parts": []map[string]interface{}{
									{"text": text},
								},
							},
						},
					},
				})
			}
		}))
		defer mockServer.Close()

		client := &GeminiClient{
			apiKey:     "test-api-key",
			baseURL:    mockServer.URL,
			httpClient: http.DefaultClient,
		}

		responseChan, err := client.Generate("test prompt", "gemini-2.0-flash-exp", nil, nil)
		if err != nil {
			t.Fatalf("Generate returned error: %v", err)
		}

		var received []string
		for chunk := range responseChan {
			received = append(received, chunk)
		}

		// All chunks should be delivered
		combined := strings.Join(received, "")
		expected := strings.Join(texts, "")
		if combined != expected {
			t.Fatalf("expected combined text %q, got %q", expected, combined)
		}
	})
}

// TestPreservation_GeminiErrorHandling verifies that API errors are properly
// returned or delivered through the channel.
//
// **Validates: Requirements 3.9**
func TestPreservation_GeminiErrorHandling(t *testing.T) {
	t.Run("api_error_in_stream", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			errorMsg := rapid.StringMatching(`[a-zA-Z ]{5,30}`).Draw(t, "errorMsg")
			errorCode := rapid.IntRange(400, 599).Draw(t, "errorCode")

			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/event-stream")
				writeSSEEvent(w, map[string]interface{}{
					"error": map[string]interface{}{
						"code":    errorCode,
						"message": errorMsg,
						"status":  "ERROR",
					},
				})
			}))
			defer mockServer.Close()

			client := &GeminiClient{
				apiKey:     "test-api-key",
				baseURL:    mockServer.URL,
				httpClient: http.DefaultClient,
			}

			responseChan, err := client.Generate("test prompt", "gemini-2.0-flash-exp", nil, nil)
			if err != nil {
				// Error returned directly from Generate is acceptable
				return
			}

			// Error delivered through channel is also acceptable
			var received string
			for chunk := range responseChan {
				received += chunk
			}

			if !containsSubstring(received, errorMsg) {
				t.Fatalf("expected error message containing %q in channel, got: %q", errorMsg, received)
			}
		})
	})

	t.Run("http_error_status", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			statusCode := rapid.SampledFrom([]int{400, 401, 403, 404, 500, 502, 503}).Draw(t, "statusCode")

			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(statusCode)
				w.Write([]byte("error response body"))
			}))
			defer mockServer.Close()

			client := &GeminiClient{
				apiKey:     "test-api-key",
				baseURL:    mockServer.URL,
				httpClient: http.DefaultClient,
			}

			_, err := client.Generate("test prompt", "gemini-2.0-flash-exp", nil, nil)

			// Verify: HTTP error returned as error from Generate
			if err == nil {
				t.Fatalf("expected error from Generate for HTTP status %d, got nil", statusCode)
			}
		})
	})
}

// containsSubstring checks if s contains substr (case-sensitive)
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && searchSubstring(s, substr)
}

func searchSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
