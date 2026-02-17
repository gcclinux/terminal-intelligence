package types

import (
	"os"
	"path/filepath"
	"time"
)

// PaneType represents the type of pane in the split-window interface
type PaneType int

const (
	EditorPaneType PaneType = iota
	AIPaneType
	AIResponsePaneType
)

// AppConfig holds application configuration
type AppConfig struct {
	Provider     string `yaml:"provider"`      // "ollama" or "gemini"
	OllamaURL    string `yaml:"ollama_url"`
	GeminiAPIKey string `yaml:"gemini_api_key"`
	DefaultModel string `yaml:"default_model"`
	EditorTheme  string `yaml:"editor_theme"`
	WorkspaceDir string `yaml:"workspace_dir"`
	AutoSave     bool   `yaml:"auto_save"`
	TabSize      int    `yaml:"tab_size"`
}

// DefaultConfig returns default application configuration
func DefaultConfig() *AppConfig {
	homeDir, _ := os.UserHomeDir()
	return &AppConfig{
		Provider:     "ollama",
		OllamaURL:    "http://localhost:11434",
		GeminiAPIKey: "",
		DefaultModel: "llama2",
		EditorTheme:  "monokai",
		WorkspaceDir: filepath.Join(homeDir, "ti-workspace"),
		AutoSave:     false,
		TabSize:      4,
	}
}

// FileMetadata holds metadata about a file
type FileMetadata struct {
	Filepath   string
	FileType   string // "bash", "shell", "powershell", "markdown"
	IsModified bool
	LastSaved  time.Time
}

// ChatMessage represents a message in the AI conversation
type ChatMessage struct {
	Role            string // "user" or "assistant"
	Content         string
	Timestamp       time.Time
	ContextIncluded bool
	IsNotification  bool   // True if this is a change notification
	IsFixRequest    bool   // True if this is a fix request
	FilePath        string // File path context for fix requests
}

// CommandResult holds the result of a command execution
type CommandResult struct {
	Stdout        string
	Stderr        string
	ExitCode      int
	ExecutionTime time.Duration
}
