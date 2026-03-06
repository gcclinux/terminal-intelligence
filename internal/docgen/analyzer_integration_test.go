package docgen

import (
	"os"
	"path/filepath"
	"testing"
)

// TestIntegration_PythonAndJavaScriptAnalysis is an integration test
// that verifies Python and JavaScript analysis works end-to-end
func TestIntegration_PythonAndJavaScriptAnalysis(t *testing.T) {
	tmpDir := t.TempDir()

	// Create Python file
	pythonCode := `"""Module docstring."""

def public_func(x: int) -> str:
    """A public function."""
    return str(x)

class TestClass:
    """A test class."""
    def method(self):
        pass
`
	pythonFile := filepath.Join(tmpDir, "module.py")
	if err := os.WriteFile(pythonFile, []byte(pythonCode), 0644); err != nil {
		t.Fatal(err)
	}

	// Create JavaScript file
	jsCode := `/**
 * Module comment
 */

export function publicFunc(x) {
    return x * 2;
}

export class TestClass {
    method() {
        return 42;
    }
}
`
	jsFile := filepath.Join(tmpDir, "module.js")
	if err := os.WriteFile(jsFile, []byte(jsCode), 0644); err != nil {
		t.Fatal(err)
	}

	analyzer := NewProjectAnalyzer(tmpDir, nil)

	// Test Python
	pyStructure, err := analyzer.AnalyzePythonFile("module.py")
	if err != nil {
		t.Fatalf("Python analysis failed: %v", err)
	}

	if len(pyStructure.Functions) == 0 {
		t.Error("Expected Python functions to be extracted")
	}
	if len(pyStructure.Classes) == 0 {
		t.Error("Expected Python classes to be extracted")
	}
	if len(pyStructure.Exports) == 0 {
		t.Error("Expected Python exports to be extracted")
	}

	// Test JavaScript
	jsStructure, err := analyzer.AnalyzeJavaScriptFile("module.js")
	if err != nil {
		t.Fatalf("JavaScript analysis failed: %v", err)
	}

	if len(jsStructure.Functions) == 0 {
		t.Error("Expected JavaScript functions to be extracted")
	}
	if len(jsStructure.Classes) == 0 {
		t.Error("Expected JavaScript classes to be extracted")
	}
	if len(jsStructure.Exports) == 0 {
		t.Error("Expected JavaScript exports to be extracted")
	}
}
