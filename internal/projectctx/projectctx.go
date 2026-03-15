// Package projectctx provides project-aware context building, caching,
// query classification, and prompt construction for the Terminal Intelligence
// application. It scans workspace directories for key project files, assembles
// structured metadata, and augments AI prompts with real project context.
package projectctx

import (
	"fmt"
	"time"
)

// Context size limits.
const (
	// MaxFileTreeEntries is the maximum number of entries in the file tree listing.
	MaxFileTreeEntries = 200

	// MaxKeyFileBytes is the maximum number of bytes to read from any single key project file.
	MaxKeyFileBytes = 5000

	// MaxTotalContextBytes is the maximum total byte count of the assembled project context.
	MaxTotalContextBytes = 50000
)

// KeyProjectFiles lists the recognised key project files in priority order.
// When the total context approaches MaxTotalContextBytes, higher-priority files
// are kept and lower-priority files are dropped.
//
// Priority groups:
//  1. README files
//  2. Build / dependency files
//  3. Docker files
//  4. Other configuration files
var KeyProjectFiles = []string{
	// Group 1 — README files
	"README.md",
	"README",
	"README.txt",
	"README.rst",

	// Group 2 — Build / dependency files
	"go.mod",
	"go.sum",
	"package.json",
	"package-lock.json",
	"Cargo.toml",
	"pyproject.toml",
	"requirements.txt",
	"setup.py",
	"setup.cfg",
	"CMakeLists.txt",

	// Group 3 — Docker files
	"Dockerfile",
	"docker-compose.yml",
	"docker-compose.yaml",

	// Group 4 — Other configuration files
	".env.example",
	"Makefile",
}

// keyProjectFileSet is a lookup set built from KeyProjectFiles for O(1) membership checks.
var keyProjectFileSet = func() map[string]bool {
	m := make(map[string]bool, len(KeyProjectFiles))
	for _, f := range KeyProjectFiles {
		m[f] = true
	}
	return m
}()

// IsKeyProjectFile reports whether name is a recognised key project file.
func IsKeyProjectFile(name string) bool {
	return keyProjectFileSet[name]
}

// SkipDirs is the set of directory names that should be skipped during workspace scanning.
var SkipDirs = map[string]bool{
	".git":         true,
	"vendor":       true,
	"node_modules": true,
	".ti":          true,
	"build":        true,
}

// ProjectMetadata holds the assembled project context for a workspace.
// Invariants:
//   - len(FileTree) <= MaxFileTreeEntries
//   - Each value in KeyFiles has len <= MaxKeyFileBytes
//   - TotalContextBytes <= MaxTotalContextBytes
//   - FileTreeTruncated == (TotalFiles > MaxFileTreeEntries)
type ProjectMetadata struct {
	// RootDir is the absolute path to the workspace directory.
	RootDir string

	// Language is the detected primary language (e.g., "go", "python", "javascript").
	Language string

	// BuildSystem is the detected build system (e.g., "go modules", "npm", "cargo", "make").
	BuildSystem string

	// KeyFiles maps relative file paths to their contents (truncated at MaxKeyFileBytes).
	KeyFiles map[string]string

	// FileTree is the listing of discovered files (max MaxFileTreeEntries entries).
	FileTree []string

	// FileTreeTruncated indicates whether the file tree was truncated.
	FileTreeTruncated bool

	// TotalFiles is the total number of files discovered before truncation.
	TotalFiles int

	// ScannedAt is the timestamp when this metadata was built.
	ScannedAt time.Time

	// TotalContextBytes is the total byte count of the assembled context.
	TotalContextBytes int
}

// Validate checks whether the ProjectMetadata satisfies its invariants.
func (pm *ProjectMetadata) Validate() error {
	if len(pm.FileTree) > MaxFileTreeEntries {
		return fmt.Errorf("FileTree length %d exceeds MaxFileTreeEntries (%d)", len(pm.FileTree), MaxFileTreeEntries)
	}

	for path, content := range pm.KeyFiles {
		if len(content) > MaxKeyFileBytes {
			return fmt.Errorf("KeyFile %q length %d exceeds MaxKeyFileBytes (%d)", path, len(content), MaxKeyFileBytes)
		}
	}

	if pm.TotalContextBytes > MaxTotalContextBytes {
		return fmt.Errorf("TotalContextBytes %d exceeds MaxTotalContextBytes (%d)", pm.TotalContextBytes, MaxTotalContextBytes)
	}

	shouldBeTruncated := pm.TotalFiles > MaxFileTreeEntries
	if pm.FileTreeTruncated != shouldBeTruncated {
		return fmt.Errorf("FileTreeTruncated is %v but TotalFiles is %d (expected truncated=%v)",
			pm.FileTreeTruncated, pm.TotalFiles, shouldBeTruncated)
	}

	return nil
}

// ClassificationResult holds the outcome of query classification.
type ClassificationResult struct {
	// NeedsProjectContext is true if the message is a project-level question.
	NeedsProjectContext bool

	// SearchTerms contains extracted search terms if search-like intent was detected.
	SearchTerms []string
}
