package projectctx

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ContextBuilder scans a workspace directory and assembles project metadata.
type ContextBuilder struct{}

// NewContextBuilder creates a new ContextBuilder.
func NewContextBuilder() *ContextBuilder {
	return &ContextBuilder{}
}

// Build scans the workspace at rootDir and returns ProjectMetadata.
// It respects SkipDirs, reads key project files (truncating at MaxKeyFileBytes),
// generates a file tree (max MaxFileTreeEntries entries), and caps total context
// at MaxTotalContextBytes using priority ordering.
func (cb *ContextBuilder) Build(rootDir string) (*ProjectMetadata, error) {
	// Validate that rootDir exists and is readable.
	info, err := os.Stat(rootDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("workspace directory does not exist: %s", rootDir)
		}
		return nil, fmt.Errorf("workspace directory is unreadable: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("workspace path is not a directory: %s", rootDir)
	}

	// Collect all file paths and detect key project files.
	var allFiles []string
	rawKeyFiles := make(map[string][]byte)

	err = filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// Skip unreadable entries gracefully.
			return nil
		}

		// Get path relative to rootDir.
		rel, relErr := filepath.Rel(rootDir, path)
		if relErr != nil {
			return nil
		}

		// Skip the root directory itself.
		if rel == "." {
			return nil
		}

		// Check if this directory should be skipped.
		if d.IsDir() {
			if SkipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}

		// Collect file path.
		// Use forward slashes for consistency.
		rel = filepath.ToSlash(rel)
		allFiles = append(allFiles, rel)

		// Check if this is a key project file (must be at root level).
		if IsKeyProjectFile(d.Name()) && !strings.Contains(rel, "/") {
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				// Skip unreadable key files gracefully.
				return nil
			}
			if len(data) > MaxKeyFileBytes {
				data = data[:MaxKeyFileBytes]
			}
			rawKeyFiles[rel] = data
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("error scanning workspace: %w", err)
	}

	// Sort files for deterministic output.
	sort.Strings(allFiles)

	totalFiles := len(allFiles)

	// Detect primary language and build system.
	language := detectLanguage(rawKeyFiles)
	buildSystem := detectBuildSystem(rawKeyFiles)

	// Assemble ProjectMetadata with priority-based context capping.
	meta := &ProjectMetadata{
		RootDir:   rootDir,
		Language:  language,
		BuildSystem: buildSystem,
		KeyFiles:  make(map[string]string),
		ScannedAt: time.Now(),
		TotalFiles: totalFiles,
	}

	// Add key files in priority order, respecting the total context limit.
	totalBytes := 0
	for _, name := range KeyProjectFiles {
		data, ok := rawKeyFiles[name]
		if !ok {
			continue
		}
		content := string(data)
		contentBytes := len(content)
		if totalBytes+contentBytes > MaxTotalContextBytes {
			// Try to fit a truncated version.
			remaining := MaxTotalContextBytes - totalBytes
			if remaining > 0 {
				meta.KeyFiles[name] = content[:remaining]
				totalBytes += remaining
			}
			break
		}
		meta.KeyFiles[name] = content
		totalBytes += contentBytes
	}

	// Build file tree, truncating at MaxFileTreeEntries first.
	fileTree := allFiles
	if len(fileTree) > MaxFileTreeEntries {
		fileTree = fileTree[:MaxFileTreeEntries]
	}

	// Further truncate file tree to fit within remaining context budget.
	fileTreeBytes := calcFileTreeBytes(fileTree)
	for totalBytes+fileTreeBytes > MaxTotalContextBytes && len(fileTree) > 0 {
		fileTree = fileTree[:len(fileTree)-1]
		fileTreeBytes = calcFileTreeBytes(fileTree)
	}

	meta.FileTree = fileTree
	meta.FileTreeTruncated = totalFiles > MaxFileTreeEntries
	totalBytes += fileTreeBytes
	meta.TotalContextBytes = totalBytes

	return meta, nil
}

// calcFileTreeBytes returns the total byte count of the file tree entries.
func calcFileTreeBytes(tree []string) int {
	total := 0
	for _, entry := range tree {
		total += len(entry)
	}
	return total
}

// detectLanguage returns the primary language based on key project files found.
func detectLanguage(keyFiles map[string][]byte) string {
	// Check in a deterministic priority order.
	if _, ok := keyFiles["go.mod"]; ok {
		return "go"
	}
	if _, ok := keyFiles["package.json"]; ok {
		return "javascript"
	}
	if _, ok := keyFiles["Cargo.toml"]; ok {
		return "rust"
	}
	if _, ok := keyFiles["pyproject.toml"]; ok {
		return "python"
	}
	if _, ok := keyFiles["requirements.txt"]; ok {
		return "python"
	}
	if _, ok := keyFiles["CMakeLists.txt"]; ok {
		return "c/c++"
	}
	return ""
}

// detectBuildSystem returns the build system based on key project files found.
func detectBuildSystem(keyFiles map[string][]byte) string {
	if _, ok := keyFiles["go.mod"]; ok {
		return "go modules"
	}
	if _, ok := keyFiles["package.json"]; ok {
		return "npm"
	}
	if _, ok := keyFiles["Cargo.toml"]; ok {
		return "cargo"
	}
	if _, ok := keyFiles["pyproject.toml"]; ok {
		return "pip/pyproject"
	}
	if _, ok := keyFiles["Makefile"]; ok {
		return "make"
	}
	if _, ok := keyFiles["CMakeLists.txt"]; ok {
		return "cmake"
	}
	return ""
}
