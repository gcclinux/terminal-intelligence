package docgen

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProjectAnalyzer_AnalyzePythonFile(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()

	// Create a sample Python file
	pythonCode := `"""
This is a module docstring.
It describes the module.
"""

def public_function(param1, param2: str) -> int:
    """This is a public function."""
    return 42

def _private_function():
    """This is a private function."""
    pass

class PublicClass:
    """This is a public class."""
    
    def method1(self, arg):
        """A method."""
        pass
    
    def _private_method(self):
        """A private method."""
        pass

class _PrivateClass:
    """This is a private class."""
    pass
`

	pythonFile := filepath.Join(tmpDir, "test.py")
	err := os.WriteFile(pythonFile, []byte(pythonCode), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create analyzer
	analyzer := NewProjectAnalyzer(tmpDir, nil)

	// Analyze the Python file
	structure, err := analyzer.AnalyzePythonFile("test.py")
	if err != nil {
		t.Fatalf("AnalyzePythonFile failed: %v", err)
	}

	// Verify package info
	if len(structure.Packages) != 1 {
		t.Errorf("Expected 1 package, got %d", len(structure.Packages))
	}
	if structure.Packages[0].Name != "test" {
		t.Errorf("Expected package name 'test', got '%s'", structure.Packages[0].Name)
	}
	if structure.Packages[0].Description == "" {
		t.Error("Expected package description to be extracted")
	}

	// Verify functions
	if len(structure.Functions) != 2 {
		t.Errorf("Expected 2 functions, got %d", len(structure.Functions))
	}

	// Check public function
	foundPublicFunc := false
	for _, fn := range structure.Functions {
		if fn.Name == "public_function" {
			foundPublicFunc = true
			if !fn.IsExported {
				t.Error("public_function should be exported")
			}
			if len(fn.Parameters) != 2 {
				t.Errorf("Expected 2 parameters, got %d", len(fn.Parameters))
			}
		}
	}
	if !foundPublicFunc {
		t.Error("public_function not found")
	}

	// Verify classes
	if len(structure.Classes) != 2 {
		t.Errorf("Expected 2 classes, got %d", len(structure.Classes))
	}

	// Check public class
	foundPublicClass := false
	for _, cls := range structure.Classes {
		if cls.Name == "PublicClass" {
			foundPublicClass = true
			if !cls.IsExported {
				t.Error("PublicClass should be exported")
			}
			if len(cls.Methods) < 1 {
				t.Error("Expected at least 1 method in PublicClass")
			}
		}
	}
	if !foundPublicClass {
		t.Error("PublicClass not found")
	}

	// Verify exports (only public items)
	publicExports := 0
	for _, exp := range structure.Exports {
		if exp.Name[0] != '_' {
			publicExports++
		}
	}
	if publicExports != 2 {
		t.Errorf("Expected 2 public exports, got %d", publicExports)
	}
}

func TestProjectAnalyzer_AnalyzeJavaScriptFile(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()

	// Create a sample JavaScript file
	jsCode := `/**
 * This is a module comment.
 */

export function publicFunction(param1, param2) {
    return 42;
}

function privateFunction() {
    return 0;
}

export class PublicClass {
    constructor() {
        this.value = 0;
    }
    
    method1(arg) {
        return arg;
    }
    
    async asyncMethod() {
        return Promise.resolve();
    }
}

class PrivateClass {
}

export const arrowFunc = (x, y) => x + y;

const privateArrow = () => {};

export { privateFunction as renamedExport };
`

	jsFile := filepath.Join(tmpDir, "test.js")
	err := os.WriteFile(jsFile, []byte(jsCode), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create analyzer
	analyzer := NewProjectAnalyzer(tmpDir, nil)

	// Analyze the JavaScript file
	structure, err := analyzer.AnalyzeJavaScriptFile("test.js")
	if err != nil {
		t.Fatalf("AnalyzeJavaScriptFile failed: %v", err)
	}

	// Verify package info
	if len(structure.Packages) != 1 {
		t.Errorf("Expected 1 package, got %d", len(structure.Packages))
	}
	if structure.Packages[0].Name != "test" {
		t.Errorf("Expected package name 'test', got '%s'", structure.Packages[0].Name)
	}

	// Verify functions (should include both regular and arrow functions)
	if len(structure.Functions) < 3 {
		t.Errorf("Expected at least 3 functions, got %d", len(structure.Functions))
	}

	// Check for publicFunction
	foundPublicFunc := false
	for _, fn := range structure.Functions {
		if fn.Name == "publicFunction" {
			foundPublicFunc = true
			if len(fn.Parameters) != 2 {
				t.Errorf("Expected 2 parameters, got %d", len(fn.Parameters))
			}
		}
	}
	if !foundPublicFunc {
		t.Error("publicFunction not found")
	}

	// Verify classes
	if len(structure.Classes) < 1 {
		t.Errorf("Expected at least 1 class, got %d", len(structure.Classes))
	}

	// Check PublicClass
	foundPublicClass := false
	for _, cls := range structure.Classes {
		if cls.Name == "PublicClass" {
			foundPublicClass = true
			if cls.IsExported != true {
				t.Error("PublicClass should be marked as exported")
			}
			if len(cls.Methods) < 1 {
				t.Errorf("Expected at least 1 method in PublicClass, got %d", len(cls.Methods))
			}
		}
	}
	if !foundPublicClass {
		t.Error("PublicClass not found")
	}

	// Verify exports
	if len(structure.Exports) < 3 {
		t.Errorf("Expected at least 3 exports, got %d", len(structure.Exports))
	}

	// Check for specific exports
	exportNames := make(map[string]bool)
	for _, exp := range structure.Exports {
		exportNames[exp.Name] = true
	}

	if !exportNames["publicFunction"] {
		t.Error("publicFunction should be in exports")
	}
	if !exportNames["PublicClass"] {
		t.Error("PublicClass should be in exports")
	}
	if !exportNames["arrowFunc"] {
		t.Error("arrowFunc should be in exports")
	}
}

func TestProjectAnalyzer_AnalyzePythonFile_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	pythonFile := filepath.Join(tmpDir, "empty.py")
	err := os.WriteFile(pythonFile, []byte(""), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	analyzer := NewProjectAnalyzer(tmpDir, nil)
	structure, err := analyzer.AnalyzePythonFile("empty.py")
	if err != nil {
		t.Fatalf("AnalyzePythonFile failed: %v", err)
	}

	if len(structure.Functions) != 0 {
		t.Errorf("Expected 0 functions in empty file, got %d", len(structure.Functions))
	}
	if len(structure.Classes) != 0 {
		t.Errorf("Expected 0 classes in empty file, got %d", len(structure.Classes))
	}
}

func TestProjectAnalyzer_AnalyzeJavaScriptFile_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	jsFile := filepath.Join(tmpDir, "empty.js")
	err := os.WriteFile(jsFile, []byte(""), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	analyzer := NewProjectAnalyzer(tmpDir, nil)
	structure, err := analyzer.AnalyzeJavaScriptFile("empty.js")
	if err != nil {
		t.Fatalf("AnalyzeJavaScriptFile failed: %v", err)
	}

	if len(structure.Functions) != 0 {
		t.Errorf("Expected 0 functions in empty file, got %d", len(structure.Functions))
	}
	if len(structure.Classes) != 0 {
		t.Errorf("Expected 0 classes in empty file, got %d", len(structure.Classes))
	}
}
