package validation

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

// Pipeline coordinates the entire validation workflow from file change detection to result display
type Pipeline struct {
	fileChangeDetector   *FileChangeDetector
	validationEngine     *ValidationEngine
	languageDetector     *LanguageDetector
	compilerInterface    *CompilerInterface
	chatPanelIntegration *ChatPanelIntegration

	// Validation queue for concurrent requests
	validationQueue chan []string
	queueMu         sync.Mutex
	isProcessing    bool

	// Context for cancellation
	ctx        context.Context
	cancelFunc context.CancelFunc
}

// NewPipeline creates a new validation pipeline with all components wired together
func NewPipeline() *Pipeline {
	// Create all components
	languageDetector := NewLanguageDetector()
	compilerInterface := NewCompilerInterface()
	chatPanelIntegration := NewChatPanelIntegration()
	validationEngine := NewValidationEngine(languageDetector, compilerInterface, chatPanelIntegration)
	fileChangeDetector := NewFileChangeDetector()

	ctx, cancel := context.WithCancel(context.Background())

	pipeline := &Pipeline{
		fileChangeDetector:   fileChangeDetector,
		validationEngine:     validationEngine,
		languageDetector:     languageDetector,
		compilerInterface:    compilerInterface,
		chatPanelIntegration: chatPanelIntegration,
		validationQueue:      make(chan []string, 10), // Buffer up to 10 validation requests
		ctx:                  ctx,
		cancelFunc:           cancel,
	}

	// Wire file change detector to validation engine
	fileChangeDetector.OnFileChange(pipeline.handleFileChange)

	// Start queue processor
	go pipeline.processValidationQueue()

	return pipeline
}

// handleFileChange is called when files are changed by the AI
func (p *Pipeline) handleFileChange(event FileChangeEvent) {
	// For now, we trigger validation on modify and create operations
	// Delete operations don't need validation
	if event.Operation == OperationDelete {
		return
	}

	// Get all modified files from the current batch
	files := p.fileChangeDetector.GetModifiedFiles()

	// Queue validation
	p.QueueValidation(files)
}

// QueueValidation adds files to the validation queue
func (p *Pipeline) QueueValidation(files []string) {
	if len(files) == 0 {
		return
	}

	// Non-blocking send to queue
	select {
	case p.validationQueue <- files:
		// Successfully queued
	default:
		// Queue is full, log or handle error
		// For now, we'll just skip this validation
		fmt.Println("Warning: Validation queue is full, skipping validation")
	}
}

// processValidationQueue processes validation requests from the queue
func (p *Pipeline) processValidationQueue() {
	for {
		select {
		case <-p.ctx.Done():
			// Pipeline is shutting down
			return
		case files := <-p.validationQueue:
			// Process validation
			p.queueMu.Lock()
			p.isProcessing = true
			p.queueMu.Unlock()

			// Execute validation
			_, err := p.validationEngine.ValidateFiles(files)
			if err != nil {
				// Handle validation error
				p.handleValidationError(err)
			}

			p.queueMu.Lock()
			p.isProcessing = false
			p.queueMu.Unlock()
		}
	}
}

// handleValidationError handles errors that occur during validation
func (p *Pipeline) handleValidationError(err error) {
	// Determine error type and display appropriate message
	message := ""

	switch {
	case err == nil:
		return
	case err == context.DeadlineExceeded:
		message = "⚠️ Validation timed out. The validation process took too long to complete."
	case err == context.Canceled:
		message = "🛑 Validation cancelled."
	default:
		// Check for specific error patterns
		errMsg := err.Error()
		switch {
		case strings.Contains(errMsg, "command not found") || strings.Contains(errMsg, "executable file not found"):
			// Extract command name if possible
			message = fmt.Sprintf("❌ Validator not found: %s. Please ensure the required compiler/validator is installed.", errMsg)
		case strings.Contains(errMsg, "permission denied"):
			message = fmt.Sprintf("❌ Permission denied: %s", errMsg)
		case strings.Contains(errMsg, "no such file or directory"):
			message = fmt.Sprintf("⚠️ File not found: %s (may have been deleted)", errMsg)
		case strings.Contains(errMsg, "invalid file path"):
			message = fmt.Sprintf("❌ Invalid file path: %s", errMsg)
		default:
			message = fmt.Sprintf("⚠️ Validation error: %s", errMsg)
		}
	}

	// Display error message in chat panel
	p.chatPanelIntegration.addMessage(message)
}

// TriggerValidation manually triggers validation for a set of files
// This is useful for testing or manual validation requests
func (p *Pipeline) TriggerValidation(files []string) (*ValidationSession, error) {
	return p.validationEngine.ValidateFiles(files)
}

// GetValidationEngine returns the validation engine (for testing)
func (p *Pipeline) GetValidationEngine() *ValidationEngine {
	return p.validationEngine
}

// GetFileChangeDetector returns the file change detector (for testing)
func (p *Pipeline) GetFileChangeDetector() *FileChangeDetector {
	return p.fileChangeDetector
}

// GetChatPanelIntegration returns the chat panel integration (for testing)
func (p *Pipeline) GetChatPanelIntegration() *ChatPanelIntegration {
	return p.chatPanelIntegration
}

// IsProcessing returns whether the pipeline is currently processing a validation
func (p *Pipeline) IsProcessing() bool {
	p.queueMu.Lock()
	defer p.queueMu.Unlock()
	return p.isProcessing
}

// Shutdown gracefully shuts down the pipeline
func (p *Pipeline) Shutdown() {
	p.cancelFunc()
}
