package ollama

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// OllamaClient handles communication with the Ollama service via REST API
type OllamaClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewOllamaClient creates a new Ollama client with configurable base URL
// Args:
//   baseURL: string - base URL for Ollama API (default: "http://localhost:11434")
// Returns: initialized OllamaClient
func NewOllamaClient(baseURL string) *OllamaClient {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	return &OllamaClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// IsAvailable checks if Ollama service is available
// Returns: true if Ollama is reachable, error if not
func (oc *OllamaClient) IsAvailable() (bool, error) {
	resp, err := oc.httpClient.Get(oc.baseURL + "/api/tags")
	if err != nil {
		return false, fmt.Errorf("failed to connect to Ollama service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("Ollama service returned status %d", resp.StatusCode)
	}

	return true, nil
}

// generateRequest represents the request body for the generate API
type generateRequest struct {
	Model   string `json:"model"`
	Prompt  string `json:"prompt"`
	Stream  bool   `json:"stream"`
	Context []int  `json:"context,omitempty"`
}

// generateResponse represents a streaming response chunk from the generate API
type generateResponse struct {
	Model     string `json:"model"`
	Response  string `json:"response"`
	Done      bool   `json:"done"`
	Context   []int  `json:"context,omitempty"`
	Error     string `json:"error,omitempty"`
}

// Generate generates AI response with streaming
// Args:
//   prompt: string - user's prompt to the AI
//   model: string - model name to use (default: "llama2")
//   context: []int - optional context tokens from previous conversation
// Returns: channel for streaming response chunks, error if request fails
func (oc *OllamaClient) Generate(prompt string, model string, context []int) (<-chan string, error) {
	if model == "" {
		model = "llama2"
	}

	reqBody := generateRequest{
		Model:   model,
		Prompt:  prompt,
		Stream:  true,
		Context: context,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", oc.baseURL+"/api/generate", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Use a client without timeout for streaming
	streamClient := &http.Client{
		Timeout: 0, // No timeout for streaming
	}

	resp, err := streamClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("Ollama API returned status %d", resp.StatusCode)
	}

	// Create channel for streaming responses
	responseChan := make(chan string, 10)

	// Start goroutine to read streaming response
	go func() {
		defer close(responseChan)
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Bytes()
			
			var genResp generateResponse
			if err := json.Unmarshal(line, &genResp); err != nil {
				// Send error message to channel
				responseChan <- fmt.Sprintf("Error parsing response: %v", err)
				return
			}

			// Check for API error
			if genResp.Error != "" {
				responseChan <- fmt.Sprintf("API Error: %s", genResp.Error)
				return
			}

			// Send response chunk to channel
			if genResp.Response != "" {
				responseChan <- genResp.Response
			}

			// Stop if done
			if genResp.Done {
				return
			}
		}

		if err := scanner.Err(); err != nil {
			responseChan <- fmt.Sprintf("Error reading response: %v", err)
		}
	}()

	return responseChan, nil
}

// modelsResponse represents the response from the list models API
type modelsResponse struct {
	Models []modelInfo `json:"models"`
}

// modelInfo represents information about a single model
type modelInfo struct {
	Name       string    `json:"name"`
	ModifiedAt time.Time `json:"modified_at"`
	Size       int64     `json:"size"`
}

// ListModels lists available models in Ollama
// Returns: slice of model names, error if request fails
func (oc *OllamaClient) ListModels() ([]string, error) {
	resp, err := oc.httpClient.Get(oc.baseURL + "/api/tags")
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Ollama service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Ollama API returned status %d: %s", resp.StatusCode, string(body))
	}

	var modelsResp modelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	modelNames := make([]string, len(modelsResp.Models))
	for i, model := range modelsResp.Models {
		modelNames[i] = model.Name
	}

	return modelNames, nil
}
