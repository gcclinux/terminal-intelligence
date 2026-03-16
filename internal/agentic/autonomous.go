package agentic

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/user/terminal-intelligence/internal/ai"
)

var projectNameRe = regexp.MustCompile(`(?im)project\s*name\s*[:\-]?\s*` + `\**` + "`?" + `([a-zA-Z0-9\-_]+)` + "`?" + `\**`)

// CreatorState represents the current state of the Autonomous Creator state machine.
type CreatorState int

const (
	StatePlanning CreatorState = iota
	StateWaitingApproval
	StateSetup
	StateFileCreation
	StateDependencies
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

	// Fallback fixer for unresolvable test/build errors (optional, nil = skip fallback)
	fixer *AgenticProjectFixer
	// Logger for fallback progress messages (optional, nil = skip logging)
	logger *ActionLogger
}

// NewAutonomousCreator initializes a new creator flow.
func NewAutonomousCreator(client ai.AIClient, model, workspace, desc string, fixer *AgenticProjectFixer, logger *ActionLogger) *AutonomousCreator {
	return &AutonomousCreator{
		AIClient:    client,
		Model:       model,
		Workspace:   workspace,
		Description: desc,
		State:       StatePlanning,
		FilesToMake: make(map[string]string),
		fixer:       fixer,
		logger:      logger,
	}
}

// extractFileFromError extracts the first Go compiler file reference from error
// output matching the pattern <file>.go:<line>:<col>: and returns the absolute
// path by joining projectDir with the filename. Returns empty string if no match.
func extractFileFromError(errorOutput string, projectDir string) string {
	re := regexp.MustCompile(`(\S+\.go):\d+:\d+:`)
	m := re.FindStringSubmatch(errorOutput)
	if len(m) < 2 {
		return ""
	}
	return filepath.Join(projectDir, m[1])
}

// buildFallbackRequest constructs a FixSessionRequest for the fallback fix cycle.
// It formats the error context into a message and extracts the first file reference
// from the error output (if any) to set OpenFilePath.
func buildFallbackRequest(errorOutput, errorType, failedCmd, projectType, projectDir string) *FixSessionRequest {
	msg := fmt.Sprintf("The following %s error occurred while running %q in a %s project:\n\n%s\n\nPlease analyze the error and fix the code so that the command succeeds.",
		errorType, failedCmd, projectType, errorOutput)
	return &FixSessionRequest{
		Message:      msg,
		ProjectRoot:  projectDir,
		MaxAttempts:  5,
		MaxCycles:    2,
		OpenFilePath: extractFileFromError(errorOutput, projectDir),
	}
}

// fallbackFix delegates an unresolvable error to the AgenticProjectFixer.
// It returns (nil, nil) when c.fixer is nil so the caller can fall through
// to the existing abort behaviour.
func (c *AutonomousCreator) fallbackFix(errorOutput, errorType, failedCmd string) (*FixSessionResult, error) {
	if c.fixer == nil {
		return nil, nil
	}

	projectType := c.detectProjectType()

	if c.logger != nil {
		c.logger.Log("Starting fallback fix cycle for %s error", errorType)
	}

	request := buildFallbackRequest(errorOutput, errorType, failedCmd, projectType, c.ProjectDir)

	statusCallback := func(status string) {
		if c.logger != nil {
			c.logger.Log("create-fallback: %s", status)
		}
	}

	result, err := c.fixer.ProcessFixCommand(request, statusCallback)
	if err != nil {
		if c.logger != nil {
			c.logger.Log("Fallback fix cycle failed with error: %v", err)
		}
		return nil, err
	}

	if result != nil && result.Success {
		if c.logger != nil {
			c.logger.Log("Fallback fix cycle succeeded after %d attempts across %d cycles", result.TotalAttempts, result.TotalCycles)
		}
	} else if result != nil {
		if c.logger != nil {
			c.logger.Log("Fallback fix cycle failed: %s", result.ErrorMessage)
		}
	}

	return result, nil
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
	case StateFileCreation:
		return c.doFileCreation()
	case StateDependencies:
		return c.doDependencies()
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
1. A project name. If the user specified a name, use that exact name. Otherwise suggest a short, lowercase, hyphenated name.
2. A high-level architecture overview.
3. The specific files and folder structure that will be created.
4. The commands needed to initialize dependencies (e.g. go mod init, pip install).
5. The command to run the application to test it.
IMPORTANT: Use the programming language the user requested. If no language is specified, choose the most appropriate one.`, c.Description)

	plan, err := aicall(c.AIClient, c.Model, prompt)
	if err != nil {
		return "", err
	}

	c.Plan = plan
	// Normalize port 5000 → 8080 in the plan (port 5000 is blocked on Windows/macOS)
	c.Plan = strings.ReplaceAll(c.Plan, ":5000", ":8080")
	c.Plan = strings.ReplaceAll(c.Plan, "port 5000", "port 8080")
	c.Plan = strings.ReplaceAll(c.Plan, "port=5000", "port=8080")
	// Extract project name
	c.ProjectName = extractProjectName(plan)
	if c.ProjectName == "" {
		c.ProjectName = "ti-autonomous-app"
	}
	c.ProjectDir = filepath.Join(c.Workspace, c.ProjectName)
	c.ProjectDir, _ = filepath.Abs(c.ProjectDir)

	c.State = StateWaitingApproval
	return fmt.Sprintf("ai-assist %s\nPlan generated:\n\n%s\n\nDo you want to proceed? Type /proceed to continue or /cancel to abort.", getCurrentTime(), plan), nil
}

func (c *AutonomousCreator) doSetup() (string, error) {
	// Check if project directory already exists and find an available name
	originalName := c.ProjectName
	counter := 1

	for {
		if _, err := os.Stat(c.ProjectDir); os.IsNotExist(err) {
			// Directory doesn't exist, we can use it
			break
		}

		// Directory exists, try with a number suffix
		c.ProjectName = fmt.Sprintf("%s-%d", originalName, counter)
		c.ProjectDir = filepath.Join(c.Workspace, c.ProjectName)
		c.ProjectDir, _ = filepath.Abs(c.ProjectDir)
		counter++

		// Safety check to avoid infinite loop
		if counter > 100 {
			return "", fmt.Errorf("too many existing project directories with name %s", originalName)
		}
	}

	// Create project directory
	if err := os.MkdirAll(c.ProjectDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create project directory: %v", err)
	}

	var message string
	if c.ProjectName != originalName {
		message = fmt.Sprintf("ai-assist %s\nNote: Directory '%s' already exists.\nCreated project folder: %s\n\nMoving to code generation...",
			getCurrentTime(), originalName, c.ProjectName)
	} else {
		message = fmt.Sprintf("ai-assist %s\nCreated project folder: %s\n\nMoving to code generation...",
			getCurrentTime(), c.ProjectName)
	}

	c.State = StateFileCreation
	return message, nil
}

func (c *AutonomousCreator) doDependencies() (string, error) {
	// Detect Python binary if this is a Python project
	pythonBinary := detectPythonBinary()

	// Build platform-appropriate examples
	var pythonExample string
	if runtime.GOOS == "windows" {
		if pythonBinary != "" {
			pythonExample = fmt.Sprintf("Example for Python: %s -m venv venv && venv\\Scripts\\activate && pip install fastapi uvicorn", pythonBinary)
		} else {
			pythonExample = "Example for Python: python -m venv venv && venv\\Scripts\\activate && pip install fastapi uvicorn"
		}
	} else {
		if pythonBinary != "" {
			pythonExample = fmt.Sprintf("Example for Python: %s -m venv venv && source venv/bin/activate && pip install fastapi uvicorn", pythonBinary)
		} else {
			pythonExample = "Example for Python: python3 -m venv venv && source venv/bin/activate && pip install fastapi uvicorn"
		}
	}

	// Ask AI for the specific setup shell commands required.
	prompt := fmt.Sprintf(`Given the implementation plan:
%s

What are the precise terminal commands to initialize the project dependencies?
Return ONLY a script with the commands. No markdown formatting, no explanations. Just the raw commands.
Example for Go: go mod tidy
Do NOT include "go mod init" if a go.mod file already exists in the project.
%s
Assume we are already inside the project directory.`, c.Plan, pythonExample)

	cmdsStr, err := aicall(c.AIClient, c.Model, prompt)
	if err != nil {
		return "", err
	}

	cmdsStr = strings.TrimSpace(strings.TrimPrefix(strings.TrimSuffix(cmdsStr, "```"), "```bash"))
	cmdsStr = strings.TrimSpace(strings.TrimPrefix(cmdsStr, "```sh"))
	cmdsStr = strings.TrimSpace(strings.TrimPrefix(cmdsStr, "```"))

	// If go.mod already exists (created during file generation), strip "go mod init"
	// commands to avoid "go.mod already exists" errors.
	goModPath := filepath.Join(c.ProjectDir, "go.mod")
	if _, statErr := os.Stat(goModPath); statErr == nil {
		cmdsStr = stripGoModInit(cmdsStr)
	}

	if cmdsStr != "" {
		// On Windows, convert Unix-style commands to Windows-compatible ones
		if runtime.GOOS == "windows" {
			cmdsStr = convertToWindowsCommands(cmdsStr, pythonBinary)
		}

		// Prepare a shell script to execute
		scriptPath := filepath.Join(c.ProjectDir, "setup.sh")
		if runtime.GOOS == "windows" {
			scriptPath = filepath.Join(c.ProjectDir, "setup.bat")
		}

		scriptContent := cmdsStr
		if runtime.GOOS != "windows" {
			if !strings.HasPrefix(cmdsStr, "#!") {
				scriptContent = "#!/bin/bash\n" + cmdsStr
			}
			// Inject pip bootstrap so the script installs pip when missing
			scriptContent = injectPipBootstrap(scriptContent)
		}
		err = os.WriteFile(scriptPath, []byte(scriptContent), 0755)
		if err != nil {
			return "", fmt.Errorf("failed to write setup script: %v", err)
		}

		// Execute it
		var execLog string
		var out []byte
		if runtime.GOOS == "windows" {
			cmd := exec.Command("cmd", "/C", "setup.bat")
			cmd.Dir = c.ProjectDir
			out, err = cmd.CombinedOutput()
		} else {
			out, err, execLog = runScriptWithFallback(scriptPath, c.ProjectDir)
		}

		// Optional cleanup
		os.Remove(scriptPath)

		if err != nil {
			errorOutput := string(out)
			failedCmd := cmdsStr
			if c.logger != nil {
				c.logger.Log("Dependency setup failed: %v", err)
				c.logger.Log("Output: %s", errorOutput)
				c.logger.Log("Attempting fallback fix for dependency error...")
			}

			// Try fallback fix before aborting
			result2, fixErr := c.fallbackFix(errorOutput, "dependency", failedCmd)
			if fixErr != nil {
				return "", fmt.Errorf("%sdependency setup failed after fallback fix: %v\nOriginal output:\n%s", execLog, fixErr, errorOutput)
			}
			if result2 != nil && result2.Success {
				if c.logger != nil {
					c.logger.Log("Fallback fix resolved dependency issue after %d attempts", result2.TotalAttempts)
				}
				// Retry the dependency setup after fix
				if c.logger != nil {
					c.logger.Log("Retrying dependency setup after successful fix...")
				}
				retryScriptPath := filepath.Join(c.ProjectDir, "setup_retry.sh")
				if runtime.GOOS == "windows" {
					retryScriptPath = filepath.Join(c.ProjectDir, "setup_retry.bat")
				}
				retryContent := cmdsStr
				if runtime.GOOS != "windows" && !strings.HasPrefix(cmdsStr, "#!") {
					retryContent = "#!/bin/bash\n" + cmdsStr
				}
				if writeErr := os.WriteFile(retryScriptPath, []byte(retryContent), 0755); writeErr == nil {
					var retryOut []byte
					var retryErr error
					if runtime.GOOS == "windows" {
						retryCmd := exec.Command("cmd", "/C", "setup_retry.bat")
						retryCmd.Dir = c.ProjectDir
						retryOut, retryErr = retryCmd.CombinedOutput()
					} else {
						retryOut, retryErr, _ = runScriptWithFallback(retryScriptPath, c.ProjectDir)
					}
					os.Remove(retryScriptPath)
					if retryErr != nil {
						return "", fmt.Errorf("%sdependency setup failed on retry: %v\nOutput:\n%s", execLog, retryErr, string(retryOut))
					}
					if c.logger != nil {
						c.logger.Log("Dependency setup succeeded on retry after fix")
					}
				} else {
					return "", fmt.Errorf("%sdependency setup failed: could not write retry script: %v", execLog, writeErr)
				}
			} else {
				errMsg := "fix was unsuccessful"
				if result2 != nil {
					errMsg = result2.ErrorMessage
				}
				return "", fmt.Errorf("%sdependency setup failed: %v (%s)\nOutput:\n%s", execLog, err, errMsg, errorOutput)
			}
		}
	}

	c.State = StateTesting
	return fmt.Sprintf("ai-assist %s\nDependencies installed successfully.\n\nMoving to testing...", getCurrentTime()), nil
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
		// Port 5000 is blocked on Windows (firewall) and macOS Monterey+ (AirPlay).
		// Rewrite it to 8080 in all generated files.
		content = strings.ReplaceAll(content, "port=5000", "port=8080")
		content = strings.ReplaceAll(content, "port = 5000", "port = 8080")
		content = strings.ReplaceAll(content, ":5000", ":8080")
		c.FilesToMake[relPath] = content

		absPath := filepath.Join(c.ProjectDir, relPath)
		// Ensure parent dirs exist
		os.MkdirAll(filepath.Dir(absPath), 0755)

		if err := os.WriteFile(absPath, []byte(content), 0644); err != nil {
			return "", fmt.Errorf("failed to write %s: %v", relPath, err)
		}
		createdFiles = append(createdFiles, relPath)
	}

	// Post-creation dependency resolution for Go projects
	projectType := c.detectProjectType()
	if projectType == "Go" {
		if err := c.runGoModTidy(); err != nil {
			return "", fmt.Errorf("failed to run go mod tidy: %v", err)
		}
	}

	c.State = StateDependencies
	return fmt.Sprintf("ai-assist %s\nGenerated and saved %d files:\n- %s\n\nMoving to install dependencies...", getCurrentTime(), len(createdFiles), strings.Join(createdFiles, "\n- ")), nil
}


func (c *AutonomousCreator) doTesting() (string, error) {
	codeCtx := c.buildCodeContext()

	// Ask the AI to analyze the actual code and tell us how to verify it
	prompt := fmt.Sprintf(`You are an expert software engineer. A project was just generated with this plan:
%s

Here are the actual files and their contents:
%s

Analyze the code and answer these questions in EXACTLY this format (one answer per line, no extra text):
BUILD_CMD: <single shell command to compile/build the project, or NONE if not needed>
TEST_CMD: <single shell command to run tests or verify the build, or NONE if no tests>
IS_SERVER: <YES or NO - does this application start a long-running HTTP server?>
RUN_CMD: <single shell command to start the application, or NONE>
PORT: <port number the server listens on, or NONE>

Rules:
- Base your answers on the ACTUAL CODE, not assumptions.
- For Go projects the build command is typically "go build -o <name>" and run is "./<name>".
- For Python projects use the appropriate python/python3 command.
- Do NOT wrap commands in markdown. Return raw commands only.
- Assume we are already inside the project directory.`, c.Plan, codeCtx)

	analysis, err := aicall(c.AIClient, c.Model, prompt)
	if err != nil {
		return "", err
	}

	buildCmd := extractAIField(analysis, "BUILD_CMD")
	testCmd := extractAIField(analysis, "TEST_CMD")
	isServer := strings.EqualFold(extractAIField(analysis, "IS_SERVER"), "YES")
	runCmd := extractAIField(analysis, "RUN_CMD")
	port := extractAIField(analysis, "PORT")
	if port == "" || strings.EqualFold(port, "NONE") {
		port = "8080"
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("ai-assist %s\n", getCurrentTime()))

	// Step 1: Build if needed
	if buildCmd != "" && !strings.EqualFold(buildCmd, "NONE") {
		result.WriteString(fmt.Sprintf("Building: %s\n", buildCmd))
		if c.logger != nil {
			c.logger.Log("Building: %s", buildCmd)
		}
		out, buildErr := c.runShellCmd(buildCmd)
		if buildErr != nil {
			fixResult, fixErr := c.aiDrivenFix(buildCmd, string(out), "build")
			if fixErr != nil {
				return result.String(), fmt.Errorf("build failed: %v\nOutput: %s", buildErr, string(out))
			}
			result.WriteString(fixResult)
		} else {
			result.WriteString("Build successful.\n")
		}
	}

	// Step 2: Run tests if available
	if testCmd != "" && !strings.EqualFold(testCmd, "NONE") {
		result.WriteString(fmt.Sprintf("Testing: %s\n", testCmd))
		if c.logger != nil {
			c.logger.Log("Testing: %s", testCmd)
		}
		out, testErr := c.runShellCmd(testCmd)
		if testErr != nil {
			fixResult, fixErr := c.aiDrivenFix(testCmd, string(out), "test")
			if fixErr != nil {
				result.WriteString(fmt.Sprintf("Tests failed: %v\nOutput: %s\n", testErr, string(out)))
			} else {
				result.WriteString(fixResult)
			}
		} else {
			result.WriteString("Tests passed.\n")
		}
	}

	// Step 3: Smoke-test the server if it's a web server
	if isServer && runCmd != "" && !strings.EqualFold(runCmd, "NONE") {
		smokeResult, smokeErr := c.smokeTestServer(runCmd, port)
		result.WriteString(smokeResult)
		if smokeErr != nil {
			fixResult, fixErr := c.aiDrivenFix(runCmd, smokeErr.Error(), "server startup")
			if fixErr != nil {
				result.WriteString(fmt.Sprintf("Server smoke test failed: %v\n", smokeErr))
			} else {
				result.WriteString(fixResult)
			}
		}
	}

	c.State = StateDocumentation
	result.WriteString(fmt.Sprintf("\nai-assist %s\nVerification complete. Moving to documentation...\n", getCurrentTime()))
	return result.String(), nil
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
	codeCtx := c.buildCodeContext()

	// Ask the AI how to build and run this specific project
	prompt := fmt.Sprintf(`You are an expert software engineer. A project was generated with this plan:
%s

Here are the actual files:
%s

Provide the commands to build and run this application in EXACTLY this format (one per line, no extra text):
BUILD_CMD: <shell command to build, or NONE>
RUN_CMD: <shell command to run the application>
IS_SERVER: <YES or NO - is this a long-running server?>
PORT: <port number if server, or NONE>
RUN_INSTRUCTIONS: <one-line human-readable instruction for the user to run it manually>

Rules:
- Base answers on the ACTUAL CODE.
- Do NOT wrap commands in markdown.
- Assume we are already inside the project directory.`, c.Plan, codeCtx)

	analysis, err := aicall(c.AIClient, c.Model, prompt)
	if err != nil {
		return "", err
	}

	buildCmd := extractAIField(analysis, "BUILD_CMD")
	runCmd := extractAIField(analysis, "RUN_CMD")
	isServer := strings.EqualFold(extractAIField(analysis, "IS_SERVER"), "YES")
	port := extractAIField(analysis, "PORT")
	runInstructions := extractAIField(analysis, "RUN_INSTRUCTIONS")
	if port == "" || strings.EqualFold(port, "NONE") {
		port = "8080"
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("ai-assist %s\n", getCurrentTime()))

	// Build if needed
	if buildCmd != "" && !strings.EqualFold(buildCmd, "NONE") {
		result.WriteString(fmt.Sprintf("Building: %s\n", buildCmd))
		out, buildErr := c.runShellCmd(buildCmd)
		if buildErr != nil {
			fixResult, fixErr := c.aiDrivenFix(buildCmd, string(out), "build")
			if fixErr != nil {
				result.WriteString(fmt.Sprintf("Build failed: %s\n", string(out)))
				result.WriteString(fmt.Sprintf("\nTo build manually: cd %s && %s\n", c.ProjectName, buildCmd))
				c.State = StateDone
				result.WriteString("\nApp Creation complete!")
				return result.String(), nil
			}
			result.WriteString(fixResult)
		} else {
			result.WriteString("Build successful.\n")
		}
	}

	if isServer && runCmd != "" && !strings.EqualFold(runCmd, "NONE") {
		result.WriteString(fmt.Sprintf("Web server detected (port %s)\n", port))
		result.WriteString(fmt.Sprintf("🌐 Application URL: http://localhost:%s\n\n", port))
		result.WriteString("Starting server in new terminal window...\n")

		started := c.launchInTerminal(runCmd)
		if started {
			result.WriteString("✓ Server is now running in a new terminal window!\n\n")
		} else {
			result.WriteString(fmt.Sprintf("Could not open terminal. To start manually:\n  cd %s\n  %s\n", c.ProjectName, runCmd))
		}

		result.WriteString(fmt.Sprintf("🌐 Application URL: http://localhost:%s\n", port))
	} else if runCmd != "" && !strings.EqualFold(runCmd, "NONE") {
		result.WriteString(fmt.Sprintf("Running: %s\n\n--- Application Output ---\n", runCmd))
		out, runErr := c.runShellCmd(runCmd)
		if runErr != nil {
			result.WriteString(fmt.Sprintf("Error: %v\n", runErr))
		}
		result.WriteString(string(out))
		result.WriteString("\n--- End Output ---\n")
	}

	if runInstructions != "" && !strings.EqualFold(runInstructions, "NONE") {
		result.WriteString(fmt.Sprintf("\nTo run again: %s\n", runInstructions))
	}

	c.State = StateDone
	result.WriteString("\nApp Creation complete!")
	return result.String(), nil
}






// runScriptWithFallback runs a shell script, trying bash first then sh as fallback.
// It returns the combined output, any error, and a log string describing what was attempted.
func runScriptWithFallback(scriptPath, dir string) ([]byte, error, string) {
	var log strings.Builder

	// Try bash first
	log.WriteString(fmt.Sprintf("ai-assist %s\nAttempting to run script with bash...\n", getCurrentTime()))
	cmdBash := exec.Command("bash", filepath.Base(scriptPath))
	cmdBash.Dir = dir
	out, err := cmdBash.CombinedOutput()
	if err == nil {
		return out, nil, log.String()
	}

	log.WriteString(fmt.Sprintf("bash failed: %v\nOutput:\n%s\n\nai-assist %s\nFalling back to sh...\n", err, string(out), getCurrentTime()))

	// Fallback to sh
	cmdSh := exec.Command("sh", filepath.Base(scriptPath))
	cmdSh.Dir = dir
	out, err = cmdSh.CombinedOutput()
	if err != nil {
		log.WriteString(fmt.Sprintf("sh also failed: %v\nOutput:\n%s\n", err, string(out)))
		return out, err, log.String()
	}

	log.WriteString(fmt.Sprintf("sh succeeded.\n"))
	return out, nil, log.String()
}

func aicall(client ai.AIClient, model, prompt string) (string, error) {
	ch, err := client.Generate(prompt, model, nil, nil)
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
	// Try the standard "Project Name:" format first (handles **bold**, `backtick`, or plain)
	matches := projectNameRe.FindStringSubmatch(plan)
	if len(matches) >= 2 && matches[1] != "" {
		name := strings.TrimSpace(matches[1])
		name = strings.Trim(name, "`")
		name = strings.Trim(name, "*")
		return strings.ToLower(name)
	}

	// Try to find project name in bold markdown on its own line: **project-name**
	boldRe := regexp.MustCompile(`(?m)\*\*([a-zA-Z0-9][a-zA-Z0-9\-_]*)\*\*`)
	boldMatches := boldRe.FindAllStringSubmatch(plan, -1)
	if len(boldMatches) > 0 && len(boldMatches[0]) >= 2 {
		return strings.ToLower(boldMatches[0][1])
	}

	// Try to find project name in backticks on its own line
	backticksRe := regexp.MustCompile("(?m)`([a-zA-Z0-9][a-zA-Z0-9\\-_]*)`")
	backticksMatches := backticksRe.FindAllStringSubmatch(plan, -1)

	// Look for the first backtick-enclosed name that looks like a project name
	for _, match := range backticksMatches {
		if len(match) >= 2 {
			name := match[1]
			if strings.Contains(name, "-") || strings.Contains(name, "_") {
				return strings.ToLower(name)
			}
		}
	}

	// If we found any backtick name, use the first one
	if len(backticksMatches) > 0 && len(backticksMatches[0]) >= 2 {
		return strings.ToLower(backticksMatches[0][1])
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




// stripGoModInit removes "go mod init ..." segments from a command string
// when go.mod was already generated during file creation. It handles both
// chained commands (&&) and standalone lines, preserving the rest of the script.
func stripGoModInit(cmds string) string {
	var cleaned []string
	for _, line := range strings.Split(cmds, "\n") {
		// Handle && chains within a single line
		parts := strings.Split(line, "&&")
		var kept []string
		for _, p := range parts {
			trimmed := strings.TrimSpace(p)
			if strings.HasPrefix(trimmed, "go mod init") {
				continue
			}
			kept = append(kept, p)
		}
		if len(kept) > 0 {
			cleaned = append(cleaned, strings.Join(kept, "&&"))
		}
	}
	return strings.TrimSpace(strings.Join(cleaned, "\n"))
}

// runGoModTidy runs go mod tidy to download dependencies
func (c *AutonomousCreator) runGoModTidy() error {
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = c.ProjectDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("go mod tidy failed: %v\nOutput: %s", err, string(output))
	}
	return nil
}

// isPortAvailable checks if a port is available for binding
func isPortAvailable(port string) (bool, error) {
	// Try to listen on the port
	listener, err := net.Listen("tcp", "localhost:"+port)
	if err != nil {
		return false, err
	}
	listener.Close()
	return true, nil
}

// cleanAIResponse strips markdown code fences from AI responses.
func cleanAIResponse(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```bash")
	s = strings.TrimPrefix(s, "```sh")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	return strings.TrimSpace(s)
}

// extractAIField extracts a value from a structured AI response like "FIELD: value".
func extractAIField(response, field string) string {
	prefix := field + ":"
	for _, line := range strings.Split(response, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToUpper(trimmed), strings.ToUpper(prefix)) {
			val := strings.TrimSpace(trimmed[len(prefix):])
			val = cleanAIResponse(val)
			return val
		}
	}
	return ""
}

// buildCodeContext creates a string representation of all generated files and their
// contents so the AI can inspect the actual code when deciding what to do.
func (c *AutonomousCreator) buildCodeContext() string {
	var sb strings.Builder
	for path, content := range c.FilesToMake {
		sb.WriteString(fmt.Sprintf("--- %s ---\n%s\n\n", path, content))
	}
	return sb.String()
}

// runShellCmd executes a shell command in the project directory and returns output.
func (c *AutonomousCreator) runShellCmd(cmdStr string) ([]byte, error) {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/C", cmdStr)
	} else {
		cmd = exec.Command("bash", "-c", cmdStr)
	}
	cmd.Dir = c.ProjectDir
	return cmd.CombinedOutput()
}

// aiDrivenFix asks the AI to analyze an error, fix the code, and retry the command.
// It performs up to 3 fix attempts. Returns a log of what happened or an error if
// all attempts fail.
func (c *AutonomousCreator) aiDrivenFix(failedCmd, errorOutput, context string) (string, error) {
	codeCtx := c.buildCodeContext()
	var result strings.Builder
	maxAttempts := 3

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if c.logger != nil {
			c.logger.Log("Fix attempt %d/%d for %s error", attempt, maxAttempts, context)
		}
		result.WriteString(fmt.Sprintf("ai-assist %s\nFix attempt %d/%d for %s error...\n", getCurrentTime(), attempt, maxAttempts, context))

		// Ask AI to diagnose and fix
		prompt := fmt.Sprintf(`A %s error occurred while running: %s

Error output:
%s

Here are the project files:
%s

Analyze the error and provide fixes. Return your response in EXACTLY this format:

For each file that needs changing:
FIX_FILE: <relative path>
FIX_CONTENT: <complete new file content>
END_FIX

After all fixes:
RETRY_CMD: <the command to retry, or SAME to reuse the original>

If the error cannot be fixed by changing code (e.g. missing system tool), return:
UNFIXABLE: <explanation>

Rules:
- Provide the COMPLETE file content, not just the changed lines.
- Do NOT wrap content in markdown code fences.
- Base fixes on the actual error message and code.`, context, failedCmd, errorOutput, codeCtx)

		fixResponse, err := aicall(c.AIClient, c.Model, prompt)
		if err != nil {
			return result.String(), fmt.Errorf("AI fix call failed: %v", err)
		}

		// Check if unfixable
		if unfixable := extractAIField(fixResponse, "UNFIXABLE"); unfixable != "" {
			return result.String(), fmt.Errorf("AI determined error is unfixable: %s", unfixable)
		}

		// Apply fixes
		fixesApplied := 0
		lines := strings.Split(fixResponse, "\n")
		for i := 0; i < len(lines); i++ {
			trimmed := strings.TrimSpace(lines[i])
			if strings.HasPrefix(trimmed, "FIX_FILE:") {
				filePath := strings.TrimSpace(trimmed[len("FIX_FILE:"):])
				// Collect content until END_FIX
				var content strings.Builder
				inContent := false
				for i++; i < len(lines); i++ {
					if strings.TrimSpace(lines[i]) == "END_FIX" {
						break
					}
					if !inContent && strings.HasPrefix(strings.TrimSpace(lines[i]), "FIX_CONTENT:") {
						// First line of content might be on the same line as FIX_CONTENT:
						firstLine := strings.TrimSpace(lines[i][len("FIX_CONTENT:"):])
						if firstLine != "" {
							content.WriteString(firstLine)
							content.WriteString("\n")
						}
						inContent = true
						continue
					}
					if inContent {
						content.WriteString(lines[i])
						content.WriteString("\n")
					}
				}
				if filePath != "" && content.Len() > 0 {
					absPath := filepath.Join(c.ProjectDir, filePath)
					os.MkdirAll(filepath.Dir(absPath), 0755)
					cleanContent := cleanAIResponse(content.String())
					if err := os.WriteFile(absPath, []byte(cleanContent), 0644); err == nil {
						c.FilesToMake[filePath] = cleanContent
						fixesApplied++
						result.WriteString(fmt.Sprintf("Fixed: %s\n", filePath))
					}
				}
			}
		}

		if fixesApplied == 0 {
			// Try the fallback fixer if available
			if c.fixer != nil {
				result2, fixErr := c.fallbackFix(errorOutput, context, failedCmd)
				if fixErr == nil && result2 != nil && result2.Success {
					result.WriteString("Fallback fixer resolved the issue.\n")
				}
			}
		}

		// Retry the command
		retryCmd := extractAIField(fixResponse, "RETRY_CMD")
		if retryCmd == "" || strings.EqualFold(retryCmd, "SAME") {
			retryCmd = failedCmd
		}
		retryCmd = cleanAIResponse(retryCmd)

		out, retryErr := c.runShellCmd(retryCmd)
		if retryErr == nil {
			result.WriteString(fmt.Sprintf("✓ %s succeeded after fix.\n", context))
			return result.String(), nil
		}
		errorOutput = string(out) // feed new error into next attempt
		result.WriteString(fmt.Sprintf("Retry failed: %s\n", string(out)))
	}

	return result.String(), fmt.Errorf("%s failed after %d fix attempts", context, maxAttempts)
}

// smokeTestServer starts a server command, waits for it to respond on the given port,
// then kills it. Returns a log and nil on success, or an error if the server didn't respond.
func (c *AutonomousCreator) smokeTestServer(runCmd, port string) (string, error) {
	url := fmt.Sprintf("http://localhost:%s", port)

	// Check port availability
	portAvailable, _ := isPortAvailable(port)
	if !portAvailable {
		return "", fmt.Errorf("port %s is not available", port)
	}

	var serverCmd *exec.Cmd
	if runtime.GOOS == "windows" {
		serverCmd = exec.Command("cmd", "/C", runCmd)
	} else {
		serverCmd = exec.Command("bash", "-c", runCmd)
	}
	serverCmd.Dir = c.ProjectDir
	setProcGroupAttr(serverCmd)

	var stdoutBuf, stderrBuf strings.Builder
	serverCmd.Stdout = &stdoutBuf
	serverCmd.Stderr = &stderrBuf

	var result strings.Builder
	result.WriteString(fmt.Sprintf("ai-assist %s\nSmoke testing server: %s\n", getCurrentTime(), runCmd))
	result.WriteString(fmt.Sprintf("Expecting response at %s\n", url))

	if err := serverCmd.Start(); err != nil {
		return result.String(), fmt.Errorf("failed to start server: %v", err)
	}

	result.WriteString(fmt.Sprintf("Server started (PID %d). Waiting for HTTP response...\n", serverCmd.Process.Pid))

	httpReady := false
	client := &http.Client{Timeout: 2 * time.Second}
	for i := 0; i < 30; i++ {
		time.Sleep(1 * time.Second)
		if i > 0 && i%5 == 0 {
			result.WriteString(fmt.Sprintf("Still waiting... (%d seconds)\n", i))
		}
		resp, err := client.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode < 500 {
				httpReady = true
				break
			}
		}
	}

	killProcessGroup(serverCmd.Process.Pid)
	serverCmd.Process.Wait()

	if !httpReady {
		errMsg := fmt.Sprintf("server did not respond at %s within 30 seconds", url)
		stdout := strings.TrimSpace(stdoutBuf.String())
		stderr := strings.TrimSpace(stderrBuf.String())
		if stdout != "" {
			errMsg += "\nstdout: " + stdout
		}
		if stderr != "" {
			errMsg += "\nstderr: " + stderr
		}
		return result.String(), fmt.Errorf("%s", errMsg)
	}

	result.WriteString(fmt.Sprintf("✓ Server responded at %s — smoke test passed.\n", url))
	return result.String(), nil
}

// launchInTerminal attempts to start a command in a new terminal window.
// Returns true if successful.
func (c *AutonomousCreator) launchInTerminal(cmdStr string) bool {
	var runCmd *exec.Cmd
	if runtime.GOOS == "windows" {
		runCmd = exec.Command("cmd", "/c", "start", "cmd", "/k", cmdStr)
		runCmd.Dir = c.ProjectDir
	} else if runtime.GOOS == "darwin" {
		script := fmt.Sprintf("cd '%s' && %s", c.ProjectDir, cmdStr)
		runCmd = exec.Command("osascript", "-e",
			fmt.Sprintf("tell application \"Terminal\" to do script \"%s\"", script))
	} else if _, err := exec.LookPath("gnome-terminal"); err == nil {
		runCmd = exec.Command("gnome-terminal", "--", "bash", "-c", cmdStr)
		runCmd.Dir = c.ProjectDir
	} else if _, err := exec.LookPath("xterm"); err == nil {
		runCmd = exec.Command("xterm", "-e", "bash", "-c", cmdStr)
		runCmd.Dir = c.ProjectDir
	} else {
		// Fallback: run in background
		bgCmd := exec.Command("bash", "-c", cmdStr)
		bgCmd.Dir = c.ProjectDir
		if err := bgCmd.Start(); err != nil {
			return false
		}
		c.RunningProcess = bgCmd
		return true
	}

	if err := runCmd.Start(); err != nil {
		// Fallback to background
		bgCmd := exec.Command("bash", "-c", cmdStr)
		bgCmd.Dir = c.ProjectDir
		if err := bgCmd.Start(); err != nil {
			return false
		}
		c.RunningProcess = bgCmd
		return true
	}
	c.RunningProcess = runCmd
	return true
}


// Returns "python" or "python3" depending on what's available, or empty string if neither found
func detectPythonBinary() string {
	// Try python first (common on Windows)
	if _, err := exec.LookPath("python"); err == nil {
		return "python"
	}

	// Try python3 (common on Linux/Mac)
	if _, err := exec.LookPath("python3"); err == nil {
		return "python3"
	}

	return ""
}

// detectVenvPython returns the venv Python binary path if a venv exists in projectDir,
// otherwise falls back to the system Python binary.
func detectVenvPython(projectDir string) string {
	var venvPython string
	if runtime.GOOS == "windows" {
		venvPython = filepath.Join(projectDir, "venv", "Scripts", "python.exe")
	} else {
		venvPython = filepath.Join(projectDir, "venv", "bin", "python")
	}
	if _, err := os.Stat(venvPython); err == nil {
		return venvPython
	}
	return detectPythonBinary()
}

// injectPipBootstrap prepends a pip-detection and installation preamble to a
// shell script for Linux and macOS. If pip/pip3 is not found it attempts to
// install it using the platform package manager (apt, dnf/yum, brew) before
// the rest of the script runs. It also sets a _PIP variable that callers can
// use, and aliases bare "pip" to the detected binary.
func injectPipBootstrap(script string) string {
	preamble := `# --- pip bootstrap (injected by ti) ---
_PIP=""
if command -v pip3 >/dev/null 2>&1; then
    _PIP="pip3"
elif command -v pip >/dev/null 2>&1; then
    _PIP="pip"
else
    echo "pip not found – attempting to install..."
    if command -v apt-get >/dev/null 2>&1; then
        sudo apt-get update -qq && sudo apt-get install -y python3-pip
    elif command -v dnf >/dev/null 2>&1; then
        sudo dnf install -y python3-pip
    elif command -v yum >/dev/null 2>&1; then
        sudo yum install -y python3-pip
    elif command -v brew >/dev/null 2>&1; then
        brew install python
    else
        echo "ERROR: cannot install pip – no supported package manager found (apt, dnf, yum, brew)" >&2
        exit 1
    fi
    if command -v pip3 >/dev/null 2>&1; then
        _PIP="pip3"
    elif command -v pip >/dev/null 2>&1; then
        _PIP="pip"
    else
        echo "ERROR: pip installation failed" >&2
        exit 1
    fi
fi
shopt -s expand_aliases 2>/dev/null || true
alias pip="$_PIP"
# --- end pip bootstrap ---
`
	// Preserve existing shebang at the top
	shebang := ""
	rest := script
	if strings.HasPrefix(script, "#!") {
		idx := strings.Index(script, "\n")
		if idx != -1 {
			shebang = script[:idx+1]
			rest = script[idx+1:]
		}
	}
	return shebang + preamble + rest
}

// convertToWindowsCommands converts Unix-style shell commands to Windows batch commands
func convertToWindowsCommands(cmds, pythonBinary string) string {
	// Replace python3 with detected binary or fallback to python
	if pythonBinary != "" {
		cmds = strings.ReplaceAll(cmds, "python3", pythonBinary)
	} else {
		cmds = strings.ReplaceAll(cmds, "python3", "python")
	}

	// Replace Unix venv activation with Windows activation
	cmds = strings.ReplaceAll(cmds, "source venv/bin/activate", "venv\\Scripts\\activate")
	cmds = strings.ReplaceAll(cmds, "source venv\\bin\\activate", "venv\\Scripts\\activate")

	// Replace Unix path separators in venv paths
	cmds = strings.ReplaceAll(cmds, "venv/bin/", "venv\\Scripts\\")

	// Convert "mkdir -p dir1 dir2 ..." to "if not exist dir1 mkdir dir1 & if not exist dir2 mkdir dir2 ..."
	// Windows mkdir doesn't support -p or multiple dirs in one call
	mkdirRe := regexp.MustCompile(`mkdir\s+-p\s+(.+)`)
	cmds = mkdirRe.ReplaceAllStringFunc(cmds, func(match string) string {
		sub := mkdirRe.FindStringSubmatch(match)
		if len(sub) < 2 {
			return match
		}
		dirs := strings.Fields(sub[1])
		var parts []string
		for _, d := range dirs {
			// Convert forward slashes to backslashes
			d = strings.ReplaceAll(d, "/", "\\")
			parts = append(parts, fmt.Sprintf("if not exist %s mkdir %s", d, d))
		}
		return strings.Join(parts, " & ")
	})

	// Convert Unix-style quoted echo redirections to unquoted Windows equivalents.
	// e.g. echo "fastapi" > requirements.txt  →  echo fastapi > requirements.txt
	// Windows cmd includes the literal quotes in the output, which breaks pip.
	echoRe := regexp.MustCompile(`(?m)echo\s+"([^"]+)"`)
	cmds = echoRe.ReplaceAllString(cmds, "echo $1")

	return cmds
}
