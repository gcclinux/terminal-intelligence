package bedrock

import (
	stdcontext "context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/aws"
	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awsbedrock "github.com/aws/aws-sdk-go-v2/service/bedrock"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
)

// BedrockClient handles communication with AWS Bedrock API
type BedrockClient struct {
	client        *bedrockruntime.Client
	bedrockClient *awsbedrock.Client
	region        string
}

// anthropicRequest represents the request body for Claude models on AWS Bedrock
type anthropicRequest struct {
	AnthropicVersion string             `json:"anthropic_version"`
	MaxTokens        int                `json:"max_tokens"`
	Messages         []anthropicMessage `json:"messages"`
}

// anthropicMessage represents a message in the Anthropic Messages API format
type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// NewBedrockClient creates a new Bedrock client
// Args:
//
//	apiKey: string - AWS credentials in format "ACCESS_KEY_ID:SECRET_ACCESS_KEY"
//	region: string - AWS region (default: "us-east-1")
//
// Returns: initialized BedrockClient, error if apiKey is empty or invalid format
func NewBedrockClient(apiKey, region string) (*BedrockClient, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("Bedrock API key not provided")
	}

	// Parse API key in format "ACCESS_KEY_ID:SECRET_ACCESS_KEY"
	parts := splitAPIKey(apiKey)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid API key format: expected ACCESS_KEY_ID:SECRET_ACCESS_KEY")
	}

	accessKeyID := parts[0]
	secretAccessKey := parts[1]

	if accessKeyID == "" || secretAccessKey == "" {
		return nil, fmt.Errorf("invalid API key format: both access key ID and secret access key are required")
	}

	if region == "" {
		region = "us-east-1"
	}

	// Create AWS config with static credentials
	cfg, err := config.LoadDefaultConfig(stdcontext.TODO(),
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			accessKeyID,
			secretAccessKey,
			"", // Session token (not used)
		)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create Bedrock Runtime client
	client := bedrockruntime.NewFromConfig(cfg)

	// Create Bedrock client for control plane operations (like ListFoundationModels)
	bedrockClient := awsbedrock.NewFromConfig(cfg)

	return &BedrockClient{
		client:        client,
		bedrockClient: bedrockClient,
		region:        region,
	}, nil
}

// splitAPIKey splits the API key by colon separator
func splitAPIKey(apiKey string) []string {
	result := []string{}
	current := ""

	for _, char := range apiKey {
		if char == ':' {
			result = append(result, current)
			current = ""
		} else {
			current += string(char)
		}
	}

	if current != "" || len(result) > 0 {
		result = append(result, current)
	}

	return result
}

// convertToInferenceProfile converts direct model IDs to inference profile IDs for Claude 4+ models
// Claude 4.5+ models require inference profiles instead of direct model IDs
// Args:
//
//	modelID: string - the model ID (e.g., "anthropic.claude-haiku-4-5-v1:0" or "anthropic.claude-sonnet-4-6")
//	region: string - the AWS region
//
// Returns: inference profile ID or original model ID if not a Claude 4+ model
func convertToInferenceProfile(modelID string, region string) string {
	// Map of Claude 4+ model IDs to their inference profile patterns
	claude4Models := map[string]string{
		// Claude 4.5 models (with version suffix)
		"anthropic.claude-haiku-4-5-v1:0":           "anthropic.claude-haiku-4-5-v1:0",
		"anthropic.claude-sonnet-4-5-v1:0":          "anthropic.claude-sonnet-4-5-v1:0",
		"anthropic.claude-opus-4-5-v1:0":            "anthropic.claude-opus-4-5-v1:0",
		"anthropic.claude-3-5-sonnet-20241022-v2:0": "anthropic.claude-3-5-sonnet-20241022-v2:0",
		"anthropic.claude-3-5-haiku-20241022-v1:0":  "anthropic.claude-3-5-haiku-20241022-v1:0",
		// Claude 4.6+ models (new naming convention without date/version)
		"anthropic.claude-sonnet-4-6": "anthropic.claude-sonnet-4-6",
		"anthropic.claude-haiku-4-6":  "anthropic.claude-haiku-4-6",
		"anthropic.claude-opus-4-6":   "anthropic.claude-opus-4-6",
	}

	// Check if this is a Claude 4+ model that needs an inference profile
	if baseModel, isClaude4 := claude4Models[modelID]; isClaude4 {
		// Use geographic inference profile format: {region}.{model-id}
		// This provides regional inference with cross-region failover
		return fmt.Sprintf("us.%s", baseModel)
	}

	// Return original model ID for older models
	return modelID
}

// formatAWSError formats AWS SDK errors with HTTP status codes and descriptive messages
func formatAWSError(err error, operation string) error {
	if err == nil {
		return nil
	}

	// Try to extract HTTP response error
	var httpErr *awshttp.ResponseError
	if errors.As(err, &httpErr) {
		statusCode := httpErr.Response.StatusCode

		// Format error message based on status code
		var message string
		switch statusCode {
		case http.StatusBadRequest: // 400
			message = "Bad Request - Invalid request format or parameters"
		case http.StatusForbidden: // 403
			message = "Invalid credentials or insufficient permissions"
		case http.StatusNotFound: // 404
			message = "Model may be invalid or unavailable in region"
		case http.StatusTooManyRequests: // 429
			message = "Rate limit exceeded - please retry after a delay"
		case http.StatusInternalServerError: // 500
			message = "AWS Bedrock service error"
		default:
			message = fmt.Sprintf("HTTP error: %s", httpErr.Error())
		}

		return fmt.Errorf("Bedrock %s (status %d): %s", operation, statusCode, message)
	}

	// If not an HTTP error, return the original error with context
	return fmt.Errorf("Bedrock %s: %w", operation, err)
}

// buildRequestBody constructs the request body for Claude models on AWS Bedrock
// Args:
//
//	prompt: string - user's prompt to the AI
//
// Returns: anthropicRequest struct ready to be marshaled to JSON
func buildRequestBody(prompt string) anthropicRequest {
	return anthropicRequest{
		AnthropicVersion: "bedrock-2023-05-31",
		MaxTokens:        4096,
		Messages: []anthropicMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}
}

// IsAvailable checks if Bedrock API is available
// Returns: true if Bedrock is reachable, error if not
func (bc *BedrockClient) IsAvailable() (bool, error) {
	// The API key validation is already done in NewBedrockClient
	// Here we attempt a lightweight API call to verify connectivity

	// Use ListFoundationModels as a lightweight check
	// This verifies both connectivity and credentials
	ctx := stdcontext.TODO()
	_, err := bc.bedrockClient.ListFoundationModels(ctx, &awsbedrock.ListFoundationModelsInput{})

	if err != nil {
		return false, formatAWSError(err, "IsAvailable")
	}

	return true, nil
}

// Generate generates AI response with streaming
// Args:
//
//	prompt: string - user's prompt to the AI
//	model: string - model name to use (default: "us.anthropic.claude-haiku-4-5-v1:0")
//	context: []int - optional context tokens from previous conversation
//
// Returns: channel for streaming response chunks, error if request fails
func (bc *BedrockClient) Generate(prompt string, model string, context []int) (<-chan string, error) {
	// Default model to Claude Haiku 4.5 if empty
	if model == "" {
		model = "us.anthropic.claude-haiku-4-5-v1:0"
	}

	// Convert direct model IDs to inference profile IDs for Claude 4.5 models
	model = convertToInferenceProfile(model, bc.region)

	// Construct request body using anthropicRequest struct
	requestBody := buildRequestBody(prompt)

	// Marshal request body to JSON
	requestBodyJSON, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Call InvokeModelWithResponseStream API
	ctx := stdcontext.Background()
	output, err := bc.client.InvokeModelWithResponseStream(ctx, &bedrockruntime.InvokeModelWithResponseStreamInput{
		ModelId:     &model,
		Body:        requestBodyJSON,
		ContentType: aws.String("application/json"),
	})
	if err != nil {
		return nil, formatAWSError(err, "Generate")
	}

	// Create buffered channel (capacity 10) for response chunks
	responseChan := make(chan string, 10)

	// Launch goroutine to process event stream
	go func() {
		defer close(responseChan)
		defer output.GetStream().Close()

		// Track if we encountered an error during processing
		var processingErr error

		// Process event stream
		for event := range output.GetStream().Events() {
			switch e := event.(type) {
			case *types.ResponseStreamMemberChunk:
				// Parse the chunk bytes to extract text content
				var response map[string]any
				if err := json.Unmarshal(e.Value.Bytes, &response); err != nil {
					// Event parsing failure - send error and stop processing
					processingErr = fmt.Errorf("error parsing response: %w", err)
					responseChan <- fmt.Sprintf("Error parsing response: %v", err)
					return
				}

				// Check for content_block_delta event type
				if eventType, ok := response["type"].(string); ok && eventType == "content_block_delta" {
					// Extract delta object
					if delta, ok := response["delta"].(map[string]any); ok {
						// Extract text from delta
						if text, ok := delta["text"].(string); ok && text != "" {
							responseChan <- text
						}
					}
				}

				// Check for message_stop event to close channel
				if eventType, ok := response["type"].(string); ok && eventType == "message_stop" {
					return
				}

			default:
				// Ignore other event types (including error events which are handled by stream.Err())
			}
		}

		// Check for stream errors after processing completes
		// This catches errors that occurred during streaming but weren't event parsing errors
		if processingErr == nil {
			if err := output.GetStream().Err(); err != nil {
				formattedErr := formatAWSError(err, "streaming")
				responseChan <- fmt.Sprintf("Stream error: %v", formattedErr)
			}
		}
	}()

	// Return channel immediately
	return responseChan, nil
}

// ListModels lists available Bedrock models
// Returns: slice of model names, error if request fails
func (bc *BedrockClient) ListModels() ([]string, error) {
	// Return hardcoded list of supported Claude models
	// Note: Claude 4+ models will be automatically converted to inference profiles
	return []string{
		"anthropic.claude-sonnet-4-6",
		"anthropic.claude-haiku-4-6",
		"anthropic.claude-opus-4-6",
		"anthropic.claude-haiku-4-5-v1:0",
		"anthropic.claude-sonnet-4-5-v1:0",
		"anthropic.claude-3-5-sonnet-20241022-v2:0",
		"anthropic.claude-3-5-haiku-20241022-v1:0",
	}, nil
}
