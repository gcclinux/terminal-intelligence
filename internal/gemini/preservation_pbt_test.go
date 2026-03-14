package gemini

// Preservation Property Tests - Gemini Streaming Text and Error Behavior Unchanged
//
// **Validates: Requirements 3.3, 3.8, 3.9**
//
// These tests verify that the CURRENT (unfixed) Gemini response behavior is preserved.
// They must PASS on unfixed code to establish the baseline.
//
// Property 2: Preservation - For all valid Gemini responses with candidate content,
// the response channel receives the candidate text.
// For error scenarios, errors are returned or delivered through the channel.

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"pgregory.net/rapid"
)

// TestPreservation_GeminiCandidateTextDelivery verifies that for all valid Gemini
// responses with candidate content, the response channel receives the candidate text.
//
// **Validates: Requirements 3.3, 3.8**
func TestPreservation_GeminiCandidateTextDelivery(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate non-empty candidate text (ASCII to avoid JSON encoding issues)
		candidateText := rapid.StringMatching(`[a-zA-Z0-9 .,!?]{1,200}`).Draw(t, "candidateText")

		// Create mock Gemini API server
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			response := map[string]interface{}{
				"candidates": []map[string]interface{}{
					{
						"content": map[string]interface{}{
							"parts": []map[string]interface{}{
								{"text": candidateText},
							},
						},
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}))
		defer mockServer.Close()

		// Create client with mock server URL (using unexported baseURL field)
		client := &GeminiClient{
			apiKey:     "test-api-key",
			baseURL:    mockServer.URL,
			httpClient: http.DefaultClient,
		}

		// Use CURRENT Generate signature with nil callback
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

// TestPreservation_GeminiMultipleCandidateParts verifies that when a Gemini response
// has multiple parts, only the first part's text is delivered (current behavior).
//
// **Validates: Requirements 3.3**
func TestPreservation_GeminiMultipleCandidateParts(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate 2-5 text parts
		numParts := rapid.IntRange(2, 5).Draw(t, "numParts")
		parts := make([]map[string]interface{}, numParts)
		firstText := rapid.StringMatching(`[a-zA-Z0-9 ]{1,50}`).Draw(t, "firstText")
		parts[0] = map[string]interface{}{"text": firstText}
		for i := 1; i < numParts; i++ {
			parts[i] = map[string]interface{}{
				"text": rapid.StringMatching(`[a-zA-Z0-9 ]{1,50}`).Draw(t, fmt.Sprintf("part_%d", i)),
			}
		}

		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			response := map[string]interface{}{
				"candidates": []map[string]interface{}{
					{
						"content": map[string]interface{}{
							"parts": parts,
						},
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
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

		// Current behavior: only first part's text is delivered
		if len(received) != 1 {
			t.Fatalf("expected 1 text chunk (first part only), got %d: %v", len(received), received)
		}

		if received[0] != firstText {
			t.Fatalf("expected first part text %q, got %q", firstText, received[0])
		}
	})
}

// TestPreservation_GeminiErrorHandling verifies that API errors are properly
// returned or delivered through the channel.
//
// **Validates: Requirements 3.9**
func TestPreservation_GeminiErrorHandling(t *testing.T) {
	t.Run("api_error_in_response", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			errorMsg := rapid.StringMatching(`[a-zA-Z ]{5,30}`).Draw(t, "errorMsg")
			errorCode := rapid.IntRange(400, 599).Draw(t, "errorCode")

			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				response := map[string]interface{}{
					"error": map[string]interface{}{
						"code":    errorCode,
						"message": errorMsg,
						"status":  "ERROR",
					},
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(response)
			}))
			defer mockServer.Close()

			client := &GeminiClient{
				apiKey:     "test-api-key",
				baseURL:    mockServer.URL,
				httpClient: http.DefaultClient,
			}

			_, err := client.Generate("test prompt", "gemini-2.0-flash-exp", nil, nil)

			// Verify: API error returned as error from Generate
			if err == nil {
				t.Fatal("expected error from Generate for API error response, got nil")
			}

			// Error message should contain the API error message
			if !containsSubstring(err.Error(), errorMsg) {
				t.Fatalf("expected error to contain %q, got: %v", errorMsg, err)
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
