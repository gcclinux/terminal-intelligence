package bedrock

// Preservation Property Tests - Bedrock Streaming Text and Error Behavior Unchanged
//
// **Validates: Requirements 3.1, 3.2, 3.8, 3.9**
//
// These tests verify that the CURRENT (unfixed) Bedrock event parsing behavior
// is preserved. They must PASS on unfixed code to establish the baseline.
//
// Property 2: Preservation - For all valid Bedrock streams with content_block_delta
// text events, the response channel receives the same text chunks in order.
// message_stop closes the channel. Error messages are delivered through the channel.
//
// Since Bedrock uses the AWS SDK for streaming (which is difficult to mock at the
// HTTP level), these tests verify the event parsing logic that runs inside the
// Generate goroutine by simulating the JSON parsing behavior directly.

import (
	"encoding/json"
	"fmt"
	"testing"

	"pgregory.net/rapid"
)

// simulateBedrockEventProcessing replicates the event parsing logic from
// Generate's goroutine. This is the CURRENT (unfixed) logic that processes
// ResponseStreamMemberChunk events.
//
// Returns: extracted text chunks, whether message_stop was found, any error message
func simulateBedrockEventProcessing(events [][]byte) (texts []string, stopped bool, errMsg string) {
	for _, eventBytes := range events {
		var response map[string]any
		if err := json.Unmarshal(eventBytes, &response); err != nil {
			errMsg = fmt.Sprintf("Error parsing response: %v", err)
			return texts, false, errMsg
		}

		// Check for content_block_delta event type
		if eventType, ok := response["type"].(string); ok && eventType == "content_block_delta" {
			if delta, ok := response["delta"].(map[string]any); ok {
				if text, ok := delta["text"].(string); ok && text != "" {
					texts = append(texts, text)
				}
			}
		}

		// Check for message_stop event to close channel
		if eventType, ok := response["type"].(string); ok && eventType == "message_stop" {
			return texts, true, ""
		}
	}
	return texts, false, ""
}

// TestPreservation_BedrockTextChunksInOrder verifies that for all valid Bedrock
// streams with content_block_delta text events, the extracted text chunks match
// the input in order.
//
// **Validates: Requirements 3.1, 3.8**
func TestPreservation_BedrockTextChunksInOrder(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate 1-10 non-empty text chunks
		numChunks := rapid.IntRange(1, 10).Draw(t, "numChunks")
		textChunks := make([]string, numChunks)
		for i := 0; i < numChunks; i++ {
			textChunks[i] = rapid.StringMatching(`[a-zA-Z0-9 ]{1,50}`).Draw(t, fmt.Sprintf("chunk_%d", i))
		}

		// Build event stream: message_start + content_block_delta events + message_stop
		var events [][]byte

		// message_start event (should be ignored by current logic)
		startEvent, _ := json.Marshal(map[string]interface{}{
			"type": "message_start",
		})
		events = append(events, startEvent)

		// content_block_delta events with text
		for _, text := range textChunks {
			event, _ := json.Marshal(map[string]interface{}{
				"type": "content_block_delta",
				"delta": map[string]interface{}{
					"type": "text_delta",
					"text": text,
				},
			})
			events = append(events, event)
		}

		// message_stop event
		stopEvent, _ := json.Marshal(map[string]interface{}{
			"type": "message_stop",
		})
		events = append(events, stopEvent)

		// Process events using the same logic as Generate's goroutine
		extracted, stopped, errMsg := simulateBedrockEventProcessing(events)

		// Verify: no error
		if errMsg != "" {
			t.Fatalf("unexpected error: %s", errMsg)
		}

		// Verify: message_stop was processed
		if !stopped {
			t.Fatal("expected message_stop to be processed")
		}

		// Verify: all text chunks extracted in order
		if len(extracted) != len(textChunks) {
			t.Fatalf("expected %d text chunks, got %d", len(textChunks), len(extracted))
		}

		for i, expected := range textChunks {
			if extracted[i] != expected {
				t.Fatalf("chunk %d: expected %q, got %q", i, expected, extracted[i])
			}
		}
	})
}

// TestPreservation_BedrockMessageStopClosesStream verifies that message_stop
// event causes processing to stop, even if there are more events after it.
//
// **Validates: Requirements 3.2**
func TestPreservation_BedrockMessageStopClosesStream(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate text before message_stop
		beforeText := rapid.StringMatching(`[a-zA-Z0-9]{1,20}`).Draw(t, "beforeText")
		// Generate text after message_stop (should be ignored)
		afterText := rapid.StringMatching(`[a-zA-Z0-9]{1,20}`).Draw(t, "afterText")

		var events [][]byte

		// content_block_delta before stop
		event1, _ := json.Marshal(map[string]interface{}{
			"type": "content_block_delta",
			"delta": map[string]interface{}{
				"type": "text_delta",
				"text": beforeText,
			},
		})
		events = append(events, event1)

		// message_stop
		stopEvent, _ := json.Marshal(map[string]interface{}{
			"type": "message_stop",
		})
		events = append(events, stopEvent)

		// content_block_delta after stop (should never be processed)
		event2, _ := json.Marshal(map[string]interface{}{
			"type": "content_block_delta",
			"delta": map[string]interface{}{
				"type": "text_delta",
				"text": afterText,
			},
		})
		events = append(events, event2)

		extracted, stopped, errMsg := simulateBedrockEventProcessing(events)

		if errMsg != "" {
			t.Fatalf("unexpected error: %s", errMsg)
		}

		if !stopped {
			t.Fatal("expected message_stop to be processed")
		}

		// Only the text before message_stop should be extracted
		if len(extracted) != 1 {
			t.Fatalf("expected 1 text chunk (before stop), got %d: %v", len(extracted), extracted)
		}

		if extracted[0] != beforeText {
			t.Fatalf("expected %q, got %q", beforeText, extracted[0])
		}
	})
}

// TestPreservation_BedrockEmptyTextChunksFiltered verifies that empty text
// in content_block_delta events is filtered out (not sent to channel).
//
// **Validates: Requirements 3.1**
func TestPreservation_BedrockEmptyTextChunksFiltered(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a mix of empty and non-empty text chunks
		numChunks := rapid.IntRange(2, 8).Draw(t, "numChunks")
		var allTexts []string
		var expectedNonEmpty []string

		for i := 0; i < numChunks; i++ {
			isEmpty := rapid.Bool().Draw(t, fmt.Sprintf("isEmpty_%d", i))
			if isEmpty {
				allTexts = append(allTexts, "")
			} else {
				text := rapid.StringMatching(`[a-zA-Z0-9]{1,20}`).Draw(t, fmt.Sprintf("text_%d", i))
				allTexts = append(allTexts, text)
				expectedNonEmpty = append(expectedNonEmpty, text)
			}
		}

		var events [][]byte
		for _, text := range allTexts {
			event, _ := json.Marshal(map[string]interface{}{
				"type": "content_block_delta",
				"delta": map[string]interface{}{
					"type": "text_delta",
					"text": text,
				},
			})
			events = append(events, event)
		}

		stopEvent, _ := json.Marshal(map[string]interface{}{
			"type": "message_stop",
		})
		events = append(events, stopEvent)

		extracted, _, errMsg := simulateBedrockEventProcessing(events)

		if errMsg != "" {
			t.Fatalf("unexpected error: %s", errMsg)
		}

		// Verify: only non-empty texts extracted
		if len(extracted) != len(expectedNonEmpty) {
			t.Fatalf("expected %d non-empty chunks, got %d", len(expectedNonEmpty), len(extracted))
		}

		for i, expected := range expectedNonEmpty {
			if extracted[i] != expected {
				t.Fatalf("non-empty chunk %d: expected %q, got %q", i, expected, extracted[i])
			}
		}
	})
}

// TestPreservation_BedrockMalformedEventError verifies that malformed JSON events
// produce an error message (delivered through the channel in the real implementation).
//
// **Validates: Requirements 3.9**
func TestPreservation_BedrockMalformedEventError(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate some valid events followed by malformed JSON
		validText := rapid.StringMatching(`[a-zA-Z0-9]{1,20}`).Draw(t, "validText")
		garbage := rapid.StringMatching(`[a-zA-Z]{5,20}`).Draw(t, "garbage")

		var events [][]byte

		// Valid content_block_delta
		validEvent, _ := json.Marshal(map[string]interface{}{
			"type": "content_block_delta",
			"delta": map[string]interface{}{
				"type": "text_delta",
				"text": validText,
			},
		})
		events = append(events, validEvent)

		// Malformed JSON
		events = append(events, []byte(garbage))

		extracted, stopped, errMsg := simulateBedrockEventProcessing(events)

		// Verify: valid text before error was extracted
		if len(extracted) != 1 || extracted[0] != validText {
			t.Fatalf("expected valid text %q before error, got: %v", validText, extracted)
		}

		// Verify: processing stopped (not via message_stop)
		if stopped {
			t.Fatal("expected processing to stop due to error, not message_stop")
		}

		// Verify: error message produced
		if errMsg == "" {
			t.Fatal("expected error message for malformed JSON")
		}

		if !searchStr(errMsg, "Error parsing response") {
			t.Fatalf("expected 'Error parsing response' in error, got: %s", errMsg)
		}
	})
}

// searchStr checks if s contains substr
func searchStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
