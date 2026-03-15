package projectctx

import (
	"fmt"
	"strings"
)

// PromptBuilder constructs context-augmented prompts for the AI.
type PromptBuilder struct{}

// NewPromptBuilder creates a new PromptBuilder.
func NewPromptBuilder() *PromptBuilder {
	return &PromptBuilder{}
}

// Build constructs a Context_Prompt from ProjectMetadata, the user's message,
// optional search results, and optional current file context.
// The returned string is the full prompt to send to AIClient.Generate().
func (pb *PromptBuilder) Build(meta *ProjectMetadata, userMessage string, searchResults []string, fileContext string) string {
	var b strings.Builder

	// 1. System instruction directing the AI to answer from context.
	b.WriteString("You are a project-aware AI assistant. Answer the user's question based on the provided project context below. ")
	b.WriteString("If the information is not available in the context, state that clearly rather than guessing.\n\n")

	// 2. Project file tree listing.
	b.WriteString("## Project File Tree\n\n")
	if len(meta.FileTree) > 0 {
		for _, entry := range meta.FileTree {
			b.WriteString(entry)
			b.WriteByte('\n')
		}
	} else {
		b.WriteString("(no files discovered)\n")
	}
	if meta.FileTreeTruncated {
		fmt.Fprintf(&b, "(truncated — showing %d of %d files)\n", len(meta.FileTree), meta.TotalFiles)
	}
	b.WriteByte('\n')

	// 3. Detected language and build system.
	b.WriteString("## Project Info\n\n")
	if meta.Language != "" {
		fmt.Fprintf(&b, "- Language: %s\n", meta.Language)
	} else {
		b.WriteString("- Language: unknown\n")
	}
	if meta.BuildSystem != "" {
		fmt.Fprintf(&b, "- Build System: %s\n", meta.BuildSystem)
	} else {
		b.WriteString("- Build System: unknown\n")
	}
	b.WriteByte('\n')

	// 4. Contents of each discovered key project file.
	if len(meta.KeyFiles) > 0 {
		b.WriteString("## Key Project Files\n\n")
		// Emit in priority order for deterministic output.
		for _, name := range KeyProjectFiles {
			content, ok := meta.KeyFiles[name]
			if !ok {
				continue
			}
			fmt.Fprintf(&b, "### %s\n\n```\n%s\n```\n\n", name, content)
		}
	}

	// 5. Optional search results.
	if len(searchResults) > 0 {
		b.WriteString("## Search Results\n\n")
		for _, sr := range searchResults {
			b.WriteString(sr)
			b.WriteByte('\n')
		}
		b.WriteByte('\n')
	}

	// 6. Optional current file context.
	if fileContext != "" {
		b.WriteString("## Current File Context\n\n")
		b.WriteString(fileContext)
		b.WriteString("\n\n")
	}

	// 7. User's original message.
	b.WriteString("## User Question\n\n")
	b.WriteString(userMessage)
	b.WriteByte('\n')

	// 8. Debug byte count comment.
	prompt := b.String()
	fmt.Fprintf(&b, "\n<!-- context: %d bytes -->\n", len(prompt))

	return b.String()
}
