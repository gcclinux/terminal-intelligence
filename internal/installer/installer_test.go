package installer

import (
	"runtime"
	"testing"
)

func TestIsGoInstalled(t *testing.T) {
	installer := NewLanguageInstaller()

	// This test will pass if Go is installed (which it should be to run tests)
	installed := installer.IsGoInstalled()

	if !installed {
		t.Skip("Go is not installed, skipping test")
	}

	// If we got here, Go is installed
	version := installer.GetGoVersion()
	if version == "" {
		t.Error("Go is installed but version is empty")
	}

	t.Logf("Go is installed: %s", version)
}

func TestGetGoInstallCommand(t *testing.T) {
	installer := NewLanguageInstaller()

	packageManager, command, err := installer.GetGoInstallCommand()

	if err != nil {
		t.Fatalf("GetGoInstallCommand failed: %v", err)
	}

	// Verify we get appropriate commands for each platform
	switch runtime.GOOS {
	case "windows":
		if packageManager != "winget" {
			t.Errorf("Expected winget on Windows, got %s", packageManager)
		}
		if command != "winget install -e --id GoLang.Go" {
			t.Errorf("Unexpected command: %s", command)
		}
	case "darwin":
		if packageManager != "brew" {
			t.Errorf("Expected brew on macOS, got %s", packageManager)
		}
		if command != "brew install go" {
			t.Errorf("Unexpected command: %s", command)
		}
	case "linux":
		if packageManager != "direct" {
			t.Errorf("Expected direct on Linux, got %s", packageManager)
		}
	default:
		t.Logf("Unknown OS: %s", runtime.GOOS)
	}

	t.Logf("Platform: %s, Package Manager: %s, Command: %s", runtime.GOOS, packageManager, command)
}

func TestCheckLanguageForFile(t *testing.T) {
	installer := NewLanguageInstaller()

	tests := []struct {
		fileType     string
		expectCheck  bool
		languageName string
	}{
		{"go", true, "Go"},
		{"python", true, "Python"},
		{"bash", true, ""}, // bash is assumed available
		{"powershell", true, ""},
		{"markdown", true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.fileType, func(t *testing.T) {
			installed, langName := installer.CheckLanguageForFile(tt.fileType)

			if tt.fileType == "go" {
				// Go should be installed (we're running tests with it)
				if !installed {
					t.Errorf("Go should be detected as installed")
				}
				if langName == "" {
					t.Errorf("Go version should not be empty")
				}
			}

			if tt.languageName != "" && langName != tt.languageName {
				t.Logf("Expected language name %s, got %s", tt.languageName, langName)
			}

			t.Logf("File type: %s, Installed: %v, Language: %s", tt.fileType, installed, langName)
		})
	}
}

func TestIsPythonInstalled(t *testing.T) {
	installer := NewLanguageInstaller()

	installed := installer.IsPythonInstalled()

	if installed {
		version := installer.GetPythonVersion()
		t.Logf("Python is installed: %s", version)
	} else {
		t.Log("Python is not installed (this is OK)")
	}
}
