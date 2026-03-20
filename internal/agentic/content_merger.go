package agentic

import (
	"fmt"
	"strings"
)

// ContentMerger combines existing file content with AI-generated content
// according to the classified EditIntent.
type ContentMerger struct{}

// NewContentMerger creates a new ContentMerger instance.
func NewContentMerger() *ContentMerger {
	return &ContentMerger{}
}

// Merge combines existing content with new content according to the EditIntent.
// For "append": concatenates with a single newline separator.
// For "insert": places new content after the anchor line, falls back to append.
// For "replace": returns newContent as-is.
// For "patch": delegates to applySearchReplace (existing function).
func (cm *ContentMerger) Merge(existingContent, newContent string, intent EditIntent) (string, error) {
	if newContent == "" {
		return "", fmt.Errorf("newContent must not be empty")
	}

	switch intent.OperationType {
	case "append":
		return cm.mergeAppend(existingContent, newContent), nil
	case "insert":
		return cm.mergeInsert(existingContent, newContent), nil
	case "replace":
		return newContent, nil
	case "patch":
		return cm.mergePatch(existingContent, newContent)
	default:
		return "", fmt.Errorf("unknown operation type: %s", intent.OperationType)
	}
}

// mergeAppend concatenates existing content with new content, ensuring exactly
// one newline separator between them.
func (cm *ContentMerger) mergeAppend(existingContent, newContent string) string {
	if existingContent == "" {
		return newContent
	}
	if strings.HasSuffix(existingContent, "\n") {
		return existingContent + newContent
	}
	return existingContent + "\n" + newContent
}

// mergeInsert places new content after the anchor line in existing content.
// The first line of newContent is treated as the anchor line; the rest is the
// content to insert. If the anchor is not found, falls back to append.
func (cm *ContentMerger) mergeInsert(existingContent, newContent string) string {
	lines := strings.SplitN(newContent, "\n", 2)
	anchor := lines[0]
	insertContent := ""
	if len(lines) > 1 {
		insertContent = lines[1]
	}

	anchorIdx := cm.findAnchorLine(existingContent, anchor)
	if anchorIdx == -1 {
		// Anchor not found — fall back to append behavior.
		return cm.mergeAppend(existingContent, newContent)
	}

	existingLines := strings.Split(existingContent, "\n")
	var result []string
	result = append(result, existingLines[:anchorIdx+1]...)
	if insertContent != "" {
		result = append(result, insertContent)
	}
	result = append(result, existingLines[anchorIdx+1:]...)
	return strings.Join(result, "\n")
}

// mergePatch parses newContent as SEARCH/REPLACE blocks and applies them
// sequentially to existingContent using the existing applySearchReplace function.
func (cm *ContentMerger) mergePatch(existingContent, newContent string) (string, error) {
	patches := parseSearchReplace(newContent)
	if len(patches) == 0 {
		// If no SEARCH/REPLACE blocks found, try a single direct apply.
		return applySearchReplace(existingContent, "", newContent)
	}

	result := existingContent
	for _, p := range patches {
		var err error
		result, err = applySearchReplace(result, p.search, p.replace)
		if err != nil {
			return "", fmt.Errorf("patch apply failed: %w", err)
		}
	}
	return result, nil
}

// findAnchorLine locates the line number of the anchor in existingContent.
// Returns -1 if not found. The match is based on trimmed line comparison.
func (cm *ContentMerger) findAnchorLine(existingContent, anchor string) int {
	lines := strings.Split(existingContent, "\n")
	trimmedAnchor := strings.TrimSpace(anchor)
	for i, line := range lines {
		if strings.TrimSpace(line) == trimmedAnchor {
			return i
		}
	}
	return -1
}
