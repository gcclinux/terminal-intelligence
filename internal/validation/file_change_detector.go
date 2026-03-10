package validation

import (
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// FileChangeDetector captures file modification events from AI operations
type FileChangeDetector struct {
	mu            sync.RWMutex
	modifiedFiles []string
	callbacks     []func(FileChangeEvent)
	nonCodeExts   map[string]bool
}

// NewFileChangeDetector creates a new FileChangeDetector instance
func NewFileChangeDetector() *FileChangeDetector {
	return &FileChangeDetector{
		modifiedFiles: make([]string, 0),
		callbacks:     make([]func(FileChangeEvent), 0),
		nonCodeExts: map[string]bool{
			".md":   true,
			".txt":  true,
			".json": true,
			".yaml": true,
			".yml":  true,
			".xml":  true,
			".html": true,
			".css":  true,
			".svg":  true,
			".png":  true,
			".jpg":  true,
			".jpeg": true,
			".gif":  true,
		},
	}
}

// OnFileChange subscribes to file change events
func (fcd *FileChangeDetector) OnFileChange(callback func(FileChangeEvent)) {
	fcd.mu.Lock()
	defer fcd.mu.Unlock()
	fcd.callbacks = append(fcd.callbacks, callback)
}

// NotifyFileChange notifies all subscribers of a file change event
func (fcd *FileChangeDetector) NotifyFileChange(event FileChangeEvent) {
	// Filter out non-code files early
	if fcd.isNonCodeFile(event.FilePath) {
		return
	}

	fcd.mu.Lock()
	// Add to modified files list
	fcd.modifiedFiles = append(fcd.modifiedFiles, event.FilePath)
	callbacks := make([]func(FileChangeEvent), len(fcd.callbacks))
	copy(callbacks, fcd.callbacks)
	fcd.mu.Unlock()

	// Notify all callbacks
	for _, callback := range callbacks {
		callback(event)
	}
}

// GetModifiedFiles returns the list of files modified in the current batch
func (fcd *FileChangeDetector) GetModifiedFiles() []string {
	fcd.mu.RLock()
	defer fcd.mu.RUnlock()

	files := make([]string, len(fcd.modifiedFiles))
	copy(files, fcd.modifiedFiles)
	return files
}

// ClearModifiedFiles clears the list of modified files (for starting a new batch)
func (fcd *FileChangeDetector) ClearModifiedFiles() {
	fcd.mu.Lock()
	defer fcd.mu.Unlock()
	fcd.modifiedFiles = make([]string, 0)
}

// isNonCodeFile checks if a file should be filtered out based on extension
func (fcd *FileChangeDetector) isNonCodeFile(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	return fcd.nonCodeExts[ext]
}

// RecordFileChange is a convenience method to record a file change event
func (fcd *FileChangeDetector) RecordFileChange(filePath string, operation Operation) {
	event := FileChangeEvent{
		FilePath:  filePath,
		Operation: operation,
		Timestamp: time.Now(),
	}
	fcd.NotifyFileChange(event)
}
