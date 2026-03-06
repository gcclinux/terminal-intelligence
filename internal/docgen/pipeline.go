package docgen

import (
	"fmt"

	"github.com/user/terminal-intelligence/internal/ai"
)

// Pipeline orchestrates the documentation generation process
type Pipeline struct {
	parser        *CommandParser
	classifier    *RequestClassifier
	analyzer      *ProjectAnalyzer
	generator     *DocumentationGenerator
	writer        *FileWriter
	feedback      *FeedbackManager
	workspaceRoot string
	aiClient      ai.AIClient
	model         string
}

// NewPipeline creates a new documentation generation pipeline
func NewPipeline(workspaceRoot string, aiClient ai.AIClient, model string, chatPane ChatPaneInterface) *Pipeline {
	return &Pipeline{
		parser:        NewCommandParser(),
		classifier:    NewRequestClassifier(),
		writer:        NewFileWriter(workspaceRoot),
		feedback:      NewFeedbackManager(chatPane),
		workspaceRoot: workspaceRoot,
		aiClient:      aiClient,
		model:         model,
	}
}

// ProcessCommand processes a user command and generates documentation if applicable
// Returns true if the command was a documentation generation request, false otherwise
func (p *Pipeline) ProcessCommand(input string) (bool, error) {
	// Parse command
	parsed, err := p.parser.Parse(input)
	if err != nil {
		// Not a documentation command
		return false, nil
	}

	// Check if this is a documentation request
	if !parsed.IsDocRequest {
		return false, nil
	}

	// This is a documentation generation request
	return true, p.generateDocumentation(parsed)
}

// generateDocumentation executes the full documentation generation pipeline
func (p *Pipeline) generateDocumentation(parsed *ParsedCommand) error {
	// Classify the request
	classification := p.classifier.Classify(parsed.NaturalLanguage)

	// Notify start
	p.feedback.NotifyStart(classification.Types)

	// Analyze project
	p.feedback.NotifyProgress("Analyzing project", "")
	p.analyzer = NewProjectAnalyzer(p.workspaceRoot, parsed.ScopeFilters)
	analysisResult, err := p.analyzer.Analyze()
	if err != nil {
		p.feedback.NotifyError(fmt.Errorf("analysis failed: %w", err))
		return err
	}

	// Generate documentation
	p.feedback.NotifyProgress("Generating documentation", "")
	p.generator = NewDocumentationGenerator(analysisResult)
	docs, err := p.generator.GenerateMultiple(classification.Types)
	if err != nil {
		p.feedback.NotifyError(fmt.Errorf("generation failed: %w", err))
		return err
	}

	// Write files
	p.feedback.NotifyProgress("Writing files", "")
	var results []*WriteResult
	var conflicts []*WriteResult

	for _, doc := range docs {
		// Check for conflicts first
		if p.writer.CheckExists(doc.Filename) {
			conflict := &WriteResult{
				Filename: doc.Filename,
				Path:     p.workspaceRoot + "/" + doc.Filename,
				Existed:  true,
				Written:  false,
			}
			conflicts = append(conflicts, conflict)
			continue
		}

		// Write the file
		result, err := p.writer.Write(doc, false)
		if err != nil {
			// Log error but continue with other files
			p.feedback.NotifyProgress("Warning", fmt.Sprintf("Failed to write %s: %v", doc.Filename, err))
			continue
		}
		results = append(results, result)
	}

	// Report conflicts if any
	if len(conflicts) > 0 {
		p.feedback.NotifyFileConflict(conflicts)
	}

	// Notify completion
	if len(results) > 0 {
		p.feedback.NotifyComplete(results, parsed.ScopeFilters)
	}

	return nil
}

// ProcessCommandWithOverwrite processes a command and allows overwriting existing files
func (p *Pipeline) ProcessCommandWithOverwrite(input string, overwrite bool) (bool, error) {
	// Parse command
	parsed, err := p.parser.Parse(input)
	if err != nil {
		return false, nil
	}

	if !parsed.IsDocRequest {
		return false, nil
	}

	return true, p.generateDocumentationWithOverwrite(parsed, overwrite)
}

// generateDocumentationWithOverwrite executes the pipeline with overwrite option
func (p *Pipeline) generateDocumentationWithOverwrite(parsed *ParsedCommand, overwrite bool) error {
	// Classify the request
	classification := p.classifier.Classify(parsed.NaturalLanguage)

	// Notify start
	p.feedback.NotifyStart(classification.Types)

	// Analyze project
	p.feedback.NotifyProgress("Analyzing project", "")
	p.analyzer = NewProjectAnalyzer(p.workspaceRoot, parsed.ScopeFilters)
	analysisResult, err := p.analyzer.Analyze()
	if err != nil {
		p.feedback.NotifyError(fmt.Errorf("analysis failed: %w", err))
		return err
	}

	// Generate documentation
	p.feedback.NotifyProgress("Generating documentation", "")
	p.generator = NewDocumentationGenerator(analysisResult)
	docs, err := p.generator.GenerateMultiple(classification.Types)
	if err != nil {
		p.feedback.NotifyError(fmt.Errorf("generation failed: %w", err))
		return err
	}

	// Write files
	p.feedback.NotifyProgress("Writing files", "")
	var results []*WriteResult

	for _, doc := range docs {
		result, err := p.writer.Write(doc, overwrite)
		if err != nil {
			p.feedback.NotifyProgress("Warning", fmt.Sprintf("Failed to write %s: %v", doc.Filename, err))
			continue
		}
		results = append(results, result)
	}

	// Notify completion
	if len(results) > 0 {
		p.feedback.NotifyComplete(results, parsed.ScopeFilters)
	}

	return nil
}
