package ai

import "github.com/user/terminal-intelligence/internal/types"

// AIClient is an interface for AI service providers
type AIClient interface {
	// IsAvailable checks if the AI service is available
	IsAvailable() (bool, error)

	// Generate generates AI response with streaming
	Generate(prompt string, model string, context []int, onTokenUsage func(types.TokenUsage)) (<-chan string, error)

	// ListModels lists available models
	ListModels() ([]string, error)
}
