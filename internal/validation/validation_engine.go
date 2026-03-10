package validation

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ValidationEngine orchestrates the validation pipeline and manages validation state
type ValidationEngine struct {
	languageDetector     *LanguageDetector
	compilerInterface    *CompilerInterface
	chatPanelIntegration *ChatPanelIntegration

	// Session management
	currentSession *ValidationSession
	mu             sync.RWMutex

	// Cancellation support
	cancelFunc context.CancelFunc

	// Progress tracking
	progressTicker *time.Ticker
	progressDone   chan bool
}

// NewValidationEngine creates a new ValidationEngine with all required components
func NewValidationEngine(
	languageDetector *LanguageDetector,
	compilerInterface *CompilerInterface,
	chatPanelIntegration *ChatPanelIntegration,
) *ValidationEngine {
	return &ValidationEngine{
		languageDetector:     languageDetector,
		compilerInterface:    compilerInterface,
		chatPanelIntegration: chatPanelIntegration,
	}
}

// ValidateFiles starts validation for a set of files
// Returns a ValidationSession that tracks the validation progress
func (ve *ValidationEngine) ValidateFiles(files []string) (*ValidationSession, error) {
	if len(files) == 0 {
		// Empty file list - skip validation silently
		return nil, nil
	}

	// Create a new validation session
	session := &ValidationSession{
		ID:        uuid.New().String(),
		Files:     files,
		StartTime: time.Now(),
		Status:    StatusPending,
		Results:   make([]ValidationResult, 0),
	}

	// Store the current session
	ve.mu.Lock()
	ve.currentSession = session
	ve.mu.Unlock()

	// Group files by language
	filesByLanguage := ve.groupFilesByLanguage(files)

	// Separate supported and unsupported files
	supportedFiles := make(map[Language][]string)
	unsupportedFiles := make([]string, 0)

	for lang, langFiles := range filesByLanguage {
		if lang == LanguageUnsupported {
			unsupportedFiles = append(unsupportedFiles, langFiles...)
		} else {
			supportedFiles[lang] = langFiles
		}
	}

	// Show unsupported file notification if any
	if len(unsupportedFiles) > 0 {
		supportedLanguages := ve.languageDetector.GetSupportedLanguages()
		ve.chatPanelIntegration.ShowUnsupportedLanguage(unsupportedFiles, supportedLanguages)
	}

	// If all files are unsupported, complete the session
	if len(supportedFiles) == 0 {
		endTime := time.Now()
		session.EndTime = &endTime
		session.Status = StatusCompleted
		return session, nil
	}

	// Update session status to running
	ve.mu.Lock()
	session.Status = StatusRunning
	ve.mu.Unlock()

	// Create a context for cancellation support
	ctx, cancel := context.WithCancel(context.Background())
	ve.mu.Lock()
	ve.cancelFunc = cancel
	ve.mu.Unlock()

	// Start progress tracking for long-running validations
	ve.startProgressTracking(ctx)

	// Validate each language group
	for lang, langFiles := range supportedFiles {
		// Check if cancelled
		select {
		case <-ctx.Done():
			ve.mu.Lock()
			session.Status = StatusCancelled
			endTime := time.Now()
			session.EndTime = &endTime
			ve.mu.Unlock()
			return session, ctx.Err()
		default:
		}

		// Show validation start message with per-unit status
		ve.chatPanelIntegration.ShowValidationStart(langFiles, lang)

		// For Python, validate each file independently and show per-file status
		if lang == LanguagePython {
			for _, file := range langFiles {
				// Execute validation for single file
				result, err := ve.compilerInterface.ValidateWithContext(ctx, lang, []string{file})
				if err != nil {
					// Validation execution failed
					ve.mu.Lock()
					session.Status = StatusFailed
					endTime := time.Now()
					session.EndTime = &endTime
					ve.mu.Unlock()
					return session, fmt.Errorf("validation failed for %s: %w", lang, err)
				}

				// Add result to session
				ve.mu.Lock()
				session.Results = append(session.Results, result)
				ve.mu.Unlock()

				// Show validation result for this file
				if result.Success {
					ve.chatPanelIntegration.ShowValidationSuccess(result)
				} else {
					ve.chatPanelIntegration.ShowValidationFailure(result)
				}
			}
		} else {
			// For Go and other languages, validate as a package
			result, err := ve.compilerInterface.ValidateWithContext(ctx, lang, langFiles)
			if err != nil {
				// Validation execution failed
				ve.mu.Lock()
				session.Status = StatusFailed
				endTime := time.Now()
				session.EndTime = &endTime
				ve.mu.Unlock()
				return session, fmt.Errorf("validation failed for %s: %w", lang, err)
			}

			// Add result to session
			ve.mu.Lock()
			session.Results = append(session.Results, result)
			ve.mu.Unlock()

			// Show validation result
			if result.Success {
				ve.chatPanelIntegration.ShowValidationSuccess(result)
			} else {
				ve.chatPanelIntegration.ShowValidationFailure(result)
			}
		}
	}

	// Complete the session
	ve.mu.Lock()
	session.Status = StatusCompleted
	endTime := time.Now()
	session.EndTime = &endTime
	ve.mu.Unlock()

	// Stop progress tracking
	ve.stopProgressTracking()

	return session, nil
}

// groupFilesByLanguage groups files by their detected programming language
func (ve *ValidationEngine) groupFilesByLanguage(files []string) map[Language][]string {
	filesByLanguage := make(map[Language][]string)

	for _, file := range files {
		lang := ve.languageDetector.DetectLanguage(file)
		filesByLanguage[lang] = append(filesByLanguage[lang], file)
	}

	return filesByLanguage
}

// GetStatus returns the current validation status
func (ve *ValidationEngine) GetStatus() *ValidationStatus {
	ve.mu.RLock()
	defer ve.mu.RUnlock()

	if ve.currentSession == nil {
		status := StatusPending
		return &status
	}

	return &ve.currentSession.Status
}

// GetCurrentSession returns the current validation session
func (ve *ValidationEngine) GetCurrentSession() *ValidationSession {
	ve.mu.RLock()
	defer ve.mu.RUnlock()

	return ve.currentSession
}

// Cancel cancels the ongoing validation
func (ve *ValidationEngine) Cancel() {
	ve.mu.Lock()

	if ve.cancelFunc != nil {
		ve.cancelFunc()
		ve.cancelFunc = nil
	}

	if ve.currentSession != nil && ve.currentSession.Status == StatusRunning {
		ve.currentSession.Status = StatusCancelled
		endTime := time.Now()
		ve.currentSession.EndTime = &endTime
	}

	// Stop progress tracking (without holding the lock)
	ticker := ve.progressTicker
	done := ve.progressDone
	ve.progressTicker = nil
	ve.progressDone = nil

	ve.mu.Unlock()

	// Stop ticker and close channel outside the lock
	if ticker != nil {
		ticker.Stop()
	}
	if done != nil {
		close(done)
	}
}

// startProgressTracking starts tracking progress for long-running validations
func (ve *ValidationEngine) startProgressTracking(ctx context.Context) {
	ve.mu.Lock()
	defer ve.mu.Unlock()

	// Create a ticker that fires every second
	ve.progressTicker = time.NewTicker(1 * time.Second)
	ve.progressDone = make(chan bool)

	go func() {
		startTime := time.Now()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ve.progressDone:
				return
			case <-ve.progressTicker.C:
				elapsed := time.Since(startTime)
				// Only show progress indicator if validation exceeds 5 seconds
				if elapsed >= 5*time.Second {
					ve.showProgressIndicator(elapsed)
				}
			}
		}
	}()
}

// stopProgressTracking stops the progress tracking
func (ve *ValidationEngine) stopProgressTracking() {
	// Get ticker and done channel while holding lock
	ve.mu.Lock()
	ticker := ve.progressTicker
	done := ve.progressDone
	ve.progressTicker = nil
	ve.progressDone = nil
	ve.mu.Unlock()

	// Stop ticker and close channel outside the lock
	if ticker != nil {
		ticker.Stop()
	}
	if done != nil {
		close(done)
	}
}

// showProgressIndicator displays a progress indicator in the chat panel
func (ve *ValidationEngine) showProgressIndicator(elapsed time.Duration) {
	ve.mu.RLock()
	session := ve.currentSession
	ve.mu.RUnlock()

	if session == nil || session.Status != StatusRunning {
		return
	}

	// Determine which language is currently being validated
	// For simplicity, we'll show a generic progress message
	// In a real implementation, we could track which language is currently being validated
	language := LanguageGo // Default to Go for now
	if len(session.Results) > 0 {
		language = session.Results[len(session.Results)-1].Language
	}

	ve.chatPanelIntegration.ShowValidationProgress(language, elapsed.Seconds())
}
