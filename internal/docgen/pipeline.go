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
	aiGenerator   *AIGenerator
	writer        *FileWriter
	feedback      *FeedbackManager
	workspaceRoot string
	aiClient      ai.AIClient
	model         string
}

// NewPipeline creates a new documentation generation pipeline
func NewPipeline(workspaceRoot string, aiClient ai.AIClient, model string, chatPane ChatPaneInterface) *Pipeline {
	feedback := NewFeedbackManager(chatPane)
	aiGen := NewAIGenerator(aiClient, model, feedback)
	return &Pipeline{
		parser:        NewCommandParser(),
		classifier:    NewRequestClassifier(),
		writer:        NewFileWriter(workspaceRoot),
		feedback:      feedback,
		aiGenerator:   aiGen,
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

	// Check AI availability before creating any files
	available, err := p.aiClient.IsAvailable()
	if err != nil || !available {
		p.feedback.NotifyError(fmt.Errorf("AI service is unavailable"))
		return nil
	}

	// Notify start
	p.feedback.NotifyStart(classification.Types)

	// Step 1: Create empty document files first
	var filesToGenerate []*GeneratedDoc
	for _, docType := range classification.Types {
		doc := &GeneratedDoc{
			Type:     docType,
			Content:  "", // Empty initially
			Filename: docType.Filename(),
		}
		filesToGenerate = append(filesToGenerate, doc)
	}

	// Step 2: Create files and open the first one in editor
	var results []*WriteResult
	var firstFile *WriteResult

	for i, doc := range filesToGenerate {
		// Check for conflicts first
		if p.writer.CheckExists(doc.Filename) {
			conflict := &WriteResult{
				Filename: doc.Filename,
				Path:     p.workspaceRoot + "/" + doc.Filename,
				Existed:  true,
				Written:  false,
			}
			p.feedback.NotifyFileConflict([]*WriteResult{conflict})
			continue
		}

		// Create empty file
		emptyDoc := &GeneratedDoc{
			Type:     doc.Type,
			Content:  "# " + documentationTypeName(doc.Type) + "\n\nGenerating...",
			Filename: doc.Filename,
		}

		result, err := p.writer.Write(emptyDoc, false)
		if err != nil {
			p.feedback.NotifyProgress("Warning", fmt.Sprintf("Failed to create %s: %v", doc.Filename, err))
			continue
		}

		results = append(results, result)

		// Step 3: Open first file in editor and notify
		if i == 0 {
			firstFile = result
			p.feedback.NotifyFileCreated(result)
		}
	}

	if len(results) == 0 {
		// All files had conflicts - this is not an error, just nothing to do
		return nil
	}

	// Step 4: Analyze project
	p.feedback.NotifyProgress("Analyzing project", "")
	p.analyzer = NewProjectAnalyzer(p.workspaceRoot, parsed.ScopeFilters)
	analysisResult, err := p.analyzer.Analyze()
	if err != nil {
		p.feedback.NotifyError(fmt.Errorf("analysis failed: %w", err))
		return err
	}

	// Step 5: Generate documentation
	p.feedback.NotifyProgress("Generating documentation", "")
	docs, err := p.aiGenerator.GenerateMultiple(analysisResult, classification.Types)
	if err != nil {
		p.feedback.NotifyError(fmt.Errorf("generation failed: %w", err))
		return err
	}

	// Step 6: Update files with generated content
	p.feedback.NotifyProgress("Writing documentation", "")
	var finalResults []*WriteResult

	for _, doc := range docs {
		// Write the actual content (overwrite the placeholder)
		result, err := p.writer.Write(doc, true)
		if err != nil {
			p.feedback.NotifyProgress("Warning", fmt.Sprintf("Failed to write %s: %v", doc.Filename, err))
			continue
		}
		finalResults = append(finalResults, result)
	}

	// Step 7: Reload the first file in editor to show updated content
	if firstFile != nil {
		if err := p.feedback.chatPane.OpenFileInEditor(firstFile.Path); err != nil {
			p.feedback.NotifyProgress("Warning", fmt.Sprintf("Could not reload file in editor: %v", err))
		}
	}

	// Step 8: Notify completion
	if len(finalResults) > 0 {
		p.feedback.NotifyComplete(finalResults, parsed.ScopeFilters)
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

	// Check AI availability before creating any files
	available, err := p.aiClient.IsAvailable()
	if err != nil || !available {
		p.feedback.NotifyError(fmt.Errorf("AI service is unavailable"))
		return nil
	}

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
	docs, err := p.aiGenerator.GenerateMultiple(analysisResult, classification.Types)
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
