package dirtracker

import (
	"os"
	"path/filepath"
	"strings"
)

// CodeBlockInfo represents a code block with its language tag.
type CodeBlockInfo struct {
	Language string // e.g., "bash", "go", "python", ""
	Content  string // The code block content (without fence markers)
}

// DirCommand represents a parsed directory command.
type DirCommand struct {
	Type   string // "mkdir" or "cd"
	Path   string // The directory argument
	MkdirP bool   // True if mkdir had -p flag
}

// DirectoryTracker computes effective working directories from
// a sequence of code blocks in an AI response.
type DirectoryTracker struct {
	workspaceRoot string
}

// New creates a DirectoryTracker anchored to the given workspace root.
// If workspaceRoot is empty, falls back to os.Getwd().
func New(workspaceRoot string) *DirectoryTracker {
	if workspaceRoot == "" {
		wd, err := os.Getwd()
		if err == nil {
			workspaceRoot = wd
		}
	}
	return &DirectoryTracker{workspaceRoot: workspaceRoot}
}

// ParseBashCommands extracts mkdir and cd commands from a single bash
// code block's content. Returns the sequence of directory operations.
func (dt *DirectoryTracker) ParseBashCommands(content string) []DirCommand {
	var commands []DirCommand

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip empty lines and comment lines
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Split on && to handle compound commands
		parts := strings.Split(line, "&&")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}

			cmd := parseCommand(part)
			if cmd != nil {
				commands = append(commands, *cmd)
			}
		}
	}

	return commands
}

// parseCommand parses a single command string into a DirCommand, or nil if not recognized.
func parseCommand(cmd string) *DirCommand {
	fields := strings.Fields(cmd)
	if len(fields) == 0 {
		return nil
	}

	switch fields[0] {
	case "mkdir":
		return parseMkdir(fields[1:])
	case "cd":
		return parseCd(fields[1:])
	default:
		return nil
	}
}

// parseMkdir parses mkdir arguments into a DirCommand.
func parseMkdir(args []string) *DirCommand {
	if len(args) == 0 {
		return nil
	}

	mkdirP := false
	var path string

	for _, arg := range args {
		if arg == "-p" {
			mkdirP = true
		} else if !strings.HasPrefix(arg, "-") {
			path = arg
			break
		}
	}

	if path == "" {
		return nil
	}

	return &DirCommand{
		Type:   "mkdir",
		Path:   path,
		MkdirP: mkdirP,
	}
}

// parseCd parses cd arguments into a DirCommand.
func parseCd(args []string) *DirCommand {
	if len(args) == 0 {
		return nil
	}

	path := args[0]
	if path == "" {
		return nil
	}

	return &DirCommand{
		Type: "cd",
		Path: path,
	}
}

// ComputeMappings analyzes the ordered code blocks and returns a
// slice where result[i] is the effective working directory for block i.
// The returned paths are always absolute.
func (dt *DirectoryTracker) ComputeMappings(blocks []CodeBlockInfo) []string {
	if len(blocks) == 0 {
		return []string{}
	}

	mappings := make([]string, len(blocks))
	currentDir := dt.workspaceRoot

	for i, block := range blocks {
		if isBashBlock(block.Language) {
			commands := dt.ParseBashCommands(block.Content)
			for _, cmd := range commands {
				if cmd.Type == "cd" {
					currentDir = dt.resolveCD(currentDir, cmd.Path)
				}
			}
		}
		mappings[i] = currentDir
	}

	return mappings
}

// isBashBlock returns true if the language tag indicates a bash/shell block.
func isBashBlock(lang string) bool {
	l := strings.ToLower(strings.TrimSpace(lang))
	return l == "bash" || l == "sh" || l == "shell"
}

// resolveCD resolves a cd target path relative to the current directory,
// clamped at the workspace root. Returns the new effective directory.
func (dt *DirectoryTracker) resolveCD(currentDir, target string) string {
	// Ignore absolute paths
	if strings.HasPrefix(target, "/") {
		return currentDir
	}

	// Ignore home directory references
	if target == "~" || strings.HasPrefix(target, "~/") {
		return currentDir
	}

	// Ignore $HOME
	if target == "$HOME" || strings.HasPrefix(target, "$HOME/") {
		return currentDir
	}

	// Ignore shell variable expansions
	if strings.HasPrefix(target, "$") {
		return currentDir
	}

	// Resolve relative path
	resolved := filepath.Join(currentDir, target)
	resolved = filepath.Clean(resolved)

	// Clamp at workspace root: ensure resolved path is within workspace
	if !strings.HasPrefix(resolved, dt.workspaceRoot) {
		return dt.workspaceRoot
	}

	return resolved
}
