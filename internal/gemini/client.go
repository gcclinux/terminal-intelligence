package gemini

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/user/terminal-intelligence/internal/types"
)

// GeminiClient handles communication with Google's Gemini API
type GeminiClient struct {
	apiKey     string
	httpClient *http.Client
	baseURL    string
}

// NewGeminiClient creates a new Gemini client
func NewGeminiClient(apiKey string) *GeminiClient {
	return &GeminiClient{
		apiKey:  apiKey,
		baseURL: "https://generativelanguage.googleapis.com/v1beta",
		httpClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}
}

// NewGeminiClientWithURL creates a new Gemini client with a custom base URL (for testing)
func NewGeminiClientWithURL(apiKey string, baseURL string) *GeminiClient {
	return &GeminiClient{
		apiKey:  apiKey,
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}
}

// geminiRequest represents the request body for Gemini API
type geminiRequest struct {
	Contents []geminiContent `json:"contents"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

// geminiResponse represents a single response chunk from Gemini API
// Used for both streaming (streamGenerateContent) and non-streaming responses.
type geminiResponse struct {
	Candidates    []geminiCandidate    `json:"candidates"`
	Error         *geminiError         `json:"error,omitempty"`
	UsageMetadata *geminiUsageMetadata `json:"usageMetadata,omitempty"`
}

// geminiUsageMetadata represents token usage data from the Gemini API response
type geminiUsageMetadata struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
	TotalTokenCount      int `json:"totalTokenCount"`
	ThoughtsTokenCount   int `json:"thoughtsTokenCount"`
}

type geminiCandidate struct {
	Content geminiContent `json:"content"`
}

type geminiError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Status  string `json:"status"`
}

// IsAvailable checks if Gemini API is available
func (gc *GeminiClient) IsAvailable() (bool, error) {
	if gc.apiKey == "" {
		return false, fmt.Errorf("Gemini API key not provided")
	}
	return true, nil
}

// Generate generates AI response from Gemini using the streaming API.
// This uses streamGenerateContent to receive chunks as they are generated,
// providing real-time streaming and reliable token usage reporting.
func (gc *GeminiClient) Generate(prompt string, model string, context []int, onTokenUsage func(types.TokenUsage)) (<-chan string, error) {
	if model == "" {
		model = "gemini-2.0-flash-exp"
	}

	reqBody := geminiRequest{
		Contents: []geminiContent{
			{
				Parts: []geminiPart{
					{Text: prompt},
				},
			},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/models/%s:streamGenerateContent?alt=sse&key=%s", gc.baseURL, model, gc.apiKey)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := gc.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Gemini API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Create buffered channel for streaming response chunks
	responseChan := make(chan string, 10)

	// Launch goroutine to process SSE stream
	go func() {
		defer close(responseChan)
		defer resp.Body.Close()

		var inputTokens, outputTokens, totalTokens int

		scanner := bufio.NewScanner(resp.Body)
		// Increase scanner buffer for large responses
		scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

		for scanner.Scan() {
			line := scanner.Text()

			// SSE format: lines starting with "data: " contain JSON
			if len(line) < 6 || line[:6] != "data: " {
				continue
			}
			data := line[6:]

			var chunk geminiResponse
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				continue
			}

			// Check for API error in stream
			if chunk.Error != nil {
				responseChan <- fmt.Sprintf("Error: %s", chunk.Error.Message)
				return
			}

			// Extract text from candidates
			if len(chunk.Candidates) > 0 {
				candidate := chunk.Candidates[0]
				if len(candidate.Content.Parts) > 0 {
					text := candidate.Content.Parts[0].Text
					if text != "" {
						responseChan <- text
					}
				}
			}

			// Capture token usage from usageMetadata (present in the final chunk)
			if chunk.UsageMetadata != nil {
				inputTokens = chunk.UsageMetadata.PromptTokenCount
				outputTokens = chunk.UsageMetadata.CandidatesTokenCount
				totalTokens = chunk.UsageMetadata.TotalTokenCount
			}
		}

		// Report token usage before goroutine exits
		if onTokenUsage != nil {
			onTokenUsage(types.TokenUsage{
				InputTokens:  inputTokens,
				OutputTokens: outputTokens,
				TotalTokens:  totalTokens,
			})
		}
	}()

	return responseChan, nil
}

// ListModels lists available Gemini models
func (gc *GeminiClient) ListModels() ([]string, error) {
	return []string{
		"gemini-2.0-flash-exp",
		"gemini-2.0-flash",
		"gemini-2.5-flash-preview",
		"gemini-2.5-pro-preview",
		"gemini-3-flash-preview",
		"gemini-3-pro-preview",
	}, nil
}
