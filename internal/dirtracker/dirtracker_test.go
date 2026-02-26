package dirtracker

import (
	"path/filepath"
	"testing"
)

// --- Tests for ParseBashCommands (Task 6.1) ---

func TestParseBashCommands(t *testing.T) {
	dt := New("/workspace")

	tests := []struct {
		name     string
		content  string
		expected []DirCommand
	}{
		{
			name:    "simple mkdir",
			content: "mkdir myproject",
			expected: []DirCommand{
				{Type: "mkdir", Path: "myproject", MkdirP: false},
			},
		},
		{
			name:    "mkdir with -p flag",
			content: "mkdir -p a/b/c",
			expected: []DirCommand{
				{Type: "mkdir", Path: "a/b/c", MkdirP: true},
			},
		},
		{
			name:    "simple cd",
			content: "cd myproject",
			expected: []DirCommand{
				{Type: "cd", Path: "myproject"},
			},
		},
		{
			name:    "cd ..",
			content: "cd ..",
			expected: []DirCommand{
				{Type: "cd", Path: ".."},
			},
		},
		{
			name:    "mkdir && cd compound",
			content: "mkdir myproject && cd myproject",
			expected: []DirCommand{
				{Type: "mkdir", Path: "myproject", MkdirP: false},
				{Type: "cd", Path: "myproject"},
			},
		},
		{
			name:     "commented lines are ignored",
			content:  "# mkdir secret\n# cd secret\necho hello",
			expected: []DirCommand{},
		},
		{
			name:     "empty input",
			content:  "",
			expected: []DirCommand(nil),
		},
		{
			name:     "mkdir with no args (malformed)",
			content:  "mkdir",
			expected: []DirCommand(nil),
		},
		{
			name:     "cd with no args (malformed)",
			content:  "cd",
			expected: []DirCommand(nil),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := dt.ParseBashCommands(tc.content)
			if len(got) != len(tc.expected) {
				t.Fatalf("expected %d commands, got %d: %+v", len(tc.expected), len(got), got)
			}
			for i := range got {
				if got[i].Type != tc.expected[i].Type {
					t.Errorf("command[%d].Type = %q, want %q", i, got[i].Type, tc.expected[i].Type)
				}
				if got[i].Path != tc.expected[i].Path {
					t.Errorf("command[%d].Path = %q, want %q", i, got[i].Path, tc.expected[i].Path)
				}
				if got[i].MkdirP != tc.expected[i].MkdirP {
					t.Errorf("command[%d].MkdirP = %v, want %v", i, got[i].MkdirP, tc.expected[i].MkdirP)
				}
			}
		})
	}
}

// --- Tests for ComputeMappings (Task 6.2) ---

func TestComputeMappings(t *testing.T) {
	root := filepath.Join("/", "workspace")

	tests := []struct {
		name     string
		blocks   []CodeBlockInfo
		expected []string
	}{
		{
			name:     "no bash blocks",
			blocks:   []CodeBlockInfo{},
			expected: []string{},
		},
		{
			name: "single bash block with cd",
			blocks: []CodeBlockInfo{
				{Language: "bash", Content: "cd myproject"},
			},
			expected: []string{
				filepath.Join(root, "myproject"),
			},
		},
		{
			name: "multiple bash blocks accumulating state",
			blocks: []CodeBlockInfo{
				{Language: "bash", Content: "mkdir proj && cd proj"},
				{Language: "bash", Content: "mkdir sub && cd sub"},
			},
			expected: []string{
				filepath.Join(root, "proj"),
				filepath.Join(root, "proj", "sub"),
			},
		},
		{
			name: "non-bash blocks inherit directory",
			blocks: []CodeBlockInfo{
				{Language: "bash", Content: "cd myproject"},
				{Language: "go", Content: "package main"},
				{Language: "go", Content: "func main() {}"},
			},
			expected: []string{
				filepath.Join(root, "myproject"),
				filepath.Join(root, "myproject"),
				filepath.Join(root, "myproject"),
			},
		},
		{
			name: "cd past workspace root is clamped",
			blocks: []CodeBlockInfo{
				{Language: "bash", Content: "cd ..\ncd ..\ncd .."},
			},
			expected: []string{
				root,
			},
		},
		{
			name: "absolute path ignored",
			blocks: []CodeBlockInfo{
				{Language: "bash", Content: "cd /usr/local"},
			},
			expected: []string{
				root,
			},
		},
		{
			name: "home path ignored",
			blocks: []CodeBlockInfo{
				{Language: "bash", Content: "cd ~"},
			},
			expected: []string{
				root,
			},
		},
		{
			name: "variable path ignored",
			blocks: []CodeBlockInfo{
				{Language: "bash", Content: "cd $PROJECT_DIR"},
			},
			expected: []string{
				root,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dt := New(root)
			got := dt.ComputeMappings(tc.blocks)
			if len(got) != len(tc.expected) {
				t.Fatalf("expected %d mappings, got %d: %v", len(tc.expected), len(got), got)
			}
			for i := range got {
				if got[i] != tc.expected[i] {
					t.Errorf("mapping[%d] = %q, want %q", i, got[i], tc.expected[i])
				}
			}
		})
	}
}
