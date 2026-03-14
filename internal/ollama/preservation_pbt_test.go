package ollama

// Preservation Property Tests - Streaming Text and Error Behavior Unchanged
//
// **Validates: Requirements 3.4, 3.5, 3.8, 3.9**
//
// These tests verify that the CURRENT (unfixed) streaming behavior is preserved.
// They must PASS on unfixed code to establish the baseline.
//
// Property 2: Preservation - For all valid Ollama streams with response chunks,
// the response channel receives text in order and closes on done: true.
// For error scenarios, error messages are delivered through the channel.

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// TestPreservation_OllamaTextStreamingOrder verifies that for all valid Ollama
// streams with response chunks, the response channel receives text in order
// and closes on done: true.
//
// **Validates: Requirements 3.4, 3.5, 3.8**
func TestPreservation_OllamaTextStreamingOrder(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate 1-10 non-empty text chunks
		numChunks := rapid.IntRange(1, 10).Draw(t, "numChunks")
		chunks := make([]string, numChunks)
		for i := 0; i < numChunks; i++ {
			// Generate non-empty ASCII text (avoid control chars that break JSON)
			chunks[i] = rapid.StringMatching(`[a-zA-Z0-9 ]{1,50}`).Draw(t, fmt.Sprintf("chunk_%d", i))
		}

		// Create mock Ollama server that streams these chunks
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/api/tags" {
				json.NewEncoder(w).Encode(map[string]interface{}{
					"models": []interface{}{},
				})
				return
			}

			flusher, ok := w.(http.Flusher)
			if !ok {
				t.Fatal("expected http.Flusher")
				return
			}

			w.Header().Set("Content-Type", "application/x-ndjson")

			// Stream text chunks
			for _, chunk := range chunks {
				data, _ := json.Marshal(map[string]interface{}{
					"model":    "testmodel",
					"response": chunk,
					"done":     false,
				})
				fmt.Fprintf(w, "%s\n", data)
				flusher.Flush()
			}

			// Send done chunk
			data, _ := json.Marshal(map[string]interface{}{
				"model":    "testmodel",
				"response": "",
				"done":     true,
				"context":  []int{1, 2, 3},
			})
			fmt.Fprintf(w, "%s\n", data)
			flusher.Flush()
		}))
		defer mockServer.Close()

		client := NewOllamaClient(mockServer.URL)

		// Use CURRENT (unfixed) Generate signature: no callback parameter
		responseChan, err := client.Generate("test prompt", "testmodel", nil, nil)
		if err != nil {
			t.Fatalf("Generate returned error: %v", err)
		}

		// Collect all received chunks
		var received []string
		for chunk := range responseChan {
			received = append(received, chunk)
		}

		// Verify: all text chunks received in order
		if len(received) != len(chunks) {
			t.Fatalf("expected %d chunks, got %d", len(chunks), len(received))
		}

		for i, expected := range chunks {
			if received[i] != expected {
				t.Fatalf("chunk %d: expected %q, got %q", i, expected, received[i])
			}
		}

		// Verify: channel is closed (range loop exited)
		_, ok := <-responseChan
		if ok {
			t.Fatal("expected channel to be closed after done:true")
		}
	})
}

// TestPreservation_OllamaErrorHandling verifies that error messages (malformed JSON,
// API errors) are delivered through the response channel.
//
// **Validates: Requirements 3.9**
func TestPreservation_OllamaErrorHandling(t *testing.T) {
	t.Run("malformed_json", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			// Generate some garbage that isn't valid JSON
			garbage := rapid.StringMatching(`[a-zA-Z]{5,20}`).Draw(t, "garbage")

			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/api/tags" {
					json.NewEncoder(w).Encode(map[string]interface{}{
						"models": []interface{}{},
					})
					return
				}

				flusher, ok := w.(http.Flusher)
				if !ok {
					return
				}
				w.Header().Set("Content-Type", "application/x-ndjson")
				// Send malformed JSON
				fmt.Fprintf(w, "%s\n", garbage)
				flusher.Flush()
			}))
			defer mockServer.Close()

			client := NewOllamaClient(mockServer.URL)
			responseChan, err := client.Generate("test", "testmodel", nil, nil)
			if err != nil {
				t.Fatalf("Generate returned error: %v", err)
			}

			// Collect all messages from channel
			var messages []string
			for msg := range responseChan {
				messages = append(messages, msg)
			}

			// Verify: error message delivered through channel
			if len(messages) == 0 {
				t.Fatal("expected error message in channel, got nothing")
			}

			// The error message should mention parsing
			found := false
			for _, msg := range messages {
				if strings.Contains(msg, "Error parsing response") {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("expected 'Error parsing response' message, got: %v", messages)
			}
		})
	})

	t.Run("api_error", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			errorMsg := rapid.StringMatching(`[a-zA-Z ]{5,30}`).Draw(t, "errorMsg")

			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/api/tags" {
					json.NewEncoder(w).Encode(map[string]interface{}{
						"models": []interface{}{},
					})
					return
				}

				flusher, ok := w.(http.Flusher)
				if !ok {
					return
				}
				w.Header().Set("Content-Type", "application/x-ndjson")
				data, _ := json.Marshal(map[string]interface{}{
					"model":    "testmodel",
					"response": "",
					"done":     false,
					"error":    errorMsg,
				})
				fmt.Fprintf(w, "%s\n", data)
				flusher.Flush()
			}))
			defer mockServer.Close()

			client := NewOllamaClient(mockServer.URL)
			responseChan, err := client.Generate("test", "testmodel", nil, nil)
			if err != nil {
				t.Fatalf("Generate returned error: %v", err)
			}

			var messages []string
			for msg := range responseChan {
				messages = append(messages, msg)
			}

			// Verify: API error delivered through channel
			if len(messages) == 0 {
				t.Fatal("expected API error message in channel, got nothing")
			}

			found := false
			for _, msg := range messages {
				if strings.Contains(msg, "API Error:") && strings.Contains(msg, errorMsg) {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("expected 'API Error: %s' message, got: %v", errorMsg, messages)
			}
		})
	})
}
