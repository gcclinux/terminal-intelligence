package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/user/terminal-intelligence/internal/config"
	"github.com/user/terminal-intelligence/internal/types"
	"github.com/user/terminal-intelligence/internal/ui"
)

var version = "0.0.2.6"
var buildNumber = "dev"

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "-v" || os.Args[1] == "--version") {
		fmt.Printf("Terminal Intelligence v%s (build %s)\n", version, buildNumber)
		return
	}

	// 1. Initialize Default Config
	appCfg := types.DefaultConfig()

	// 2. Load Configuration Logic
	configPath, err := config.ConfigFilePath()
	if err != nil {
		// Fatal error: cannot even determine where config should be
		fmt.Fprintf(os.Stderr, "Error determining config path: %v\n", err)
		os.Exit(1)
	}

	if _, err := os.Stat(configPath); err == nil {
		// File exists: Load and validate
		jcfg, err := config.LoadFromFile(configPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}
		if err := config.Validate(jcfg); err != nil {
			fmt.Fprintf(os.Stderr, "Invalid config: %v\n", err)
			os.Exit(1)
		}
		config.ApplyToAppConfig(jcfg, appCfg)
	} else if os.IsNotExist(err) {
		// File missing: Create it once and move on
		_, _ = config.CreateDefaultConfig()
		// Note: We don't exit here; we just continue with appCfg defaults
	}

	// 3. Runtime Defaults (Do NOT save these back to the JSON file)
	// This ensures the app respects the current folder it's opened in
	// unless the user has explicitly hardcoded a path in their config.
	workspaceChanged := false
	if appCfg.WorkspaceDir == "" {
		if cwd, err := os.Getwd(); err == nil {
			appCfg.WorkspaceDir = cwd
			workspaceChanged = true
		} else {
			appCfg.WorkspaceDir = "." // Fallback
			workspaceChanged = true
		}
	}

	// 4. Ensure Workspace exists
	if err := os.MkdirAll(appCfg.WorkspaceDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating workspace: %v\n", err)
		os.Exit(1)
	}

	// 5. Save workspace to config if it was set from current directory
	if workspaceChanged {
		_ = config.UpdateWorkspace(appCfg.WorkspaceDir)
	}

	// 6. Run Application
	app := ui.New(appCfg, buildNumber)
	p := tea.NewProgram(app, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Application error: %v\n", err)
		os.Exit(1)
	}
}
