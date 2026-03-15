package projectctx

import (
	"strings"
)

// questionIndicators are words/symbols that suggest the user is asking a question.
var questionIndicators = []string{
	"?", "how", "what", "where", "why", "which",
	"explain", "describe", "show", "tell", "list",
}

// projectTerms are words that suggest the question is about the project.
var projectTerms = []string{
	"project", "repo", "codebase", "build", "run", "install",
	"deploy", "test", "config", "structure", "architecture",
	"dependency", "dependencies",
}

// searchPhrases are phrases that indicate search-like intent.
// Order matters: longer/more-specific phrases should come first to avoid
// partial matches (e.g. "which file" before "which" in indicators).
var searchPhrases = []string{
	"where is",
	"which file",
	"search for",
	"find",
	"locate",
	"grep",
}

// QueryClassifier classifies user messages to determine if project context is needed.
type QueryClassifier struct{}

// NewQueryClassifier creates a new QueryClassifier.
func NewQueryClassifier() *QueryClassifier {
	return &QueryClassifier{}
}

// Classify analyzes a message and returns a ClassificationResult.
// Messages starting with /ask are always classified as needing project context.
// Otherwise, a message needs project context when it contains at least one
// question indicator AND at least one project term.
// Search-like intent phrases extract search terms into SearchTerms.
func (qc *QueryClassifier) Classify(message string) ClassificationResult {
	result := ClassificationResult{}
	trimmed := strings.TrimSpace(message)
	lower := strings.ToLower(trimmed)

	// /ask prefix (case-insensitive) always triggers project context.
	if strings.HasPrefix(lower, "/ask") {
		result.NeedsProjectContext = true
		// Extract search terms from the remainder after /ask as well.
		remainder := strings.TrimSpace(trimmed[4:])
		if remainder != "" {
			result.SearchTerms = extractSearchTerms(strings.ToLower(remainder))
		}
		return result
	}

	// Check for question indicators and project terms.
	hasIndicator := containsAny(lower, questionIndicators)
	hasTerm := containsAny(lower, projectTerms)

	if hasIndicator && hasTerm {
		result.NeedsProjectContext = true
	}

	// Extract search terms regardless of classification (search intent is independent).
	result.SearchTerms = extractSearchTerms(lower)

	return result
}

// containsAny reports whether text contains any of the given tokens.
// For "?" it does a simple substring check. For words it checks for
// whole-word boundaries to avoid false positives (e.g. "shower" matching "show").
func containsAny(text string, tokens []string) bool {
	for _, tok := range tokens {
		if tok == "?" {
			if strings.Contains(text, "?") {
				return true
			}
			continue
		}
		if containsWord(text, tok) {
			return true
		}
	}
	return false
}

// containsWord checks if text contains tok as a whole word.
func containsWord(text, tok string) bool {
	idx := 0
	for {
		pos := strings.Index(text[idx:], tok)
		if pos < 0 {
			return false
		}
		absPos := idx + pos
		endPos := absPos + len(tok)

		leftOK := absPos == 0 || !isWordChar(text[absPos-1])
		rightOK := endPos == len(text) || !isWordChar(text[endPos])

		if leftOK && rightOK {
			return true
		}
		idx = absPos + 1
		if idx >= len(text) {
			return false
		}
	}
}

// isWordChar reports whether b is a letter, digit, or underscore.
func isWordChar(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '_'
}

// extractSearchTerms looks for search-like intent phrases and extracts the
// term that follows them. Returns nil if no search intent is found.
func extractSearchTerms(lower string) []string {
	var terms []string
	for _, phrase := range searchPhrases {
		idx := 0
		for {
			pos := strings.Index(lower[idx:], phrase)
			if pos < 0 {
				break
			}
			absPos := idx + pos
			afterPhrase := absPos + len(phrase)
			if afterPhrase < len(lower) {
				rest := strings.TrimSpace(lower[afterPhrase:])
				if rest != "" {
					// Take the first "word group" — up to the next punctuation or end.
					term := extractTerm(rest)
					if term != "" {
						terms = append(terms, term)
					}
				}
			}
			idx = absPos + 1
			if idx >= len(lower) {
				break
			}
		}
	}
	return terms
}

// extractTerm extracts the first meaningful term from text.
// It takes words until it hits a sentence-ending punctuation or a search phrase.
func extractTerm(text string) string {
	// Take up to the first sentence-ending punctuation.
	end := len(text)
	for _, sep := range []string{"?", ".", "!", ",", "\n"} {
		if i := strings.Index(text, sep); i >= 0 && i < end {
			end = i
		}
	}
	term := strings.TrimSpace(text[:end])
	// Limit to a reasonable number of words (max 5).
	words := strings.Fields(term)
	if len(words) > 5 {
		words = words[:5]
	}
	return strings.Join(words, " ")
}
