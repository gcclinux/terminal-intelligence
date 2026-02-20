package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/user/terminal-intelligence/internal/config"
	"github.com/user/terminal-intelligence/internal/types"
	"github.com/user/terminal-intelligence/internal/ui"
)

var version = "0.0.1.0"

// Build number (injected at compile time via -ldflags)
var buildNumber = "dev"

func main() {
	// Define CLI flags
	workspaceDir := flag.String("workspace", "", "Workspace directory for file operations (default: ~/ti-workspace)")
	ollamaURL := flag.String("ollama", "", "Ollama API URL (default: http://localhost:11434)")
	geminiAPIKey := flag.String("gemini", "", "Gemini API key (use this to enable Gemini instead of Ollama)")
	model := flag.String("model", "", "Default model to use (default: llama2 for Ollama, gemini-2.0-flash-exp for Gemini)")
	showVersion := flag.Bool("version", false, "Show version information")
	showHelp := flag.Bool("help", false, "Show help information")

	flag.Parse()

	// Show version
	if *showVersion {
		fmt.Printf("Terminal Intelligence (TI) version %s\n", version)
		os.Exit(0)
	}

	// Show help
	if *showHelp {
		fmt.Println("Terminal Intelligence (TI) - CLI-based IDE with AI assistance")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Printf("  %s [options]\n", os.Args[0])
		fmt.Println()
		fmt.Println("Options:")
		flag.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  # Use Ollama (default)")
		fmt.Println("  ./ti -model qwen2.5-coder:1.5b -ollama http://localhost:11434 -workspace ~/ti-workspace")
		fmt.Println()
		fmt.Println("  # Use Gemini")
		fmt.Println("  ./ti -model gemini-2.0-flash-exp -gemini YOUR_API_KEY -workspace ~/ti-workspace")
		fmt.Println()
		fmt.Println("Keyboard Shortcuts:")
		fmt.Println("  Tab         - Switch between editor and AI pane")
		fmt.Println("  Ctrl+S      - Save current file")
		fmt.Println("  Ctrl+R      - Execute current script")
		fmt.Println("  Ctrl+Enter  - Send message to AI")
		fmt.Println("  Ctrl+C      - Copy selected line or AI block")
		fmt.Println("  Ctrl+Q      - Quit application")
		fmt.Println()
		os.Exit(0)
	}

	// Load configuration
	appCfg := types.DefaultConfig()

	// Check for config file and load if present
	configPath, err := config.ConfigFilePath()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error determining config file path: %v\n", err)
		os.Exit(1)
	}

	if _, err := os.Stat(configPath); err == nil {
		// Config file exists — load, validate, and apply
		jcfg, err := config.LoadFromFile(configPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config file: %v\n", err)
			os.Exit(1)
		}
		if err := config.Validate(jcfg); err != nil {
			fmt.Fprintf(os.Stderr, "Invalid config file: %v\n", err)
			os.Exit(1)
		}
		config.ApplyToAppConfig(jcfg, appCfg)
	} else if os.IsNotExist(err) {
		// Config file doesn't exist — create default config
		createdPath, err := config.CreateDefaultConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Could not create default config file: %v\n", err)
			fmt.Fprintf(os.Stderr, "Continuing with command-line flags and defaults...\n")
		} else {
			fmt.Fprintf(os.Stderr, "Created default config file at: %s\n", createdPath)
			fmt.Fprintf(os.Stderr, "Edit this file to customize your settings.\n")
			fmt.Fprintf(os.Stderr, "Continuing with defaults for this session...\n\n")
		}
	}

	// Apply CLI flag overrides on top of config file (or defaults)
	if *geminiAPIKey != "" {
		appCfg.Provider = "gemini"
		appCfg.GeminiAPIKey = *geminiAPIKey
		if *model == "" {
			appCfg.DefaultModel = "gemini-2.0-flash-exp"
		}
	} else if appCfg.Provider != "gemini" {
		// Only default to ollama if config file didn't set gemini
		appCfg.Provider = "ollama"
		if *ollamaURL != "" {
			appCfg.OllamaURL = *ollamaURL
		}
	} else if *ollamaURL != "" {
		// CLI ollama URL override even when config says gemini
		appCfg.OllamaURL = *ollamaURL
	}

	if *workspaceDir != "" {
		appCfg.WorkspaceDir = *workspaceDir
	} else {
		// Default workspace to current working directory
		cwd, err := os.Getwd()
		if err == nil {
			appCfg.WorkspaceDir = cwd
		}
	}
	if *model != "" {
		appCfg.DefaultModel = *model
	}

	// Save updated workspace back to config.json
	jcfgToSave := config.AppConfigToJSONConfig(appCfg)
	if data, err := config.ToJSON(jcfgToSave); err == nil {
		_ = os.WriteFile(configPath, data, 0644)
	}

	// Create workspace directory if it doesn't exist
	if err := os.MkdirAll(appCfg.WorkspaceDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating workspace directory: %v\n", err)
		os.Exit(1)
	}

	// Create the application
	app := ui.New(appCfg, buildNumber)

	// Start the Bubble Tea program
	p := tea.NewProgram(app, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running application: %v\n", err)
		os.Exit(1)
	}
}
