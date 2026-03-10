package docgen

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/user/terminal-intelligence/internal/ai"
)

const (
	// DefaultModel is the AI model used when none is specified.
	DefaultModel = "llama3.2"
	// DefaultTokenBudget is the maximum rune-length of the serialised symbol
	// block included in a single prompt before truncation kicks in.
	DefaultTokenBudget = 8000
)

// AIGenerator builds structured prompts from an AnalysisResult and streams
// responses from the configured AIClient.
type AIGenerator struct {
	client      ai.AIClient
	model       string
	feedback    *FeedbackManager
	tokenBudget int
}

// NewAIGenerator creates a new AIGenerator. When model is empty, DefaultModel
// is used and the fallback is logged via feedback.NotifyProgress. Optional
// functional options (e.g. WithTokenBudget) may be supplied.
func NewAIGenerator(client ai.AIClient, model string, feedback *FeedbackManager, opts ...func(*AIGenerator)) *AIGenerator {
	g := &AIGenerator{
		client:      client,
		model:       model,
		feedback:    feedback,
		tokenBudget: DefaultTokenBudget,
	}

	for _, opt := range opts {
		opt(g)
	}

	if g.model == "" {
		g.model = DefaultModel
		if g.feedback != nil {
			g.feedback.NotifyProgress("AI model not specified", fmt.Sprintf("using default model %q", DefaultModel))
		}
	}

	return g
}

// WithTokenBudget returns a functional option that sets the token budget on an
// AIGenerator.
func WithTokenBudget(n int) func(*AIGenerator) {
	return func(g *AIGenerator) {
		g.tokenBudget = n
	}
}

// BuildPrompt constructs a deterministic prompt string from the analysis result
// and the requested documentation type. Pure function — no side effects, no I/O.
func (g *AIGenerator) BuildPrompt(result *AnalysisResult, docType DocumentationType) string {
	var sb strings.Builder

	// ── DOC_TYPE ──────────────────────────────────────────────────────────────
	sb.WriteString("DOC_TYPE: ")
	sb.WriteString(docType.String())
	sb.WriteString("\n---\n")

	// ── PROJECT CONTEXT ───────────────────────────────────────────────────────
	sb.WriteString("PROJECT CONTEXT\n")

	// Languages
	sb.WriteString("Languages: ")
	sb.WriteString(strings.Join(detectLanguages(result), ", "))
	sb.WriteString("\n")

	// Dependencies
	sb.WriteString("Dependencies: ")
	sb.WriteString(formatDependencies(result))
	sb.WriteString("\n")

	// Config files
	sb.WriteString("Config files: ")
	if result != nil && result.Configuration != nil && len(result.Configuration.ConfigFiles) > 0 {
		sb.WriteString(strings.Join(result.Configuration.ConfigFiles, ", "))
	}
	sb.WriteString("\n")

	// Package manifests
	sb.WriteString("Package manifests: ")
	if result != nil && result.Configuration != nil && len(result.Configuration.PackageManifests) > 0 {
		sb.WriteString(strings.Join(result.Configuration.PackageManifests, ", "))
	}
	sb.WriteString("\n---\n")

	// ── README ────────────────────────────────────────────────────────────────
	if result != nil && result.Documentation != nil && result.Documentation.ReadmeContent != "" {
		sb.WriteString("README\n")
		sb.WriteString(result.Documentation.ReadmeContent)
		sb.WriteString("\n---\n")
	}

	// ── API SYMBOLS (DocTypeAPI only) ─────────────────────────────────────────
	if docType == DocTypeAPI {
		sb.WriteString("API SYMBOLS\n")
		sb.WriteString(buildSymbolBlock(result, g.tokenBudget))
		sb.WriteString("---\n")
	}

	// ── INSTRUCTIONS ──────────────────────────────────────────────────────────
	sb.WriteString("INSTRUCTIONS\n")
	sb.WriteString(fmt.Sprintf(
		"Generate a %s for the project described above.\nWrite in Markdown. Be specific to this project; do not use generic placeholder text.",
		docType.String(),
	))

	return sb.String()
}

// ParsePromptMetadata extracts the DocumentationType encoded in a prompt
// produced by BuildPrompt. Used for round-trip testing only.
func ParsePromptMetadata(prompt string) (DocumentationType, error) {
	firstLine := prompt
	if idx := strings.Index(prompt, "\n"); idx >= 0 {
		firstLine = prompt[:idx]
	}

	const prefix = "DOC_TYPE: "
	if !strings.HasPrefix(firstLine, prefix) {
		return 0, fmt.Errorf("prompt does not contain a DOC_TYPE header")
	}

	value := strings.TrimSpace(firstLine[len(prefix):])

	for _, dt := range []DocumentationType{
		DocTypeUserManual,
		DocTypeInstallation,
		DocTypeAPI,
		DocTypeTutorial,
		DocTypeGeneral,
	} {
		if dt.String() == value {
			return dt, nil
		}
	}

	return 0, fmt.Errorf("unknown DOC_TYPE value: %q", value)
}
// Generate calls the AI client, streams all chunks, and returns a GeneratedDoc
// whose Content is the concatenation of every chunk in order.
// It calls feedback.NotifyProgress before streaming begins.
// If the AI client returns an error, feedback.NotifyError is called and the
// error is returned.
func (g *AIGenerator) Generate(result *AnalysisResult, docType DocumentationType) (*GeneratedDoc, error) {
	prompt := g.BuildPrompt(result, docType)

	g.feedback.NotifyProgress("AI generating...", docType.String())

	ch, err := g.client.Generate(prompt, g.model, nil)
	if err != nil {
		g.feedback.NotifyError(err)
		return nil, err
	}

	var sb strings.Builder
	for chunk := range ch {
		sb.WriteString(chunk)
	}

	return &GeneratedDoc{
		Type:     docType,
		Content:  sb.String(),
		Filename: docType.Filename(),
	}, nil
}

// GenerateMultiple calls Generate for each requested doc type, tolerating
// partial failures. Successful GeneratedDoc values are collected and returned.
// For each failure, feedback.NotifyError is called and iteration continues.
// Returns all successful docs; error is always nil (partial success is not an
// error — the pipeline handles the empty-result case).
func (g *AIGenerator) GenerateMultiple(result *AnalysisResult, docTypes []DocumentationType) ([]*GeneratedDoc, error) {
	var docs []*GeneratedDoc

	for _, docType := range docTypes {
		doc, err := g.Generate(result, docType)
		if err != nil {
			// NotifyError already called inside Generate; just continue.
			continue
		}
		docs = append(docs, doc)
	}

	return docs, nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

// detectLanguages infers the project's programming languages from package
// manifests. Falls back to file extensions in CodeStructure.Packages when no
// manifests are present.
func detectLanguages(result *AnalysisResult) []string {
	if result == nil {
		return nil
	}

	seen := make(map[string]bool)
	var langs []string

	add := func(lang string) {
		if !seen[lang] {
			seen[lang] = true
			langs = append(langs, lang)
		}
	}

	if result.Configuration != nil {
		for _, m := range result.Configuration.PackageManifests {
			base := filepath.Base(m)
			switch base {
			case "go.mod":
				add("Go")
			case "package.json":
				add("JavaScript/TypeScript")
			case "requirements.txt", "Pipfile":
				add("Python")
			case "Cargo.toml":
				add("Rust")
			}
		}
	}

	// Fallback: derive from file extensions in CodeStructure.Packages
	if len(langs) == 0 && result.CodeStructure != nil {
		for _, pkg := range result.CodeStructure.Packages {
			ext := strings.ToLower(filepath.Ext(pkg.Path))
			switch ext {
			case ".go":
				add("Go")
			case ".js", ".ts", ".jsx", ".tsx":
				add("JavaScript/TypeScript")
			case ".py":
				add("Python")
			case ".rs":
				add("Rust")
			}
		}
	}

	return langs
}

// formatDependencies serialises runtime dependencies as "name@version" pairs.
func formatDependencies(result *AnalysisResult) string {
	if result == nil || result.Dependencies == nil || len(result.Dependencies.Runtime) == 0 {
		return ""
	}

	parts := make([]string, 0, len(result.Dependencies.Runtime))
	for _, dep := range result.Dependencies.Runtime {
		if dep.Version != "" {
			parts = append(parts, dep.Name+"@"+dep.Version)
		} else {
			parts = append(parts, dep.Name)
		}
	}
	return strings.Join(parts, ", ")
}

// buildSymbolBlock serialises exported symbols and applies token-budget
// truncation. Returns the symbol text (without the surrounding section header).
func buildSymbolBlock(result *AnalysisResult, budget int) string {
	if result == nil || result.CodeStructure == nil {
		return ""
	}

	cs := result.CodeStructure

	// Collect all symbol lines in declaration order:
	// functions → structs → interfaces
	var lines []string

	for _, fn := range cs.Functions {
		if fn.IsExported {
			if fn.Signature != "" {
				lines = append(lines, fn.Signature)
			} else {
				lines = append(lines, fn.Name)
			}
		}
	}

	for _, st := range cs.Structs {
		if st.IsExported {
			var sb strings.Builder
			sb.WriteString(st.Name)
			for _, f := range st.Fields {
				sb.WriteString("\n  ")
				sb.WriteString(f.Name)
				sb.WriteString(" ")
				sb.WriteString(f.Type)
			}
			lines = append(lines, sb.String())
		}
	}

	for _, iface := range cs.Interfaces {
		if iface.IsExported {
			var sb strings.Builder
			sb.WriteString(iface.Name)
			for _, m := range iface.Methods {
				sb.WriteString("\n  ")
				sb.WriteString(m.Name)
			}
			lines = append(lines, sb.String())
		}
	}

	if len(lines) == 0 {
		return ""
	}

	// Apply token-budget truncation
	var sb strings.Builder
	runesUsed := 0
	included := 0

	for _, line := range lines {
		candidate := line + "\n"
		runesUsed += len([]rune(candidate))
		if runesUsed > budget {
			break
		}
		sb.WriteString(candidate)
		included++
	}

	omitted := len(lines) - included
	if omitted > 0 {
		sb.WriteString(fmt.Sprintf("[TRUNCATED: symbol list exceeded token budget — %d symbols omitted]\n", omitted))
	}

	return sb.String()
}
