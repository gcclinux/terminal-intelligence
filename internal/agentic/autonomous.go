package agentic

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/user/terminal-intelligence/internal/ai"
)

var projectNameRe = regexp.MustCompile(`(?im)^[#*\-\s]*project\s*name\s*[:\-]?\s*([a-z0-9\-]+)`)

// CreatorState represents the current state of the Autonomous Creator state machine.
type CreatorState int

const (
	StatePlanning CreatorState = iota
	StateWaitingApproval
	StateSetup
	StateDependencies
	StateFileCreation
	StateTesting
	StateDocumentation
	StateBuildAndRun
	StateDone
)

// AutonomousCreator orchestrates the /create workflow.
type AutonomousCreator struct {
	AIClient    ai.AIClient
	Model       string
	Workspace   string
	Description string

	State       CreatorState
	ProjectName string
	ProjectDir  string
	Plan        string
	FilesToMake map[string]string // map of relative path to content

	// Callbacks for UI interactions
	OpenFileCallback func(filePath string) error

	// Running process (for web servers)
	RunningProcess *exec.Cmd
	ServerURL      string // URL of the running server
}

// NewAutonomousCreator initializes a new creator flow.
func NewAutonomousCreator(client ai.AIClient, model, workspace, desc string) *AutonomousCreator {
	return &AutonomousCreator{
		AIClient:    client,
		Model:       model,
		Workspace:   workspace,
		Description: desc,
		State:       StatePlanning,
		FilesToMake: make(map[string]string),
	}
}

// Emulate a state machine step
func (c *AutonomousCreator) Step() (string, error) {
	switch c.State {
	case StatePlanning:
		return c.doPlanning()
	case StateWaitingApproval:
		return "Waiting for user approval... (Type /proceed or yes)", nil
	case StateSetup:
		return c.doSetup()
	case StateDependencies:
		return c.doDependencies()
	case StateFileCreation:
		return c.doFileCreation()
	case StateTesting:
		return c.doTesting()
	case StateDocumentation:
		return c.doDocumentation()
	case StateBuildAndRun:
		return c.doBuildAndRun()
	case StateDone:
		return "Application creation complete! You can now run your app.", nil
	default:
		return "", fmt.Errorf("unknown creator state")
	}
}

func (c *AutonomousCreator) doPlanning() (string, error) {
	prompt := fmt.Sprintf(`You are an expert autonomous software engineer.
The user wants to create a new application from scratch with the following description:
"%s"

Please provide an implementation plan. Include:
1. A suggested project name (very short, lowercase, hyphenated).
2. A high-level architecture overview.
3. The specific files and folder structure that will be created.
4. The commands needed to initialize dependencies (e.g. go mod init, pip install).
5. The command to run the application to test it.`, c.Description)

	plan, err := aicall(c.AIClient, c.Model, prompt)
	if err != nil {
		return "", err
	}

	c.Plan = plan
	// Extract project name
	c.ProjectName = extractProjectName(plan)
	if c.ProjectName == "" {
		c.ProjectName = "ti-autonomous-app"
	}
	c.ProjectDir = filepath.Join(c.Workspace, c.ProjectName)

	c.State = StateWaitingApproval
	return fmt.Sprintf("ai-assist %s\nPlan generated:\n\n%s\n\nDo you want to proceed? Type /proceed to continue or /cancel to abort.", getCurrentTime(), plan), nil
}

func (c *AutonomousCreator) doSetup() (string, error) {
	// Create project directory
	if err := os.MkdirAll(c.ProjectDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create project directory: %v", err)
	}

	c.State = StateDependencies
	return fmt.Sprintf("ai-assist %s\nCreated project folder: %s\n\nMoving to install dependencies...", getCurrentTime(), c.ProjectName), nil
}

func (c *AutonomousCreator) doDependencies() (string, error) {
	// Ask AI for the specific setup shell commands required.
	prompt := fmt.Sprintf(`Given the implementation plan:
%s

What are the precise terminal commands to initialize the project dependencies?
Return ONLY a bash script with the commands. No markdown formatting, no explanations. Just the raw commands.
Example for Go: go mod init my-app && go mod tidy
Example for Python: python3 -m venv venv && source venv/bin/activate && pip install fastapi uvicorn
Assume we are already inside the project directory.`, c.Plan)

	cmdsStr, err := aicall(c.AIClient, c.Model, prompt)
	if err != nil {
		return "", err
	}

	cmdsStr = strings.TrimSpace(strings.TrimPrefix(strings.TrimSuffix(cmdsStr, "```"), "```bash"))
	cmdsStr = strings.TrimSpace(strings.TrimPrefix(cmdsStr, "```sh"))
	cmdsStr = strings.TrimSpace(strings.TrimPrefix(cmdsStr, "```"))

	if cmdsStr != "" {
		// Prepare a shell script to execute
		scriptPath := filepath.Join(c.ProjectDir, "setup.sh")
		if runtime.GOOS == "windows" {
			scriptPath = filepath.Join(c.ProjectDir, "setup.bat")
		}

		err = os.WriteFile(scriptPath, []byte(cmdsStr), 0755)
		if err != nil {
			return "", fmt.Errorf("failed to write setup script: %v", err)
		}

		// Execute it
		var cmd *exec.Cmd
		if runtime.GOOS == "windows" {
			cmd = exec.Command("cmd", "/C", "setup.bat")
		} else {
			cmd = exec.Command("sh", "setup.sh")
		}
		cmd.Dir = c.ProjectDir
		out, err := cmd.CombinedOutput()

		// Optional cleanup
		os.Remove(scriptPath)

		if err != nil {
			return "", fmt.Errorf("dependency setup failed: %v\nOutput:\n%s", err, string(out))
		}
	}

	c.State = StateFileCreation
	return fmt.Sprintf("ai-assist %s\nDependencies installed successfully.\n\nMoving to code generation...", getCurrentTime()), nil
}

func (c *AutonomousCreator) doFileCreation() (string, error) {
	prompt := fmt.Sprintf(`Given the implementation plan:
%s

Generate all the necessary code files for this project.
Return the files inside standard Markdown code blocks with the relative filepath specified immediately before the code block.

Example:
**main.go**
`+"```go"+`
package main
// ...
`+"```"+`

**utils/helper.go**
`+"```go"+`
package utils
// ...
`+"```"+`

Only return the file paths and code blocks. No other text.`, c.Plan)

	response, err := aicall(c.AIClient, c.Model, prompt)
	if err != nil {
		return "", err
	}

	// Simple parser for "**path/to/file.ext**\n```lang\ncontent\n```"
	lines := strings.Split(response, "\n")
	var currentFile string
	var currentContent strings.Builder
	inBlock := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check for file name
		if !inBlock && strings.HasPrefix(trimmed, "**") && strings.HasSuffix(trimmed, "**") {
			currentFile = strings.Trim(trimmed, "*")
			continue
		}

		if strings.HasPrefix(trimmed, "```") {
			if inBlock {
				// End of block
				if currentFile != "" {
					c.FilesToMake[currentFile] = currentContent.String()
				}
				currentFile = ""
				currentContent.Reset()
				inBlock = false
			} else {
				// Start of block
				inBlock = true
			}
			continue
		}

		if inBlock {
			currentContent.WriteString(line + "\n")
		}
	}

	// Write files to disk
	createdFiles := []string{}
	for relPath, content := range c.FilesToMake {
		absPath := filepath.Join(c.ProjectDir, relPath)
		// Ensure parent dirs exist
		os.MkdirAll(filepath.Dir(absPath), 0755)

		if err := os.WriteFile(absPath, []byte(content), 0644); err != nil {
			return "", fmt.Errorf("failed to write %s: %v", relPath, err)
		}
		createdFiles = append(createdFiles, relPath)
	}

	c.State = StateTesting
	return fmt.Sprintf("ai-assist %s\nGenerated and saved %d files:\n- %s\n\nMoving to testing...", getCurrentTime(), len(createdFiles), strings.Join(createdFiles, "\n- ")), nil
}

func (c *AutonomousCreator) doTesting() (string, error) {
	// Detect project type from created files
	projectType := c.detectProjectType()

	// Build context-aware prompt
	prompt := fmt.Sprintf(`Given the implementation plan:
%s

The following files were created:
%s

Project type detected: %s

Please provide a single terminal command to run the main tests or build the project.
For Go projects, use: go build or go test ./...
For Python projects, use: python -m pytest or python main.py
For shell/bash projects, use: bash main.sh or shellcheck *.sh
For PowerShell projects, use: powershell -File main.ps1

IMPORTANT: Only return commands for supported project types (Go, Python, Bash, PowerShell).
Do NOT return npm, yarn, or node commands as they are not yet supported.
Return ONLY this single bash command, no formatting, no markdown.`,
		c.Plan,
		strings.Join(getFileList(c.FilesToMake), ", "),
		projectType)

	cmdStr, err := aicall(c.AIClient, c.Model, prompt)
	if err != nil {
		return "", err
	}
	cmdStr = strings.TrimSpace(strings.TrimPrefix(strings.TrimSuffix(cmdStr, "```"), "```bash"))
	cmdStr = strings.TrimSpace(strings.TrimPrefix(cmdStr, "```sh"))
	cmdStr = strings.TrimSpace(strings.TrimPrefix(cmdStr, "```"))

	if cmdStr != "" {
		// Run test/build
		scriptPath := filepath.Join(c.ProjectDir, "test.sh")
		if runtime.GOOS == "windows" {
			scriptPath = filepath.Join(c.ProjectDir, "test.bat")
		}

		err = os.WriteFile(scriptPath, []byte(cmdStr), 0755)
		if err != nil {
			return "", fmt.Errorf("failed to write test script: %v", err)
		}

		var cmd *exec.Cmd
		if runtime.GOOS == "windows" {
			cmd = exec.Command("cmd", "/C", "test.bat")
		} else {
			cmd = exec.Command("sh", "test.sh")
		}
		cmd.Dir = c.ProjectDir
		out, err := cmd.CombinedOutput()
		os.Remove(scriptPath)

		if err != nil {
			// On error we let the AI try to fix it, or simply return the error and halt.
			// Implementing self-healing here goes beyond simple flow, but we can do a single pass:
			return "", fmt.Errorf("Automated test failed: %v\nOutput: %s\n\nAborting autonomous creation.", err, string(out))
		}
	}

	c.State = StateDocumentation
	return fmt.Sprintf("ai-assist %s\nAutomated tests passed.\n\nMoving to documentation...", getCurrentTime()), nil
}

func (c *AutonomousCreator) doDocumentation() (string, error) {
	prompt := fmt.Sprintf(`Given the implementation plan:
%s

Generate a SUMMARY.md file that explains the architecture, how to build/run the project, and how it was constructed.
Return ONLY the raw markdown content.`, c.Plan)

	summary, err := aicall(c.AIClient, c.Model, prompt)
	if err != nil {
		return "", err
	}

	summary = strings.TrimSpace(strings.TrimPrefix(strings.TrimSuffix(summary, "```"), "```markdown"))
	summary = strings.TrimSpace(strings.TrimPrefix(summary, "```"))

	summaryPath := filepath.Join(c.ProjectDir, "SUMMARY.md")
	if err := os.WriteFile(summaryPath, []byte(summary), 0644); err != nil {
		return "", err
	}

	// Open SUMMARY.md in the editor
	if c.OpenFileCallback != nil {
		if err := c.OpenFileCallback(summaryPath); err != nil {
			// Log error but don't fail - this is a nice-to-have feature
			fmt.Printf("Warning: Could not open SUMMARY.md in editor: %v\n", err)
		}
	}

	c.State = StateBuildAndRun
	return fmt.Sprintf("ai-assist %s\nSuccessfully generated SUMMARY.md (opened in editor)\n\nMoving to build and run...", getCurrentTime()), nil
}

func (c *AutonomousCreator) doBuildAndRun() (string, error) {
	projectType := c.detectProjectType()

	var result strings.Builder
	result.WriteString(fmt.Sprintf("ai-assist %s\n", getCurrentTime()))

	switch projectType {
	case "Go":
		return c.buildAndRunGo()
	case "Python":
		return c.buildAndRunPython()
	case "Bash/Shell":
		return c.buildAndRunBash()
	case "PowerShell":
		return c.buildAndRunPowerShell()
	default:
		c.State = StateDone
		return fmt.Sprintf("ai-assist %s\nProject type '%s' - skipping build and run.\n\nApp Creation complete! Navigate to %s to run your application manually.",
			getCurrentTime(), projectType, c.ProjectName), nil
	}
}

func (c *AutonomousCreator) buildAndRunGo() (string, error) {
	var result strings.Builder
	result.WriteString(fmt.Sprintf("ai-assist %s\nBuilding Go application...\n", getCurrentTime()))

	// Build the application
	binaryName := c.ProjectName
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}

	buildCmd := exec.Command("go", "build", "-o", binaryName)
	buildCmd.Dir = c.ProjectDir
	buildOutput, err := buildCmd.CombinedOutput()

	if err != nil {
		return "", fmt.Errorf("build failed: %v\nOutput: %s", err, string(buildOutput))
	}

	result.WriteString(fmt.Sprintf("Build successful! Binary: %s\n\n", binaryName))

	// Detect if it's a web server by checking the plan and code
	isWebServer, port := c.detectWebServer()

	if isWebServer {
		result.WriteString(fmt.Sprintf("Web server detected (port %s)\n", port))
		result.WriteString(fmt.Sprintf("🌐 Application URL: http://localhost:%s\n\n", port))

		// Start the server in a new terminal window
		result.WriteString("Starting server in new terminal window...\n")

		// Use absolute path to the binary
		binaryPath := filepath.Join(c.ProjectDir, binaryName)

		var runCmd *exec.Cmd
		if runtime.GOOS == "windows" {
			// On Windows, use Start-Process via PowerShell to open a new window
			runCmd = exec.Command("powershell", "-Command",
				fmt.Sprintf("Start-Process -FilePath '%s' -WorkingDirectory '%s'", binaryPath, c.ProjectDir))
		} else {
			// On Linux/Mac, try different terminal emulators
			// Try gnome-terminal, xterm, or open (macOS)
			if _, err := exec.LookPath("gnome-terminal"); err == nil {
				runCmd = exec.Command("gnome-terminal", "--", binaryPath)
				runCmd.Dir = c.ProjectDir
			} else if _, err := exec.LookPath("xterm"); err == nil {
				runCmd = exec.Command("xterm", "-e", binaryPath)
				runCmd.Dir = c.ProjectDir
			} else if runtime.GOOS == "darwin" {
				// macOS: use 'open' with Terminal.app
				runCmd = exec.Command("open", "-a", "Terminal", binaryPath)
			} else {
				// Fallback: run in background without terminal
				runCmd = exec.Command(binaryPath)
				runCmd.Dir = c.ProjectDir
			}
		}

		// Start the process
		if err := runCmd.Run(); err != nil {
			result.WriteString(fmt.Sprintf("Warning: Could not open terminal window: %v\n", err))
			result.WriteString("Trying to start in background...\n")

			// Fallback: start in background
			bgCmd := exec.Command(binaryPath)
			bgCmd.Dir = c.ProjectDir
			if err := bgCmd.Start(); err != nil {
				result.WriteString(fmt.Sprintf("Error: Could not start server: %v\n", err))
				result.WriteString(fmt.Sprintf("\nTo start manually: cd %s && ./%s\n", c.ProjectName, binaryName))
			} else {
				c.RunningProcess = bgCmd
				result.WriteString("✓ Server started in background\n\n")
			}
		} else {
			result.WriteString("✓ Server is now running in a new terminal window!\n\n")
		}

		result.WriteString(fmt.Sprintf("🌐 Application URL: http://localhost:%s\n", port))
		result.WriteString("   Click the link above to open in your browser\n\n")
		result.WriteString("Note: Check the new terminal window for server logs.\n")
		result.WriteString("      Close the terminal window to stop the server.\n")
	} else {
		// Run the application and capture output
		result.WriteString("Running application...\n\n")
		result.WriteString("--- Application Output ---\n")

		// Use absolute path to the binary
		binaryPath := filepath.Join(c.ProjectDir, binaryName)
		runCmd := exec.Command(binaryPath)
		runCmd.Dir = c.ProjectDir
		output, err := runCmd.CombinedOutput()

		if err != nil {
			result.WriteString(fmt.Sprintf("Error: %v\n", err))
		}
		result.WriteString(string(output))
		result.WriteString("\n--- End Output ---\n\n")
		result.WriteString(fmt.Sprintf("To run again: cd %s && ./%s\n", c.ProjectName, binaryName))
	}

	c.State = StateDone
	result.WriteString("\nApp Creation complete!")
	return result.String(), nil
}

func (c *AutonomousCreator) buildAndRunPython() (string, error) {
	var result strings.Builder
	result.WriteString(fmt.Sprintf("ai-assist %s\n", getCurrentTime()))

	// Find the main Python file
	mainFile := c.findMainPythonFile()
	if mainFile == "" {
		c.State = StateDone
		return fmt.Sprintf("ai-assist %s\nCould not find main Python file.\n\nApp Creation complete! Navigate to %s to run your application manually.",
			getCurrentTime(), c.ProjectName), nil
	}

	// Detect if it's a web server
	isWebServer, port := c.detectWebServer()

	if isWebServer {
		result.WriteString(fmt.Sprintf("Python web server detected (port %s)\n", port))
		result.WriteString(fmt.Sprintf("🌐 Application URL: http://localhost:%s\n\n", port))

		// Start the server in a new terminal window
		result.WriteString("Starting server in new terminal window...\n")

		var runCmd *exec.Cmd
		if runtime.GOOS == "windows" {
			// On Windows, use Start-Process via PowerShell to open a new window
			runCmd = exec.Command("powershell", "-Command",
				fmt.Sprintf("Start-Process -FilePath 'python' -ArgumentList '%s' -WorkingDirectory '%s'",
					mainFile, c.ProjectDir))
		} else {
			// On Linux/Mac, try different terminal emulators
			if _, err := exec.LookPath("gnome-terminal"); err == nil {
				runCmd = exec.Command("gnome-terminal", "--", "python", mainFile)
				runCmd.Dir = c.ProjectDir
			} else if _, err := exec.LookPath("xterm"); err == nil {
				runCmd = exec.Command("xterm", "-e", "python", mainFile)
				runCmd.Dir = c.ProjectDir
			} else if runtime.GOOS == "darwin" {
				// macOS: use 'open' with Terminal.app
				script := fmt.Sprintf("cd '%s' && python %s", c.ProjectDir, mainFile)
				runCmd = exec.Command("osascript", "-e",
					fmt.Sprintf("tell application \"Terminal\" to do script \"%s\"", script))
			} else {
				// Fallback: run in background without terminal
				runCmd = exec.Command("python", mainFile)
				runCmd.Dir = c.ProjectDir
			}
		}

		// Start the process
		if err := runCmd.Run(); err != nil {
			result.WriteString(fmt.Sprintf("Warning: Could not open terminal window: %v\n", err))
			result.WriteString("Trying to start in background...\n")

			// Fallback: start in background
			bgCmd := exec.Command("python", mainFile)
			bgCmd.Dir = c.ProjectDir
			if err := bgCmd.Start(); err != nil {
				result.WriteString(fmt.Sprintf("Error: Could not start server: %v\n", err))
				result.WriteString(fmt.Sprintf("\nTo start manually: cd %s && python %s\n", c.ProjectName, mainFile))
			} else {
				c.RunningProcess = bgCmd
				result.WriteString("✓ Server started in background\n\n")
			}
		} else {
			result.WriteString("✓ Server is now running in a new terminal window!\n\n")
		}

		result.WriteString(fmt.Sprintf("🌐 Application URL: http://localhost:%s\n", port))
		result.WriteString("   Click the link above to open in your browser\n\n")
		result.WriteString("Note: Check the new terminal window for server logs.\n")
		result.WriteString("      Close the terminal window to stop the server.\n")
	} else {
		result.WriteString("Running Python application...\n\n")
		result.WriteString("--- Application Output ---\n")

		runCmd := exec.Command("python", mainFile)
		runCmd.Dir = c.ProjectDir
		output, err := runCmd.CombinedOutput()

		if err != nil {
			result.WriteString(fmt.Sprintf("Error: %v\n", err))
		}
		result.WriteString(string(output))
		result.WriteString("\n--- End Output ---\n\n")
		result.WriteString(fmt.Sprintf("To run again: cd %s && python %s\n", c.ProjectName, mainFile))
	}

	c.State = StateDone
	result.WriteString("\nApp Creation complete!")
	return result.String(), nil
}

func (c *AutonomousCreator) buildAndRunBash() (string, error) {
	var result strings.Builder
	result.WriteString(fmt.Sprintf("ai-assist %s\n", getCurrentTime()))

	// Find the main shell script
	mainFile := c.findMainShellFile()
	if mainFile == "" {
		c.State = StateDone
		return fmt.Sprintf("ai-assist %s\nCould not find main shell script.\n\nApp Creation complete! Navigate to %s to run your application manually.",
			getCurrentTime(), c.ProjectName), nil
	}

	result.WriteString("Running shell script...\n\n")
	result.WriteString("--- Application Output ---\n")

	runCmd := exec.Command("bash", mainFile)
	runCmd.Dir = c.ProjectDir
	output, err := runCmd.CombinedOutput()

	if err != nil {
		result.WriteString(fmt.Sprintf("Error: %v\n", err))
	}
	result.WriteString(string(output))
	result.WriteString("\n--- End Output ---\n\n")
	result.WriteString(fmt.Sprintf("To run again: cd %s && bash %s\n", c.ProjectName, mainFile))

	c.State = StateDone
	result.WriteString("\nApp Creation complete!")
	return result.String(), nil
}

func (c *AutonomousCreator) buildAndRunPowerShell() (string, error) {
	var result strings.Builder
	result.WriteString(fmt.Sprintf("ai-assist %s\n", getCurrentTime()))

	// Find the main PowerShell script
	mainFile := c.findMainPowerShellFile()
	if mainFile == "" {
		c.State = StateDone
		return fmt.Sprintf("ai-assist %s\nCould not find main PowerShell script.\n\nApp Creation complete! Navigate to %s to run your application manually.",
			getCurrentTime(), c.ProjectName), nil
	}

	result.WriteString("Running PowerShell script...\n\n")
	result.WriteString("--- Application Output ---\n")

	runCmd := exec.Command("powershell", "-File", mainFile)
	runCmd.Dir = c.ProjectDir
	output, err := runCmd.CombinedOutput()

	if err != nil {
		result.WriteString(fmt.Sprintf("Error: %v\n", err))
	}
	result.WriteString(string(output))
	result.WriteString("\n--- End Output ---\n\n")
	result.WriteString(fmt.Sprintf("To run again: cd %s && powershell -File %s\n", c.ProjectName, mainFile))

	c.State = StateDone
	result.WriteString("\nApp Creation complete!")
	return result.String(), nil
}

func aicall(client ai.AIClient, model, prompt string) (string, error) {
	ch, err := client.Generate(prompt, model, nil)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	for chunk := range ch {
		sb.WriteString(chunk)
	}
	return sb.String(), nil
}

func extractProjectName(plan string) string {
	matches := projectNameRe.FindStringSubmatch(plan)
	if len(matches) >= 2 && matches[1] != "" {
		return strings.TrimSpace(matches[1])
	}
	return "autonomous-app"
}

func getCurrentTime() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

// detectProjectType analyzes created files to determine project type
func (c *AutonomousCreator) detectProjectType() string {
	for filename := range c.FilesToMake {
		switch {
		case filename == "go.mod" || strings.HasSuffix(filename, ".go"):
			return "Go"
		case filename == "requirements.txt" || filename == "setup.py" || strings.HasSuffix(filename, ".py"):
			return "Python"
		case strings.HasSuffix(filename, ".sh"):
			return "Bash/Shell"
		case strings.HasSuffix(filename, ".ps1"):
			return "PowerShell"
		case filename == "package.json":
			return "Node.js (NOT SUPPORTED - use Go, Python, Bash, or PowerShell instead)"
		}
	}
	return "Unknown"
}

// getFileList returns a list of filenames from the FilesToMake map
func getFileList(files map[string]string) []string {
	list := make([]string, 0, len(files))
	for filename := range files {
		list = append(list, filename)
	}
	return list
}

// detectWebServer checks if the application is a web server and extracts the port
func (c *AutonomousCreator) detectWebServer() (bool, string) {
	// Check plan and code for web server indicators
	planLower := strings.ToLower(c.Plan)

	// Common web server indicators
	webIndicators := []string{"web server", "http server", "web application", "api server", "rest api", "localhost"}
	isWebServer := false
	for _, indicator := range webIndicators {
		if strings.Contains(planLower, indicator) {
			isWebServer = true
			break
		}
	}

	// Check code content for HTTP server patterns
	if !isWebServer {
		for _, content := range c.FilesToMake {
			contentLower := strings.ToLower(content)
			if strings.Contains(contentLower, "http.listenandserve") ||
				strings.Contains(contentLower, "http.server") ||
				strings.Contains(contentLower, "flask") ||
				strings.Contains(contentLower, "fastapi") ||
				strings.Contains(contentLower, "express") {
				isWebServer = true
				break
			}
		}
	}

	if !isWebServer {
		return false, ""
	}

	// Extract port number
	port := "8080" // default

	// Try to find port in plan
	portPatterns := []string{
		`port\s+(\d+)`,
		`:\s*(\d{4,5})`,
		`localhost:(\d+)`,
	}

	for _, pattern := range portPatterns {
		re := regexp.MustCompile(pattern)
		if matches := re.FindStringSubmatch(c.Plan); len(matches) > 1 {
			port = matches[1]
			break
		}
	}

	// Also check code content
	for _, content := range c.FilesToMake {
		for _, pattern := range portPatterns {
			re := regexp.MustCompile(pattern)
			if matches := re.FindStringSubmatch(content); len(matches) > 1 {
				port = matches[1]
				break
			}
		}
	}

	return true, port
}

// findMainPythonFile finds the main Python file to run
func (c *AutonomousCreator) findMainPythonFile() string {
	// Priority order: main.py, app.py, server.py, any .py file
	priorities := []string{"main.py", "app.py", "server.py", "run.py"}

	for _, priority := range priorities {
		if _, exists := c.FilesToMake[priority]; exists {
			return priority
		}
	}

	// Return first .py file found
	for filename := range c.FilesToMake {
		if strings.HasSuffix(filename, ".py") {
			return filename
		}
	}

	return ""
}

// findMainShellFile finds the main shell script to run
func (c *AutonomousCreator) findMainShellFile() string {
	// Priority order: main.sh, run.sh, start.sh, any .sh file
	priorities := []string{"main.sh", "run.sh", "start.sh", "script.sh"}

	for _, priority := range priorities {
		if _, exists := c.FilesToMake[priority]; exists {
			return priority
		}
	}

	// Return first .sh file found
	for filename := range c.FilesToMake {
		if strings.HasSuffix(filename, ".sh") {
			return filename
		}
	}

	return ""
}

// findMainPowerShellFile finds the main PowerShell script to run
func (c *AutonomousCreator) findMainPowerShellFile() string {
	// Priority order: main.ps1, run.ps1, start.ps1, any .ps1 file
	priorities := []string{"main.ps1", "run.ps1", "start.ps1", "script.ps1"}

	for _, priority := range priorities {
		if _, exists := c.FilesToMake[priority]; exists {
			return priority
		}
	}

	// Return first .ps1 file found
	for filename := range c.FilesToMake {
		if strings.HasSuffix(filename, ".ps1") {
			return filename
		}
	}

	return ""
}
