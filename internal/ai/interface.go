package ai

// AIClient is an interface for AI service providers
type AIClient interface {
	// IsAvailable checks if the AI service is available
	IsAvailable() (bool, error)
	
	// Generate generates AI response with streaming
	Generate(prompt string, model string, context []int) (<-chan string, error)
	
	// ListModels lists available models
	ListModels() ([]string, error)
}
