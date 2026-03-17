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
	"github.com/user/terminal-intelligence/internal/types"
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

	// Cumulative token usage across all AI calls in this creation session.
	InputTokens  int
	OutputTokens int
	TotalTokens  int

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
1. A project name. If the user specified a name, use that EXACT name as-is. Otherwise suggest a short, lowercase, hyphenated name.
2. A high-level architecture overview.
3. The COMPLETE files and folder structure that will be created. List EVERY file with its full relative path from the project root (e.g. "backend/main.go", "frontend/index.html", "frontend/styles.css"). If the application has multiple components (frontend, backend, API, etc.), organize them into separate folders.
4. The commands needed to initialize dependencies (e.g. go mod init, pip install).
5. The command to run the application to test it.

IMPORTANT RULES:
- Use the programming language the user requested. If no language is specified, choose the most appropriate one.
- If the user asks for a web application, you MUST include both frontend AND backend folders with complete implementations.
- List every single file that will be created — do not summarize with "..." or "etc".
- The project name MUST appear as "Project Name: <name>" on its own line.`, c.Description)

	plan, err := c.aicallAndTrack(prompt)
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
	codeCtx := c.buildCodeContext()
	sysCtx := getSystemContext()

	// Ask the AI for the dependency setup commands, giving it full system context
	prompt := fmt.Sprintf(`You are an expert software engineer setting up a new project.

Implementation plan:
%s

Generated files:
%s

System environment:
%s

What are the precise terminal commands to install this project's dependencies?
Return ONLY a shell script with the commands. No markdown formatting, no explanations.

Rules:
- Base your answer on the ACTUAL FILES and the system environment.
- If the system has PEP 668 (externally-managed-environment), you MUST create a virtual environment first.
- For Go projects: do NOT include "go mod init" if go.mod already exists. Just use "go mod tidy".
- For Python projects on PEP 668 systems: use "python3 -m venv venv && source venv/bin/activate && pip install -r requirements.txt" (adjust paths as needed).
- Assume we are already inside the project directory.`, c.Plan, codeCtx, sysCtx)

	cmdsStr, err := c.aicallAndTrack(prompt)
	if err != nil {
		return "", err
	}

	cmdsStr = cleanAIResponse(cmdsStr)

	// Safety: strip "go mod init" if go.mod already exists
	goModPath := filepath.Join(c.ProjectDir, "go.mod")
	if _, statErr := os.Stat(goModPath); statErr == nil {
		cmdsStr = stripGoModInit(cmdsStr)
	}

	if cmdsStr == "" {
		c.State = StateTesting
		return fmt.Sprintf("ai-assist %s\nNo dependencies to install.\n\nMoving to testing...", getCurrentTime()), nil
	}

	if c.logger != nil {
		c.logger.Log("Installing dependencies: %s", cmdsStr)
	}

	// Execute the dependency commands
	out, cmdErr := c.runShellCmd(cmdsStr)
	if cmdErr != nil {
		errorOutput := string(out)
		if c.logger != nil {
			c.logger.Log("Dependency setup failed: %v", cmdErr)
			c.logger.Log("Output: %s", errorOutput)
		}

		// Use AI-driven fix to resolve the dependency error
		fixResult, fixErr := c.aiDrivenFix(cmdsStr, errorOutput, "dependency setup")
		if fixErr != nil {
			return "", fmt.Errorf("dependency setup failed: %v\nOutput:\n%s", fixErr, errorOutput)
		}

		// aiDrivenFix succeeded
		if c.logger != nil {
			c.logger.Log("Dependency setup resolved by AI fix")
		}
		c.State = StateTesting
		return fmt.Sprintf("ai-assist %s\n%s\nDependencies installed after fix.\n\nMoving to testing...", getCurrentTime(), fixResult), nil
	}

	c.State = StateTesting
	return fmt.Sprintf("ai-assist %s\nDependencies installed successfully.\n\nMoving to testing...", getCurrentTime()), nil
}

func (c *AutonomousCreator) doFileCreation() (string, error) {
	prompt := fmt.Sprintf(`You are an expert autonomous software engineer.
Given the implementation plan below, generate ALL the necessary code files for this project.

IMPLEMENTATION PLAN:
%s

CRITICAL RULES:
1. You MUST create EVERY file and folder described in the plan above. Do not skip any.
2. If the plan specifies a frontend folder, you MUST generate frontend files inside that folder.
3. If the plan specifies a backend folder, you MUST generate backend files inside that folder.
4. All file paths must be RELATIVE to the project root (e.g. "backend/main.go", "frontend/index.html").
5. Do NOT prefix paths with the project name — files are placed inside the project directory automatically.
6. Generate complete, working code — not stubs or placeholders.
7. Follow the EXACT folder structure from the plan.

Return the files inside standard Markdown code blocks with the relative filepath specified immediately before the code block.

Example format:
**backend/main.go**
`+"```go"+`
package main
// full implementation ...
`+"```"+`

**frontend/index.html**
`+"```html"+`
<!DOCTYPE html>
<!-- full implementation ... -->
`+"```"+`

Only return the file paths and code blocks. No other text.`, c.Plan)

	response, err := c.aicallAndTrack(prompt)
	if err != nil {
		return "", err
	}

	c.FilesToMake = c.parseFileBlocks(response)

	// Write files to disk
	createdFiles := c.writeFilesToDisk()

	// Validate structure against the plan and ask AI to fill gaps
	missingResult, err := c.validateAndFillStructure(createdFiles)
	if err != nil {
		// Non-fatal: log but continue
		if c.logger != nil {
			c.logger.Log("Structure validation warning: %v", err)
		}
	}
	if missingResult != "" {
		createdFiles = c.getFileList()
	}

	var resultMsg strings.Builder
	resultMsg.WriteString(fmt.Sprintf("ai-assist %s\nGenerated and saved %d files:\n- %s\n", getCurrentTime(), len(createdFiles), strings.Join(createdFiles, "\n- ")))
	if missingResult != "" {
		resultMsg.WriteString(missingResult)
	}
	resultMsg.WriteString("\nMoving to install dependencies...")

	c.State = StateDependencies
	return resultMsg.String(), nil
}

// parseFileBlocks parses the AI response for "**path/to/file.ext**\n```lang\ncontent\n```" blocks.
func (c *AutonomousCreator) parseFileBlocks(response string) map[string]string {
	files := make(map[string]string)
	lines := strings.Split(response, "\n")
	var currentFile string
	var currentContent strings.Builder
	inBlock := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check for file name
		if !inBlock && strings.HasPrefix(trimmed, "**") && strings.HasSuffix(trimmed, "**") {
			currentFile = strings.Trim(trimmed, "* ")
			// Strip leading project name prefix if the AI accidentally included it
			// e.g. "ricardo/backend/main.go" -> "backend/main.go"
			if c.ProjectName != "" && strings.HasPrefix(currentFile, c.ProjectName+"/") {
				currentFile = strings.TrimPrefix(currentFile, c.ProjectName+"/")
			}
			continue
		}

		if strings.HasPrefix(trimmed, "```") {
			if inBlock {
				// End of block
				if currentFile != "" {
					files[currentFile] = currentContent.String()
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
	return files
}

// writeFilesToDisk writes all files in FilesToMake to the project directory.
func (c *AutonomousCreator) writeFilesToDisk() []string {
	createdFiles := []string{}
	for relPath, content := range c.FilesToMake {
		// Port 5000 is blocked on Windows (firewall) and macOS Monterey+ (AirPlay).
		content = strings.ReplaceAll(content, "port=5000", "port=8080")
		content = strings.ReplaceAll(content, "port = 5000", "port = 8080")
		content = strings.ReplaceAll(content, ":5000", ":8080")
		c.FilesToMake[relPath] = content

		absPath := filepath.Join(c.ProjectDir, relPath)
		os.MkdirAll(filepath.Dir(absPath), 0755)

		if err := os.WriteFile(absPath, []byte(content), 0644); err != nil {
			if c.logger != nil {
				c.logger.Log("Failed to write %s: %v", relPath, err)
			}
			continue
		}
		createdFiles = append(createdFiles, relPath)
	}
	return createdFiles
}

// getFileList returns a sorted list of all file paths in FilesToMake.
func (c *AutonomousCreator) getFileList() []string {
	list := make([]string, 0, len(c.FilesToMake))
	for f := range c.FilesToMake {
		list = append(list, f)
	}
	return list
}

// validateAndFillStructure asks the AI to compare the generated files against the plan
// and generates any missing files. Returns a status message and error.
func (c *AutonomousCreator) validateAndFillStructure(createdFiles []string) (string, error) {
	fileList := strings.Join(createdFiles, "\n")

	prompt := fmt.Sprintf(`You are an expert software engineer validating a project structure.

IMPLEMENTATION PLAN:
%s

FILES ACTUALLY CREATED:
%s

Compare the plan against the files that were created. Identify ANY files or folders that the plan describes but are MISSING from the created list.

If ALL files from the plan are present, respond with exactly:
STRUCTURE_OK

If files are missing, generate the missing files. Return them in this format:

MISSING_FILES:
**<relative-path>**
`+"```<lang>"+`
<complete file content>
`+"```"+`

Only output STRUCTURE_OK or the missing files. No other text.`, c.Plan, fileList)

	response, err := c.aicallAndTrack(prompt)
	if err != nil {
		return "", err
	}

	trimmed := strings.TrimSpace(response)
	if strings.HasPrefix(trimmed, "STRUCTURE_OK") {
		return "", nil
	}

	// Parse and write missing files
	missingFiles := c.parseFileBlocks(response)
	if len(missingFiles) == 0 {
		return "", nil
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("\nai-assist %s\nStructure validation found %d missing files — generating them now:\n", getCurrentTime(), len(missingFiles)))

	for relPath, content := range missingFiles {
		content = strings.ReplaceAll(content, ":5000", ":8080")
		c.FilesToMake[relPath] = content

		absPath := filepath.Join(c.ProjectDir, relPath)
		os.MkdirAll(filepath.Dir(absPath), 0755)

		if err := os.WriteFile(absPath, []byte(content), 0644); err != nil {
			if c.logger != nil {
				c.logger.Log("Failed to write missing file %s: %v", relPath, err)
			}
			continue
		}
		result.WriteString(fmt.Sprintf("- %s\n", relPath))
	}

	return result.String(), nil
}

func (c *AutonomousCreator) doTesting() (string, error) {
	codeCtx := c.buildCodeContext()
	sysCtx := getSystemContext()

	// Ask the AI to analyze the actual code and tell us how to verify it
	prompt := fmt.Sprintf(`You are an expert software engineer. A project was just generated with this plan:
%s

Here are the actual files and their contents:
%s

System environment:
%s

Analyze the code and answer these questions in EXACTLY this format (one answer per line, no extra text):
BUILD_CMD: <single shell command to compile/build the project, or NONE if not needed>
TEST_CMD: <single shell command to run tests or verify the build, or NONE if no tests>
IS_SERVER: <YES or NO - does this application start a long-running HTTP server?>
RUN_CMD: <single shell command to start the application, or NONE>
PORT: <port number the server listens on, or NONE>

Rules:
- Base your answers on the ACTUAL CODE and system environment, not assumptions.
- For Go projects the build command is typically "go build -o <name>" and run is "./<name>".
- For Python projects: if a venv directory exists, use venv/bin/python and venv/bin/pip. Otherwise use python3.
- If the system has PEP 668, commands MUST use the venv python (e.g. venv/bin/python, venv/bin/uvicorn).
- Do NOT wrap commands in markdown. Return raw commands only.
- Assume we are already inside the project directory.`, c.Plan, codeCtx, sysCtx)

	analysis, err := c.aicallAndTrack(prompt)
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

	summary, err := c.aicallAndTrack(prompt)
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
	sysCtx := getSystemContext()

	// Ask the AI how to build and run this specific project
	prompt := fmt.Sprintf(`You are an expert software engineer. A project was generated with this plan:
%s

Here are the actual files:
%s

System environment:
%s

Provide the commands to build and run this application in EXACTLY this format (one per line, no extra text):
BUILD_CMD: <shell command to build, or NONE>
RUN_CMD: <shell command to run the application>
IS_SERVER: <YES or NO - is this a long-running server?>
PORT: <port number if server, or NONE>
RUN_INSTRUCTIONS: <one-line human-readable instruction for the user to run it manually>

Rules:
- Base answers on the ACTUAL CODE and system environment.
- For Python: if a venv directory exists, use venv/bin/python and venv/bin/uvicorn etc.
- Do NOT wrap commands in markdown.
- Assume we are already inside the project directory.`, c.Plan, codeCtx, sysCtx)

	analysis, err := c.aicallAndTrack(prompt)
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
	resp, usage, err := aicallWithTokens(client, model, prompt)
	_ = usage
	return resp, err
}

// aicallAndTrack calls the AI and accumulates token usage on the creator.
func (c *AutonomousCreator) aicallAndTrack(prompt string) (string, error) {
	resp, usage, err := aicallWithTokens(c.AIClient, c.Model, prompt)
	c.InputTokens += usage.InputTokens
	c.OutputTokens += usage.OutputTokens
	c.TotalTokens += usage.InputTokens + usage.OutputTokens
	return resp, err
}

func aicallWithTokens(client ai.AIClient, model, prompt string) (string, types.TokenUsage, error) {
	var tokenUsage types.TokenUsage
	onTokenUsage := func(usage types.TokenUsage) {
		tokenUsage = usage
	}

	ch, err := client.Generate(prompt, model, nil, onTokenUsage)
	if err != nil {
		return "", tokenUsage, err
	}

	var sb strings.Builder
	for chunk := range ch {
		sb.WriteString(chunk)
	}
	return sb.String(), tokenUsage, nil
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

// getSystemContext gathers runtime environment information (OS, shell, Python
// version, Go version, etc.) so the AI can make informed decisions about
// commands that are appropriate for this specific system.
func getSystemContext() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("OS: %s/%s\n", runtime.GOOS, runtime.GOARCH))

	// Shell
	shell := os.Getenv("SHELL")
	if shell == "" && runtime.GOOS == "windows" {
		shell = "cmd"
	}
	sb.WriteString(fmt.Sprintf("Shell: %s\n", shell))

	// Go version
	if out, err := exec.Command("go", "version").Output(); err == nil {
		sb.WriteString(fmt.Sprintf("Go: %s\n", strings.TrimSpace(string(out))))
	}

	// Python version
	for _, py := range []string{"python3", "python"} {
		if out, err := exec.Command(py, "--version").Output(); err == nil {
			sb.WriteString(fmt.Sprintf("Python: %s\n", strings.TrimSpace(string(out))))
			// Check for PEP 668 (externally-managed-environment)
			testOut, testErr := exec.Command(py, "-m", "pip", "install", "--dry-run", "pip").CombinedOutput()
			if testErr != nil && strings.Contains(string(testOut), "externally-managed-environment") {
				sb.WriteString("Python-PEP668: YES (must use venv for pip install)\n")
			}
			break
		}
	}

	return sb.String()
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

// buildCodeContextFromDisk re-reads all known project files from disk so the AI
// sees the latest state (including any fixes applied by the fallback fixer).
func (c *AutonomousCreator) buildCodeContextFromDisk() string {
	var sb strings.Builder
	for relPath := range c.FilesToMake {
		absPath := filepath.Join(c.ProjectDir, relPath)
		data, err := os.ReadFile(absPath)
		if err != nil {
			// Fall back to in-memory content
			sb.WriteString(fmt.Sprintf("--- %s ---\n%s\n\n", relPath, c.FilesToMake[relPath]))
			continue
		}
		content := string(data)
		c.FilesToMake[relPath] = content // update in-memory map
		sb.WriteString(fmt.Sprintf("--- %s ---\n%s\n\n", relPath, content))
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

// aiDrivenFix asks the AI to analyze an error, fix the code or suggest alternative
// commands, and retry. It performs up to 3 fix attempts. The AI can respond with:
//   - FIX_FILE / FIX_CONTENT / END_FIX: to patch source files
//   - FIX_CMD: to provide a corrected command to run instead
//   - RETRY_CMD: the command to retry after fixes (or SAME)
//   - UNFIXABLE: if the error cannot be resolved
func (c *AutonomousCreator) aiDrivenFix(failedCmd, errorOutput, context string) (string, error) {
	var result strings.Builder
	maxAttempts := 3
	sysCtx := getSystemContext()

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		codeCtx := c.buildCodeContextFromDisk()

		if c.logger != nil {
			c.logger.Log("Fix attempt %d/%d for %s error", attempt, maxAttempts, context)
		}
		result.WriteString(fmt.Sprintf("ai-assist %s\nFix attempt %d/%d for %s error...\n", getCurrentTime(), attempt, maxAttempts, context))

		prompt := fmt.Sprintf(`You are an expert software engineer debugging a project.
A %s error occurred while running: %s

Error output:
%s

System environment:
%s

Here are the current project files:
%s

Analyze the error carefully and provide a fix. Return your response in EXACTLY this format:

OPTION A — If the fix requires changing source files:
For each file that needs changing:
FIX_FILE: <relative path>
FIX_CONTENT: <complete new file content>
END_FIX

After all file fixes:
RETRY_CMD: <the command to retry, or SAME to reuse the original>

OPTION B — If the fix requires a different command (e.g. environment setup, venv creation, different flags):
FIX_CMD: <the corrected shell command or script that should be run instead>
RETRY_CMD: <the command to retry after FIX_CMD succeeds, or SAME>

OPTION C — If the error cannot be fixed:
UNFIXABLE: <explanation>

Rules:
- Provide COMPLETE file content for FIX_FILE, not just changed lines.
- Do NOT wrap content in markdown code fences.
- Base fixes on the actual error message, code, AND system environment.
- For Python on systems with PEP 668 (externally-managed-environment), use a virtual environment.
- FIX_CMD is for running setup/environment commands (like creating a venv, installing system packages, etc.)
- You can combine FIX_FILE and FIX_CMD if both code changes and command changes are needed.
- Assume we are already inside the project directory.`, context, failedCmd, errorOutput, sysCtx, codeCtx)

		fixResponse, err := c.aicallAndTrack(prompt)
		if err != nil {
			return result.String(), fmt.Errorf("AI fix call failed: %v", err)
		}

		// Check if unfixable
		if unfixable := extractAIField(fixResponse, "UNFIXABLE"); unfixable != "" {
			return result.String(), fmt.Errorf("AI determined error is unfixable: %s", unfixable)
		}

		// Apply file fixes
		fixesApplied := 0
		lines := strings.Split(fixResponse, "\n")
		for i := 0; i < len(lines); i++ {
			trimmed := strings.TrimSpace(lines[i])
			if strings.HasPrefix(trimmed, "FIX_FILE:") {
				filePath := strings.TrimSpace(trimmed[len("FIX_FILE:"):])
				var content strings.Builder
				inContent := false
				for i++; i < len(lines); i++ {
					if strings.TrimSpace(lines[i]) == "END_FIX" {
						break
					}
					if !inContent && strings.HasPrefix(strings.TrimSpace(lines[i]), "FIX_CONTENT:") {
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
						result.WriteString(fmt.Sprintf("Fixed file: %s\n", filePath))
					}
				}
			}
		}

		// Execute FIX_CMD if provided (environment/command fixes)
		fixCmd := extractAIField(fixResponse, "FIX_CMD")
		if fixCmd != "" && !strings.EqualFold(fixCmd, "NONE") {
			fixCmd = cleanAIResponse(fixCmd)
			result.WriteString(fmt.Sprintf("Running fix command: %s\n", fixCmd))
			if c.logger != nil {
				c.logger.Log("Running fix command: %s", fixCmd)
			}
			out, cmdErr := c.runShellCmd(fixCmd)
			if cmdErr != nil {
				result.WriteString(fmt.Sprintf("Fix command failed: %s\n", string(out)))
				errorOutput = string(out)
				continue // try next attempt with the new error
			}
			result.WriteString(fmt.Sprintf("Fix command succeeded.\n"))
			fixesApplied++
		}

		if fixesApplied == 0 {
			// Try the fallback fixer if available
			if c.fixer != nil {
				result2, fixErr := c.fallbackFix(errorOutput, context, failedCmd)
				if fixErr == nil && result2 != nil && result2.Success {
					result.WriteString("Fallback fixer resolved the issue.\n")
					fixesApplied++
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
		errorOutput = string(out)
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
