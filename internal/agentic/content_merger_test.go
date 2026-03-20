package agentic

import (
	"strings"
	"testing"
)

func TestNewContentMerger(t *testing.T) {
	cm := NewContentMerger()
	if cm == nil {
		t.Fatal("NewContentMerger returned nil")
	}
}

func TestContentMerger_Merge_EmptyNewContent(t *testing.T) {
	cm := NewContentMerger()
	intent := EditIntent{OperationType: "append", Confidence: 0.9}
	_, err := cm.Merge("existing", "", intent)
	if err == nil {
		t.Fatal("expected error for empty newContent")
	}
}

func TestContentMerger_Merge_Append(t *testing.T) {
	cm := NewContentMerger()
	intent := EditIntent{OperationType: "append", Confidence: 0.9}

	tests := []struct {
		name     string
		existing string
		new      string
		want     string
	}{
		{
			name:     "basic append",
			existing: "line1\nline2",
			new:      "line3\nline4",
			want:     "line1\nline2\nline3\nline4",
		},
		{
			name:     "existing ends with newline",
			existing: "line1\nline2\n",
			new:      "line3",
			want:     "line1\nline2\nline3",
		},
		{
			name:     "empty existing",
			existing: "",
			new:      "line1",
			want:     "line1",
		},
		{
			name:     "single line existing no trailing newline",
			existing: "hello",
			new:      "world",
			want:     "hello\nworld",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := cm.Merge(tt.existing, tt.new, intent)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestContentMerger_Merge_Replace(t *testing.T) {
	cm := NewContentMerger()
	intent := EditIntent{OperationType: "replace", Confidence: 0.9}

	got, err := cm.Merge("old content", "new content", intent)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "new content" {
		t.Errorf("got %q, want %q", got, "new content")
	}
}

func TestContentMerger_Merge_Insert_AnchorFound(t *testing.T) {
	cm := NewContentMerger()
	intent := EditIntent{OperationType: "insert", Confidence: 0.9}

	existing := "line1\nline2\nline3"
	// First line of newContent is the anchor, rest is content to insert
	newContent := "line2\ninserted line"

	got, err := cm.Merge(existing, newContent, intent)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "line1\nline2\ninserted line\nline3"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestContentMerger_Merge_Insert_AnchorNotFound(t *testing.T) {
	cm := NewContentMerger()
	intent := EditIntent{OperationType: "insert", Confidence: 0.9}

	existing := "line1\nline2"
	newContent := "nonexistent anchor\nnew content"

	got, err := cm.Merge(existing, newContent, intent)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should fall back to append with the full newContent
	if !strings.HasPrefix(got, "line1\nline2") {
		t.Error("expected result to start with existing content")
	}
	if !strings.HasSuffix(got, newContent) {
		t.Error("expected result to end with newContent (fallback to append)")
	}
}

func TestContentMerger_Merge_Patch(t *testing.T) {
	cm := NewContentMerger()
	intent := EditIntent{OperationType: "patch", Confidence: 1.0}

	existing := "hello world\nfoo bar"
	newContent := "~~~SEARCH\nhello world\n~~~REPLACE\nhello universe\n~~~END"

	got, err := cm.Merge(existing, newContent, intent)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(got, "hello universe") {
		t.Errorf("expected patched content to contain 'hello universe', got %q", got)
	}
}

func TestContentMerger_Merge_UnknownOperation(t *testing.T) {
	cm := NewContentMerger()
	intent := EditIntent{OperationType: "unknown", Confidence: 0.5}

	_, err := cm.Merge("existing", "new", intent)
	if err == nil {
		t.Fatal("expected error for unknown operation type")
	}
}

func TestContentMerger_findAnchorLine(t *testing.T) {
	cm := NewContentMerger()

	tests := []struct {
		name    string
		content string
		anchor  string
		wantIdx int
	}{
		{"found exact", "line1\nline2\nline3", "line2", 1},
		{"found with whitespace", "  line1  \nline2", "line1", 0},
		{"not found", "line1\nline2", "missing", -1},
		{"empty content", "", "anchor", -1},
		{"first line", "target\nother", "target", 0},
		{"last line", "other\ntarget", "target", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cm.findAnchorLine(tt.content, tt.anchor)
			if got != tt.wantIdx {
				t.Errorf("got %d, want %d", got, tt.wantIdx)
			}
		})
	}
}
