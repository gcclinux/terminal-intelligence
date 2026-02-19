package filemanager

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
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

	// Create backup if file exists (safeguard for overwrites)
	if _, err := os.Stat(fullPath); err == nil {
		if err := fm.createBackup(fullPath); err != nil {
			return fmt.Errorf("failed to create backup: %w", err)
		}
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

	// Create backup if file exists
	if _, err := os.Stat(fullPath); err == nil {
		if err := fm.createBackup(fullPath); err != nil {
			// Log error but proceed? Or fail?
			// User requirement implies safety is key. Let's fail if backup fails to ensure we don't lose data.
			return fmt.Errorf("failed to create backup: %w", err)
		}
	}

	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		if os.IsPermission(err) {
			return fmt.Errorf("permission denied: %s", fullPath)
		}
		return fmt.Errorf("failed to write file %s: %w", fullPath, err)
	}

	return nil
}

// createBackup creates a copy of the file in the .ti directory
func (fm *FileManager) createBackup(fullPath string) error {
	relPath, err := filepath.Rel(fm.workspaceDir, fullPath)
	if err != nil {
		return err
	}
	// Normalize path separators
	relPath = filepath.ToSlash(relPath)

	// Don't backup files that are already in the .ti directory
	if strings.HasPrefix(relPath, ".ti/") {
		return nil
	}

	tiDir := filepath.Join(fm.workspaceDir, ".ti")
	// Check if .ti folder exists, if not create it
	if _, err := os.Stat(tiDir); os.IsNotExist(err) {
		if err := os.MkdirAll(tiDir, 0755); err != nil {
			return fmt.Errorf("failed to create .ti directory: %w", err)
		}
		// Since we just created the directory, check/update .gitignore
		if err := fm.ensureGitIgnore(tiDir); err != nil {
			// Log error but proceed
			fmt.Printf("Warning: failed to update .gitignore: %v\n", err)
		}
	}

	content, err := os.ReadFile(fullPath)
	if err != nil {
		return fmt.Errorf("failed to read file for backup: %w", err)
	}

	// Create timestamped filename: YYYYMMDD-HHMMSS_path_to_file
	// Replace slashes with underscores to flatten directory structure
	timestamp := time.Now().Format("20060102-150405")
	safePath := strings.ReplaceAll(relPath, "/", "_")
	backupName := fmt.Sprintf("%s_%s", timestamp, safePath)
	backupPath := filepath.Join(tiDir, backupName)

	if err := os.WriteFile(backupPath, content, 0644); err != nil {
		return fmt.Errorf("failed to write backup file: %w", err)
	}

	return nil
}

// ListBackups returns a list of backup files for a given file path
func (fm *FileManager) ListBackups(filePath string) ([]string, error) {
	fullPath := fm.resolvePath(filePath)
	relPath, err := filepath.Rel(fm.workspaceDir, fullPath)
	if err != nil {
		return nil, err
	}
	relPath = filepath.ToSlash(relPath)
	safePath := strings.ReplaceAll(relPath, "/", "_")

	tiDir := filepath.Join(fm.workspaceDir, ".ti")
	entries, err := os.ReadDir(tiDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	var backups []string
	suffix := "_" + safePath
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), suffix) {
			backups = append(backups, entry.Name())
		}
	}

	return backups, nil
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

// ensureGitIgnore adds the .ti/ directory to .gitignore if it exists and is missing the entry
func (fm *FileManager) ensureGitIgnore(tiDir string) error {
	gitIgnorePath := filepath.Join(fm.workspaceDir, ".gitignore")

	// Check if .gitignore exists
	if _, err := os.Stat(gitIgnorePath); os.IsNotExist(err) {
		return nil // No .gitignore, nothing to do
	}

	content, err := os.ReadFile(gitIgnorePath)
	if err != nil {
		return fmt.Errorf("failed to read .gitignore: %w", err)
	}

	contentStr := string(content)

	// Check if .ti/ or .ti is already ignored
	// We handle various line ending/spacing scenarios
	lines := strings.Split(contentStr, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == ".ti/" || trimmed == ".ti" {
			return nil // Already ignored
		}
	}

	// Append .ti/ to .gitignore
	// Ensure we start on a new line if the file doesn't end with one
	newContent := contentStr
	if len(newContent) > 0 && !strings.HasSuffix(newContent, "\n") {
		newContent += "\n"
	}
	newContent += ".ti/\n"

	if err := os.WriteFile(gitIgnorePath, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to update .gitignore: %w", err)
	}

	return nil
}
