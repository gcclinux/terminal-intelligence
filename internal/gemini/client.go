package gemini

import (
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
			Timeout: 30 * time.Second,
		},
	}
}

// NewGeminiClientWithURL creates a new Gemini client with a custom base URL (for testing)
func NewGeminiClientWithURL(apiKey string, baseURL string) *GeminiClient {
	return &GeminiClient{
		apiKey:  apiKey,
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
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

// geminiResponse represents the response from Gemini API
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

// Generate generates AI response from Gemini
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

	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s", gc.baseURL, model, gc.apiKey)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := gc.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Gemini API returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var geminiResp geminiResponse
	if err := json.Unmarshal(body, &geminiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Check for API error
	if geminiResp.Error != nil {
		return nil, fmt.Errorf("Gemini API error: %s", geminiResp.Error.Message)
	}

	// Create channel for response
	responseChan := make(chan string, 1)

	go func() {
		defer close(responseChan)
		
		// Extract and report token usage from usageMetadata
		if onTokenUsage != nil {
			var inputTokens, outputTokens, totalTokens int
			if geminiResp.UsageMetadata != nil {
				inputTokens = geminiResp.UsageMetadata.PromptTokenCount
				outputTokens = geminiResp.UsageMetadata.CandidatesTokenCount
				totalTokens = geminiResp.UsageMetadata.TotalTokenCount
			}
			onTokenUsage(types.TokenUsage{
				InputTokens:  inputTokens,
				OutputTokens: outputTokens,
				TotalTokens:  totalTokens,
			})
		}

		if len(geminiResp.Candidates) > 0 && len(geminiResp.Candidates[0].Content.Parts) > 0 {
			responseChan <- geminiResp.Candidates[0].Content.Parts[0].Text
		}
	}()

	return responseChan, nil
}

// ListModels lists available Gemini models
func (gc *GeminiClient) ListModels() ([]string, error) {
	return []string{
		"gemini-2.0-flash-exp",
		"gemini-1.5-flash",
		"gemini-1.5-pro",
	}, nil
}
