package docgen

import (
	"testing"
)

func TestCommandParser_Parse_BothFlags(t *testing.T) {
	parser := NewCommandParser()

	tests := []struct {
		name  string
		input string
	}{
		{"flags at start", "/project /doc create a user manual"},
		{"flags reversed", "/doc /project create a user manual"},
		{"flags with extra spaces", "/project  /doc  create a user manual"},
		{"flags in different cases", "/PROJECT /Doc create a user manual"},
		{"flags mixed case", "/Project /DOC create a user manual"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse() error = %v, want nil", err)
			}
			if !result.IsProjectWide {
				t.Errorf("IsProjectWide = false, want true")
			}
			if !result.IsDocRequest {
				t.Errorf("IsDocRequest = false, want true")
			}
			if result.NaturalLanguage == "" {
				t.Errorf("NaturalLanguage is empty, want non-empty")
			}
		})
	}
}

func TestCommandParser_Parse_SingleFlag(t *testing.T) {
	parser := NewCommandParser()

	tests := []struct {
		name            string
		input           string
		wantProjectWide bool
		wantDocRequest  bool
		wantErr         bool
	}{
		{"only project flag", "/project analyze codebase", false, false, true}, // /project alone is no longer valid for doc generation
		{"only doc flag", "/doc API reference", true, true, false},             // /doc now implies project-wide
		{"project flag uppercase", "/PROJECT analyze", false, false, true},     // /project alone is no longer valid for doc generation
		{"doc flag uppercase", "/DOC create manual", true, true, false},        // /doc now implies project-wide
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.Parse(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("Parse() expected error but got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("Parse() error = %v, want nil", err)
			}
			if result.IsProjectWide != tt.wantProjectWide {
				t.Errorf("IsProjectWide = %v, want %v", result.IsProjectWide, tt.wantProjectWide)
			}
			if result.IsDocRequest != tt.wantDocRequest {
				t.Errorf("IsDocRequest = %v, want %v", result.IsDocRequest, tt.wantDocRequest)
			}
		})
	}
}

func TestCommandParser_Parse_NaturalLanguageExtraction(t *testing.T) {
	parser := NewCommandParser()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"flags at start", "/project /doc create a user manual", "create a user manual"},
		{"flags with text before", "please /project /doc create manual", "please create manual"},
		{"flags in middle", "create /project /doc a manual", "create a manual"},
		{"extra whitespace", "/project  /doc   create   manual", "create manual"},
		{"mixed case flags", "/PROJECT /Doc create manual", "create manual"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse() error = %v, want nil", err)
			}
			if result.NaturalLanguage != tt.want {
				t.Errorf("NaturalLanguage = %q, want %q", result.NaturalLanguage, tt.want)
			}
		})
	}
}

func TestCommandParser_Parse_ScopeFilters(t *testing.T) {
	parser := NewCommandParser()

	tests := []struct {
		name        string
		input       string
		wantFilters []string
	}{
		{
			"for module pattern",
			"/project /doc create docs for module auth",
			[]string{"auth"},
		},
		{
			"in directory pattern",
			"/project /doc generate docs in directory internal",
			[]string{"internal"},
		},
		{
			"for package pattern",
			"/project /doc API docs for package utils",
			[]string{"utils"},
		},
		{
			"multiple filters",
			"/project /doc docs for module auth in directory internal",
			[]string{"auth", "internal"},
		},
		{
			"no filters",
			"/project /doc create user manual",
			[]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse() error = %v, want nil", err)
			}
			if len(result.ScopeFilters) != len(tt.wantFilters) {
				t.Errorf("ScopeFilters length = %d, want %d", len(result.ScopeFilters), len(tt.wantFilters))
			}
			for i, filter := range tt.wantFilters {
				if i >= len(result.ScopeFilters) {
					break
				}
				if result.ScopeFilters[i] != filter {
					t.Errorf("ScopeFilters[%d] = %q, want %q", i, result.ScopeFilters[i], filter)
				}
			}
		})
	}
}

func TestCommandParser_Parse_ErrorCases(t *testing.T) {
	parser := NewCommandParser()

	tests := []struct {
		name    string
		input   string
		wantErr string
	}{
		{"empty input", "", "no command provided"},
		{"whitespace only", "   ", "no command provided"},
		{"no flags", "create a user manual", "no documentation flags detected"},
		{"text without flags", "please help me", "no documentation flags detected"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parser.Parse(tt.input)
			if err == nil {
				t.Fatalf("Parse() error = nil, want error containing %q", tt.wantErr)
			}
			if err.Error() != tt.wantErr {
				t.Errorf("Parse() error = %q, want %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestCommandParser_Parse_FlagsInMiddle(t *testing.T) {
	parser := NewCommandParser()

	input := "create /project a /doc user manual"
	result, err := parser.Parse(input)

	if err != nil {
		t.Fatalf("Parse() error = %v, want nil", err)
	}
	if !result.IsProjectWide {
		t.Errorf("IsProjectWide = false, want true")
	}
	if !result.IsDocRequest {
		t.Errorf("IsDocRequest = false, want true")
	}
	// Natural language should have flags removed
	if result.NaturalLanguage != "create a user manual" {
		t.Errorf("NaturalLanguage = %q, want %q", result.NaturalLanguage, "create a user manual")
	}
}

func TestCommandParser_Parse_EdgeCases(t *testing.T) {
	parser := NewCommandParser()

	t.Run("flags in different cases", func(t *testing.T) {
		tests := []struct {
			name  string
			input string
		}{
			{"uppercase PROJECT", "/PROJECT /doc create manual"},
			{"uppercase DOC", "/project /DOC create manual"},
			{"mixed case Project", "/Project /doc create manual"},
			{"mixed case Doc", "/project /Doc create manual"},
			{"all uppercase", "/PROJECT /DOC create manual"},
			{"random mixed case", "/PrOjEcT /DoC create manual"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result, err := parser.Parse(tt.input)
				if err != nil {
					t.Fatalf("Parse() error = %v, want nil", err)
				}
				if !result.IsProjectWide {
					t.Errorf("IsProjectWide = false, want true")
				}
				if !result.IsDocRequest {
					t.Errorf("IsDocRequest = false, want true")
				}
				if result.NaturalLanguage == "" {
					t.Errorf("NaturalLanguage is empty, want non-empty")
				}
			})
		}
	})

	t.Run("flags with extra whitespace", func(t *testing.T) {
		tests := []struct {
			name  string
			input string
			want  string
		}{
			{"double space after flags", "/project  /doc  create manual", "create manual"},
			{"triple space after flags", "/project   /doc   create manual", "create manual"},
			{"multiple spaces in text", "/project /doc create   manual   here", "create manual here"},
			{"tabs after flags", "/project\t/doc\tcreate manual", "create manual"},
			{"mixed whitespace", "/project  \t /doc \t  create manual", "create manual"},
			{"leading whitespace", "  /project /doc create manual", "create manual"},
			{"trailing whitespace", "/project /doc create manual  ", "create manual"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result, err := parser.Parse(tt.input)
				if err != nil {
					t.Fatalf("Parse() error = %v, want nil", err)
				}
				if result.NaturalLanguage != tt.want {
					t.Errorf("NaturalLanguage = %q, want %q", result.NaturalLanguage, tt.want)
				}
			})
		}
	})

	t.Run("flags in middle of text", func(t *testing.T) {
		tests := []struct {
			name  string
			input string
			want  string
		}{
			{"flags separated in middle", "create /project a /doc manual", "create a manual"},
			{"flags at end", "create manual /project /doc", "create manual"},
			{"flags scattered", "please /project create /doc a manual", "please create a manual"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result, err := parser.Parse(tt.input)
				if err != nil {
					t.Fatalf("Parse() error = %v, want nil", err)
				}
				if !result.IsProjectWide {
					t.Errorf("IsProjectWide = false, want true")
				}
				if !result.IsDocRequest {
					t.Errorf("IsDocRequest = false, want true")
				}
				if result.NaturalLanguage != tt.want {
					t.Errorf("NaturalLanguage = %q, want %q", result.NaturalLanguage, tt.want)
				}
			})
		}
	})

	t.Run("single flag in middle of text", func(t *testing.T) {
		tests := []struct {
			name            string
			input           string
			want            string
			wantProjectWide bool
			wantDocRequest  bool
			wantErr         bool
		}{
			{"project flag in middle", "create /project documentation", "create documentation", false, false, true}, // /project alone should error
			{"doc flag in middle", "create /doc for the API", "create for the API", true, true, false},              // /doc implies project-wide
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result, err := parser.Parse(tt.input)
				if tt.wantErr {
					if err == nil {
						t.Fatalf("Parse() expected error but got nil")
					}
					return
				}
				if err != nil {
					t.Fatalf("Parse() error = %v, want nil", err)
				}
				if result.IsProjectWide != tt.wantProjectWide {
					t.Errorf("IsProjectWide = %v, want %v", result.IsProjectWide, tt.wantProjectWide)
				}
				if result.IsDocRequest != tt.wantDocRequest {
					t.Errorf("IsDocRequest = %v, want %v", result.IsDocRequest, tt.wantDocRequest)
				}
				if result.NaturalLanguage != tt.want {
					t.Errorf("NaturalLanguage = %q, want %q", result.NaturalLanguage, tt.want)
				}
			})
		}
	})

	t.Run("empty input errors", func(t *testing.T) {
		tests := []struct {
			name    string
			input   string
			wantErr string
		}{
			{"completely empty", "", "no command provided"},
			{"only spaces", "   ", "no command provided"},
			{"only tabs", "\t\t", "no command provided"},
			{"mixed whitespace", " \t \n ", "no command provided"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := parser.Parse(tt.input)
				if err == nil {
					t.Fatalf("Parse() error = nil, want error %q", tt.wantErr)
				}
				if err.Error() != tt.wantErr {
					t.Errorf("Parse() error = %q, want %q", err.Error(), tt.wantErr)
				}
			})
		}
	})

	t.Run("no-flag input errors", func(t *testing.T) {
		tests := []struct {
			name    string
			input   string
			wantErr string
		}{
			{"plain text", "create a user manual", "no documentation flags detected"},
			{"question", "how do I create documentation?", "no documentation flags detected"},
			{"similar words", "project documentation needed", "no documentation flags detected"},
			{"partial flag", "/proj /do create manual", "no documentation flags detected"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := parser.Parse(tt.input)
				if err == nil {
					t.Fatalf("Parse() error = nil, want error %q", tt.wantErr)
				}
				if err.Error() != tt.wantErr {
					t.Errorf("Parse() error = %q, want %q", err.Error(), tt.wantErr)
				}
			})
		}
	})
}
