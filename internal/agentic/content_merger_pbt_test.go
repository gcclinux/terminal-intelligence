package agentic

import (
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// appendIntent is the standard append EditIntent used across append property tests.
var appendIntent = EditIntent{OperationType: "append", Confidence: 0.8}

// replaceIntent is the standard replace EditIntent used across replace property tests.
var replaceIntent = EditIntent{OperationType: "replace", Confidence: 0.8}

// insertIntent is the standard insert EditIntent used across insert property tests.
var insertIntent = EditIntent{OperationType: "insert", Confidence: 0.8}

// nonEmptyLineGen generates a non-empty single line (no newlines).
func nonEmptyLineGen() *rapid.Generator[string] {
	return rapid.StringMatching(`[a-zA-Z0-9 _\-]{1,40}`)
}

// multiLineContentGen generates non-empty content with 1-5 lines.
func multiLineContentGen() *rapid.Generator[string] {
	return rapid.Custom[string](func(t *rapid.T) string {
		n := rapid.IntRange(1, 5).Draw(t, "lineCount")
		lines := make([]string, n)
		for i := 0; i < n; i++ {
			lines[i] = nonEmptyLineGen().Draw(t, "line")
		}
		return strings.Join(lines, "\n")
	})
}

// Feature: content-preserving-file-updates, Property 7: Append merge preserves existing content and appends new content
// For any non-empty existing file content and any non-empty new content, calling
// ContentMerger.Merge(existing, new, appendIntent) should produce a result that starts with
// the existing content, ends with the new content, and has exactly one newline character
// separating the existing content from the new content.
// **Validates: Requirements 3.1, 3.6, 8.1**
func TestProperty7_AppendMergePreservesContent(t *testing.T) {
	cm := NewContentMerger()

	rapid.Check(t, func(t *rapid.T) {
		existing := multiLineContentGen().Draw(t, "existing")
		newContent := multiLineContentGen().Draw(t, "newContent")

		result, err := cm.Merge(existing, newContent, appendIntent)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Normalize: strip trailing newline from existing for prefix check,
		// since mergeAppend consumes a trailing newline as the separator.
		trimmedExisting := strings.TrimRight(existing, "\n")

		// Result must start with the existing content (minus any trailing newline).
		if !strings.HasPrefix(result, trimmedExisting) {
			t.Fatalf("result does not start with existing content.\nexisting: %q\nresult:   %q",
				existing, result)
		}

		// Result must end with the new content.
		if !strings.HasSuffix(result, newContent) {
			t.Fatalf("result does not end with new content.\nnewContent: %q\nresult:     %q",
				newContent, result)
		}

		// There must be exactly one newline separating existing from new content.
		// Find where existing ends and new content begins.
		// The separator is the character(s) between trimmedExisting and newContent.
		sepStart := len(trimmedExisting)
		sepEnd := len(result) - len(newContent)
		separator := result[sepStart:sepEnd]

		if separator != "\n" {
			t.Fatalf("expected exactly one newline separator, got %q.\nexisting: %q\nnewContent: %q\nresult: %q",
				separator, existing, newContent, result)
		}
	})
}

// Feature: content-preserving-file-updates, Property 8: Append merge line count
// For any non-empty existing file content and any non-empty new content, the line count of the
// result from ContentMerger.Merge(existing, new, appendIntent) should equal the line count of
// the existing content plus the line count of the new content (accounting for the separator).
// **Validates: Requirements 8.2**
func TestProperty8_AppendMergeLineCount(t *testing.T) {
	cm := NewContentMerger()

	rapid.Check(t, func(t *rapid.T) {
		existing := multiLineContentGen().Draw(t, "existing")
		newContent := multiLineContentGen().Draw(t, "newContent")

		result, err := cm.Merge(existing, newContent, appendIntent)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		existingLines := countLines(existing)
		newLines := countLines(newContent)
		resultLines := countLines(result)

		// When existing ends with \n, mergeAppend does existingContent + newContent
		// (the trailing \n acts as the separator, so no extra line is added).
		// When existing does NOT end with \n, mergeAppend does existingContent + "\n" + newContent
		// (the \n is the separator, which doesn't add an extra line — it joins the last
		// existing line with the first new line on separate lines).
		//
		// In both cases the result line count = existingLines + newLines,
		// UNLESS existing ends with \n, in which case strings.Split counts an extra
		// empty trailing element. The trailing \n is consumed as separator, so:
		//   - existing NOT ending with \n: result = existingLines + newLines
		//   - existing ending with \n: existing has an extra empty line from Split,
		//     but that empty line becomes the separator, so result = existingLines - 1 + newLines
		//
		// Simplified: result lines = countLines(trimmedExisting) + newLines
		// where trimmedExisting = strings.TrimRight(existing, "\n")
		trimmedExisting := strings.TrimRight(existing, "\n")
		expectedLines := countLines(trimmedExisting) + newLines

		if resultLines != expectedLines {
			t.Fatalf("line count mismatch: existing=%d (trimmed=%d), new=%d, result=%d, expected=%d\nexisting: %q\nnewContent: %q\nresult: %q",
				existingLines, countLines(trimmedExisting), newLines, resultLines, expectedLines,
				existing, newContent, result)
		}
	})
}

// Feature: content-preserving-file-updates, Property 9: Replace merge returns new content
// For any existing file content and any non-empty new content, calling
// ContentMerger.Merge(existing, new, replaceIntent) should return a result equal to the new content.
// **Validates: Requirements 3.3**
func TestProperty9_ReplaceMergeReturnsNewContent(t *testing.T) {
	cm := NewContentMerger()

	rapid.Check(t, func(t *rapid.T) {
		// Existing can be anything, including empty.
		existing := rapid.StringMatching(`[a-zA-Z0-9 \n_\-]{0,80}`).Draw(t, "existing")
		newContent := multiLineContentGen().Draw(t, "newContent")

		result, err := cm.Merge(existing, newContent, replaceIntent)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result != newContent {
			t.Fatalf("replace merge did not return newContent.\nexpected: %q\ngot:      %q",
				newContent, result)
		}
	})
}

// Feature: content-preserving-file-updates, Property 10: Replace with same content is idempotent
// For any valid non-empty file content string, calling
// ContentMerger.Merge(content, content, replaceIntent) should return a result identical to
// the original content.
// **Validates: Requirements 8.3**
func TestProperty10_ReplaceIdempotent(t *testing.T) {
	cm := NewContentMerger()

	rapid.Check(t, func(t *rapid.T) {
		content := multiLineContentGen().Draw(t, "content")

		result, err := cm.Merge(content, content, replaceIntent)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result != content {
			t.Fatalf("replace with same content is not idempotent.\noriginal: %q\nresult:   %q",
				content, result)
		}
	})
}

// Feature: content-preserving-file-updates, Property 11: Insert merge places content after anchor
// For any existing file content containing at least two lines, if the anchor line exists in the
// existing content, then calling ContentMerger.Merge with insert intent should produce a result
// where the new content appears immediately after the anchor line, and all original lines are
// preserved.
// For insert, the first line of newContent is the anchor, the rest is the content to insert.
// **Validates: Requirements 3.2**
func TestProperty11_InsertMergePlacesContentAfterAnchor(t *testing.T) {
	cm := NewContentMerger()

	rapid.Check(t, func(t *rapid.T) {
		// Generate existing content with at least 2 lines.
		lineCount := rapid.IntRange(2, 6).Draw(t, "lineCount")
		existingLines := make([]string, lineCount)
		for i := 0; i < lineCount; i++ {
			existingLines[i] = nonEmptyLineGen().Draw(t, "existingLine")
		}
		existing := strings.Join(existingLines, "\n")

		// Pick a random line from existing as the anchor.
		anchorIdx := rapid.IntRange(0, lineCount-1).Draw(t, "anchorIdx")
		anchor := existingLines[anchorIdx]

		// Generate content to insert (at least one line).
		insertLineCount := rapid.IntRange(1, 3).Draw(t, "insertLineCount")
		insertLines := make([]string, insertLineCount)
		for i := 0; i < insertLineCount; i++ {
			insertLines[i] = nonEmptyLineGen().Draw(t, "insertLine")
		}
		insertContent := strings.Join(insertLines, "\n")

		// For insert intent, newContent = anchor + "\n" + content to insert.
		newContent := anchor + "\n" + insertContent

		result, err := cm.Merge(existing, newContent, insertIntent)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		resultLines := strings.Split(result, "\n")

		// Find the anchor in the result.
		anchorFound := false
		for i, line := range resultLines {
			if strings.TrimSpace(line) == strings.TrimSpace(anchor) {
				anchorFound = true

				// The inserted content should appear immediately after the anchor.
				// Check that the insert content lines follow the anchor.
				for j, insertLine := range insertLines {
					resultIdx := i + 1 + j
					if resultIdx >= len(resultLines) {
						t.Fatalf("result too short: expected insert line at index %d but result has %d lines",
							resultIdx, len(resultLines))
					}
					if strings.TrimSpace(resultLines[resultIdx]) != strings.TrimSpace(insertLine) {
						t.Fatalf("insert line mismatch at result index %d: expected %q, got %q",
							resultIdx, insertLine, resultLines[resultIdx])
					}
				}
				break
			}
		}

		if !anchorFound {
			t.Fatalf("anchor line %q not found in result:\n%s", anchor, result)
		}

		// All original lines must be preserved in the result.
		for _, origLine := range existingLines {
			if !strings.Contains(result, origLine) {
				t.Fatalf("original line %q not found in result:\n%s", origLine, result)
			}
		}
	})
}
