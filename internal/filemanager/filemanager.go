package filemanager

import (
	"fmt"
	"os"
	"path/filepath"
)

// FileManager handles all file system operations
type FileManager struct {
	workspaceDir string
}

// NewFileManager creates a new file manager with the specified workspace directory
func NewFileManager(workspaceDir string) *FileManager {
	return &FileManager{
		workspaceDir: workspaceDir,
	}
}

// CreateFile creates a new file with optional initial content
func (fm *FileManager) CreateFile(filePath string, content string) error {
	fullPath := fm.resolvePath(filePath)
	
	// Create parent directories if they don't exist
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}
	
	// Create the file
	file, err := os.Create(fullPath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", fullPath, err)
	}
	defer file.Close()
	
	// Write initial content if provided
	if content != "" {
		if _, err := file.WriteString(content); err != nil {
			return fmt.Errorf("failed to write content to file %s: %w", fullPath, err)
		}
	}
	
	return nil
}

// ReadFile reads file content from disk
func (fm *FileManager) ReadFile(filePath string) (string, error) {
	fullPath := fm.resolvePath(filePath)
	
	content, err := os.ReadFile(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("file not found: %s", fullPath)
		}
		if os.IsPermission(err) {
			return "", fmt.Errorf("permission denied: %s", fullPath)
		}
		return "", fmt.Errorf("failed to read file %s: %w", fullPath, err)
	}
	
	return string(content), nil
}

// WriteFile writes content to file
func (fm *FileManager) WriteFile(filePath string, content string) error {
	fullPath := fm.resolvePath(filePath)
	
	// Create parent directories if they don't exist
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}
	
	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		if os.IsPermission(err) {
			return fmt.Errorf("permission denied: %s", fullPath)
		}
		return fmt.Errorf("failed to write file %s: %w", fullPath, err)
	}
	
	return nil
}

// DeleteFile deletes a file from disk
func (fm *FileManager) DeleteFile(filePath string) error {
	fullPath := fm.resolvePath(filePath)
	
	if err := os.Remove(fullPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file not found: %s", fullPath)
		}
		if os.IsPermission(err) {
			return fmt.Errorf("permission denied: %s", fullPath)
		}
		return fmt.Errorf("failed to delete file %s: %w", fullPath, err)
	}
	
	return nil
}

// FileExists checks if a file exists
func (fm *FileManager) FileExists(filePath string) bool {
	fullPath := fm.resolvePath(filePath)
	_, err := os.Stat(fullPath)
	return err == nil
}
// ListFiles returns a list of all files in the workspace directory (recursively)
func (fm *FileManager) ListFiles() ([]string, error) {
	var files []string

	err := filepath.Walk(fm.workspaceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and hidden files/folders
		if info.IsDir() || filepath.Base(path)[0] == '.' {
			if info.IsDir() && filepath.Base(path)[0] == '.' {
				return filepath.SkipDir
			}
			return nil
		}

		// Get relative path from workspace
		relPath, err := filepath.Rel(fm.workspaceDir, path)
		if err != nil {
			return err
		}

		files = append(files, relPath)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}

	return files, nil
}


// resolvePath resolves a relative path to an absolute path within the workspace
func (fm *FileManager) resolvePath(path string) string {
	// Clean the path to prevent directory traversal
	cleanPath := filepath.Clean(path)
	
	// If path is already absolute, return it
	if filepath.IsAbs(cleanPath) {
		return cleanPath
	}
	
	// Join with workspace directory
	return filepath.Join(fm.workspaceDir, cleanPath)
}
