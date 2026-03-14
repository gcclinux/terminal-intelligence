package agentic

import (
	"testing"
)

func TestDetectLanguage_GoHeavy(t *testing.T) {
	files := []string{"main.go", "handler.go", "utils.go", "script.py"}
	got := detectLanguage(files)
	if got != "go" {
		t.Errorf("detectLanguage() = %q, want %q", got, "go")
	}
}

func TestDetectLanguage_PythonHeavy(t *testing.T) {
	files := []string{"app.py", "models.py", "views.py", "main.go"}
	got := detectLanguage(files)
	if got != "python" {
		t.Errorf("detectLanguage() = %q, want %q", got, "python")
	}
}

func TestDetectLanguage_BashFiles(t *testing.T) {
	files := []string{"deploy.sh", "setup.bash", "run.sh"}
	got := detectLanguage(files)
	if got != "bash" {
		t.Errorf("detectLanguage() = %q, want %q", got, "bash")
	}
}

func TestDetectLanguage_PowerShell(t *testing.T) {
	files := []string{"install.ps1", "build.ps1", "test.ps1"}
	got := detectLanguage(files)
	if got != "powershell" {
		t.Errorf("detectLanguage() = %q, want %q", got, "powershell")
	}
}

func TestDetectLanguage_EmptyInput(t *testing.T) {
	got := detectLanguage([]string{})
	if got != "" {
		t.Errorf("detectLanguage() = %q, want empty string", got)
	}
}

func TestDetectLanguage_NoRecognizedExtensions(t *testing.T) {
	files := []string{"readme.md", "config.yaml", "data.json"}
	got := detectLanguage(files)
	if got != "" {
		t.Errorf("detectLanguage() = %q, want empty string", got)
	}
}

func TestDetectLanguage_CaseInsensitiveExtensions(t *testing.T) {
	files := []string{"Main.GO", "handler.Go", "utils.go"}
	got := detectLanguage(files)
	if got != "go" {
		t.Errorf("detectLanguage() = %q, want %q", got, "go")
	}
}

func TestGetTestCommand_Go(t *testing.T) {
	got := getTestCommand("go")
	want := "go test ./..."
	if got != want {
		t.Errorf("getTestCommand(\"go\") = %q, want %q", got, want)
	}
}

func TestGetTestCommand_Python(t *testing.T) {
	got := getTestCommand("python")
	want := "python -m pytest"
	if got != want {
		t.Errorf("getTestCommand(\"python\") = %q, want %q", got, want)
	}
}

func TestGetTestCommand_Bash(t *testing.T) {
	got := getTestCommand("bash")
	want := "shellcheck"
	if got != want {
		t.Errorf("getTestCommand(\"bash\") = %q, want %q", got, want)
	}
}

func TestGetTestCommand_PowerShell(t *testing.T) {
	got := getTestCommand("powershell")
	want := "Invoke-Pester"
	if got != want {
		t.Errorf("getTestCommand(\"powershell\") = %q, want %q", got, want)
	}
}

func TestGetTestCommand_Unknown(t *testing.T) {
	got := getTestCommand("rust")
	if got != "" {
		t.Errorf("getTestCommand(\"rust\") = %q, want empty string", got)
	}
}

func TestDefaultLanguageRegistry_AllEntriesPresent(t *testing.T) {
	expected := []string{"go", "python", "bash", "powershell"}
	for _, lang := range expected {
		cfg, ok := defaultLanguageRegistry[lang]
		if !ok {
			t.Errorf("defaultLanguageRegistry missing entry for %q", lang)
			continue
		}
		if cfg.Name == "" {
			t.Errorf("defaultLanguageRegistry[%q].Name is empty", lang)
		}
		if len(cfg.Extensions) == 0 {
			t.Errorf("defaultLanguageRegistry[%q].Extensions is empty", lang)
		}
		if cfg.TestCommand == "" {
			t.Errorf("defaultLanguageRegistry[%q].TestCommand is empty", lang)
		}
	}
}
