package docgen

import (
	"fmt"
	"os"
	"path/filepath"
)

// FileWriter handles writing generated documentation to files
type FileWriter struct {
	workspaceRoot string
}

// NewFileWriter creates a new FileWriter for the given workspace root
func NewFileWriter(workspaceRoot string) *FileWriter {
	return &FileWriter{
		workspaceRoot: workspaceRoot,
	}
}

// Write saves a generated document to a file
// If overwrite is false and the file exists, returns an error without writing
// If overwrite is true, writes the file regardless of existence
func (w *FileWriter) Write(doc *GeneratedDoc, overwrite bool) (*WriteResult, error) {
	if doc == nil {
		return nil, fmt.Errorf("document is nil")
	}

	// Construct full path
	fullPath := filepath.Join(w.workspaceRoot, doc.Filename)

	// Check if file exists
	existed := w.CheckExists(doc.Filename)

	// If file exists and overwrite is false, return conflict error
	if existed && !overwrite {
		return &WriteResult{
			Filename: doc.Filename,
			Path:     fullPath,
			Existed:  true,
			Written:  false,
		}, fmt.Errorf("file %s already exists and overwrite is disabled", doc.Filename)
	}

	// Write the file
	err := os.WriteFile(fullPath, []byte(doc.Content), 0644)
	if err != nil {
		return &WriteResult{
			Filename: doc.Filename,
			Path:     fullPath,
			Existed:  existed,
			Written:  false,
		}, fmt.Errorf("failed to write file %s: %w", doc.Filename, err)
	}

	return &WriteResult{
		Filename: doc.Filename,
		Path:     fullPath,
		Existed:  existed,
		Written:  true,
	}, nil
}

// CheckExists checks if a file with the given filename exists in the workspace root
func (w *FileWriter) CheckExists(filename string) bool {
	fullPath := filepath.Join(w.workspaceRoot, filename)
	_, err := os.Stat(fullPath)
	return err == nil
}
