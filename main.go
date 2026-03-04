package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/user/terminal-intelligence/internal/config"
	"github.com/user/terminal-intelligence/internal/types"
	"github.com/user/terminal-intelligence/internal/ui"
)

var version = "0.0.2.3"

// Build number (injected at compile time via -ldflags)
var buildNumber = "dev"

func main() {
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
			fmt.Fprintf(os.Stderr, "Continuing with defaults...\n")
		} else {
			fmt.Fprintf(os.Stderr, "Created default config file at: %s\n", createdPath)
			fmt.Fprintf(os.Stderr, "Edit this file to customize your settings.\n")
			fmt.Fprintf(os.Stderr, "Continuing with defaults for this session...\n\n")
		}
	}

	// Default workspace to current working directory if not set
	if appCfg.WorkspaceDir == "" {
		cwd, err := os.Getwd()
		if err == nil {
			appCfg.WorkspaceDir = cwd
		}
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
