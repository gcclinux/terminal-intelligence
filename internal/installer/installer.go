package installer

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// LanguageInstaller handles checking and installing language runtimes
type LanguageInstaller struct {
	ProgressCallback func(message string) // Optional callback for progress updates
}

// NewLanguageInstaller creates a new language installer
func NewLanguageInstaller() *LanguageInstaller {
	return &LanguageInstaller{}
}

// SetProgressCallback sets a callback function for progress updates
func (li *LanguageInstaller) SetProgressCallback(callback func(message string)) {
	li.ProgressCallback = callback
}

// reportProgress sends a progress update if callback is set
func (li *LanguageInstaller) reportProgress(message string) {
	if li.ProgressCallback != nil {
		li.ProgressCallback(message)
	}
}

// IsGoInstalled checks if Go is installed and available in PATH
func (li *LanguageInstaller) IsGoInstalled() bool {
	cmd := exec.Command("go", "version")
	err := cmd.Run()
	return err == nil
}

// GetGoVersion returns the installed Go version or empty string if not installed
func (li *LanguageInstaller) GetGoVersion() string {
	cmd := exec.Command("go", "version")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

// GetGoInstallCommand returns the appropriate command to install Go based on OS
func (li *LanguageInstaller) GetGoInstallCommand() (string, string, error) {
	switch runtime.GOOS {
	case "windows":
		// Use winget on Windows 10/11
		return "winget", "winget install -e --id GoLang.Go", nil
	case "darwin":
		// Use Homebrew on macOS
		return "brew", "brew install go", nil
	case "linux":
		// Use direct download and install from golang.org
		return "direct", "Download and install from https://go.dev/dl/", nil
	default:
		return "", "", fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

// GetLatestGoVersion fetches the latest Go version from golang.org
func (li *LanguageInstaller) GetLatestGoVersion() (string, error) {
	resp, err := http.Get("https://go.dev/VERSION?m=text")
	if err != nil {
		return "", fmt.Errorf("failed to fetch Go version: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch Go version: HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read version response: %w", err)
	}

	version := strings.TrimSpace(string(body))
	lines := strings.Split(version, "\n")
	if len(lines) > 0 {
		return strings.TrimSpace(lines[0]), nil
	}

	return "", fmt.Errorf("invalid version response")
}

// downloadFile downloads a file from URL to the specified destination
func (li *LanguageInstaller) downloadFile(url, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}

	out, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to save file: %w", err)
	}

	return nil
}

// extractTarGz extracts a .tar.gz file to the specified destination
func (li *LanguageInstaller) extractTarGz(src, dest string) error {
	file, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open archive: %w", err)
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar: %w", err)
		}

		target := filepath.Join(dest, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
		case tar.TypeReg:
			outFile, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("failed to create file: %w", err)
			}
			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return fmt.Errorf("failed to write file: %w", err)
			}
			outFile.Close()
		}
	}

	return nil
}

// InstallGoLinux installs Go on Linux by downloading from golang.org
// InstallGoLinux installs Go on Linux by downloading from golang.org
func (li *LanguageInstaller) InstallGoLinux() (string, error) {
	var output strings.Builder
	output.WriteString("Starting Go installation for Linux...\n\n")
	li.reportProgress("🚀 Starting Go installation for Linux...")

	// Get latest version
	output.WriteString("1. Fetching latest Go version...\n")
	li.reportProgress("📡 Fetching latest Go version...")
	version, err := li.GetLatestGoVersion()
	if err != nil {
		return output.String(), fmt.Errorf("failed to get Go version: %w", err)
	}
	output.WriteString(fmt.Sprintf("   Latest version: %s\n\n", version))
	li.reportProgress(fmt.Sprintf("✓ Latest version: %s", version))

	// Determine architecture
	arch := runtime.GOARCH
	if arch != "amd64" && arch != "arm64" && arch != "386" {
		return output.String(), fmt.Errorf("unsupported architecture: %s", arch)
	}
	output.WriteString(fmt.Sprintf("2. Detected architecture: %s\n\n", arch))
	li.reportProgress(fmt.Sprintf("💻 Detected architecture: %s", arch))

	// Build download URL
	filename := fmt.Sprintf("%s.linux-%s.tar.gz", version, arch)
	url := fmt.Sprintf("https://go.dev/dl/%s", filename)
	output.WriteString(fmt.Sprintf("3. Download URL: %s\n\n", url))

	// Create temp directory
	tmpDir := os.TempDir()
	downloadPath := filepath.Join(tmpDir, filename)

	// Download
	output.WriteString("4. Downloading Go...\n")
	li.reportProgress("⬇️  Downloading Go (~140MB)... This may take a minute...")
	if err := li.downloadFile(url, downloadPath); err != nil {
		return output.String(), fmt.Errorf("download failed: %w", err)
	}
	output.WriteString("   Download complete!\n\n")
	li.reportProgress("✓ Download complete!")

	// Remove old installation if exists
	output.WriteString("5. Removing old Go installation (if exists)...\n")
	li.reportProgress("🗑️  Removing old Go installation...")
	oldGoPath := "/usr/local/go"
	if _, err := os.Stat(oldGoPath); err == nil {
		cmd := exec.Command("sudo", "rm", "-rf", oldGoPath)
		if err := cmd.Run(); err != nil {
			output.WriteString(fmt.Sprintf("   Warning: Could not remove old installation: %v\n", err))
			li.reportProgress(fmt.Sprintf("⚠️  Warning: Could not remove old installation"))
		} else {
			output.WriteString("   Old installation removed\n")
			li.reportProgress("✓ Old installation removed")
		}
	} else {
		output.WriteString("   No old installation found\n")
		li.reportProgress("✓ No old installation found")
	}
	output.WriteString("\n")

	// Extract with sudo
	output.WriteString("6. Extracting Go to /usr/local...\n")
	li.reportProgress("📦 Extracting Go to /usr/local (requires sudo)...")
	cmd := exec.Command("sudo", "tar", "-C", "/usr/local", "-xzf", downloadPath)
	if cmdOutput, err := cmd.CombinedOutput(); err != nil {
		return output.String() + string(cmdOutput), fmt.Errorf("extraction failed: %w", err)
	}
	output.WriteString("   Extraction complete!\n\n")
	li.reportProgress("✓ Extraction complete!")

	// Update PATH in shell config files
	output.WriteString("7. Updating PATH in shell configuration...\n")
	li.reportProgress("⚙️  Updating PATH in shell configuration...")
	homeDir, err := os.UserHomeDir()
	if err != nil {
		output.WriteString(fmt.Sprintf("   Warning: Could not get home directory: %v\n", err))
	} else {
		pathLine := "export PATH=$PATH:/usr/local/go/bin"

		// Update .bashrc
		bashrcPath := filepath.Join(homeDir, ".bashrc")
		if err := li.appendToFileIfNotExists(bashrcPath, pathLine); err != nil {
			output.WriteString(fmt.Sprintf("   Warning: Could not update .bashrc: %v\n", err))
		} else {
			output.WriteString("   Updated .bashrc\n")
			li.reportProgress("✓ Updated .bashrc")
		}

		// Update .profile
		profilePath := filepath.Join(homeDir, ".profile")
		if err := li.appendToFileIfNotExists(profilePath, pathLine); err != nil {
			output.WriteString(fmt.Sprintf("   Warning: Could not update .profile: %v\n", err))
		} else {
			output.WriteString("   Updated .profile\n")
			li.reportProgress("✓ Updated .profile")
		}

		// Update .zshrc if it exists
		zshrcPath := filepath.Join(homeDir, ".zshrc")
		if _, err := os.Stat(zshrcPath); err == nil {
			if err := li.appendToFileIfNotExists(zshrcPath, pathLine); err != nil {
				output.WriteString(fmt.Sprintf("   Warning: Could not update .zshrc: %v\n", err))
			} else {
				output.WriteString("   Updated .zshrc\n")
				li.reportProgress("✓ Updated .zshrc")
			}
		}
	}
	output.WriteString("\n")

	// Clean up
	output.WriteString("8. Cleaning up...\n")
	li.reportProgress("🧹 Cleaning up temporary files...")
	os.Remove(downloadPath)
	output.WriteString("   Temporary files removed\n\n")
	li.reportProgress("✓ Temporary files removed")

	// Verify installation
	output.WriteString("9. Verifying installation...\n")
	li.reportProgress("🔍 Verifying installation...")
	cmd = exec.Command("/usr/local/go/bin/go", "version")
	if cmdOutput, err := cmd.Output(); err != nil {
		output.WriteString(fmt.Sprintf("   Warning: Could not verify: %v\n", err))
		li.reportProgress(fmt.Sprintf("⚠️  Warning: Could not verify installation"))
	} else {
		versionOutput := strings.TrimSpace(string(cmdOutput))
		output.WriteString(fmt.Sprintf("   %s\n", versionOutput))
		li.reportProgress(fmt.Sprintf("✓ %s", versionOutput))
	}
	output.WriteString("\n")

	output.WriteString("✓ Go installation complete!\n")
	li.reportProgress("🎉 Go installation complete!")
	output.WriteString("\nIMPORTANT: Please restart Terminal Intelligence or run:\n")
	output.WriteString("  source ~/.bashrc\n")
	output.WriteString("to update your PATH.\n")
	li.reportProgress("⚠️  IMPORTANT: Restart Terminal Intelligence to update PATH")

	return output.String(), nil
}

// appendToFileIfNotExists appends a line to a file if it doesn't already contain it
func (li *LanguageInstaller) appendToFileIfNotExists(filepath, line string) error {
	// Read existing content
	content, err := os.ReadFile(filepath)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, create it
			return os.WriteFile(filepath, []byte(line+"\n"), 0644)
		}
		return err
	}

	// Check if line already exists
	if strings.Contains(string(content), line) {
		return nil // Already exists
	}

	// Append the line
	f, err := os.OpenFile(filepath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString("\n" + line + "\n")
	return err
}

// InstallGo attempts to install Go using the appropriate method for the OS
func (li *LanguageInstaller) InstallGo() (string, error) {
	switch runtime.GOOS {
	case "linux":
		// Use direct download and install for Linux
		return li.InstallGoLinux()
	case "windows":
		return li.InstallGoWindows()
	case "darwin":
		return li.InstallGoDarwin()
	default:
		return "", fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

// InstallGoWindows installs Go on Windows using winget
func (li *LanguageInstaller) InstallGoWindows() (string, error) {
	var output strings.Builder
	output.WriteString("Starting Go installation for Windows...\n\n")

	// Check if winget is available
	output.WriteString("1. Checking for winget...\n")
	checkCmd := exec.Command("winget", "--version")
	if err := checkCmd.Run(); err != nil {
		return output.String(), fmt.Errorf("winget is not installed or not in PATH")
	}
	output.WriteString("   winget is available\n\n")

	// Install Go
	output.WriteString("2. Installing Go via winget...\n")
	cmd := exec.Command("powershell", "-NoProfile", "-Command", "winget install -e --id GoLang.Go")
	cmdOutput, err := cmd.CombinedOutput()
	output.WriteString(string(cmdOutput))

	if err != nil {
		return output.String(), fmt.Errorf("installation failed: %w", err)
	}

	output.WriteString("\n✓ Go installation complete!\n")
	output.WriteString("\nIMPORTANT: Please restart Terminal Intelligence to update your PATH.\n")

	return output.String(), nil
}

// InstallGoDarwin installs Go on macOS using Homebrew
func (li *LanguageInstaller) InstallGoDarwin() (string, error) {
	var output strings.Builder
	output.WriteString("Starting Go installation for macOS...\n\n")

	// Check if brew is available
	output.WriteString("1. Checking for Homebrew...\n")
	checkCmd := exec.Command("brew", "--version")
	if err := checkCmd.Run(); err != nil {
		return output.String(), fmt.Errorf("Homebrew is not installed. Install from: https://brew.sh")
	}
	output.WriteString("   Homebrew is available\n\n")

	// Update brew
	output.WriteString("2. Updating Homebrew...\n")
	updateCmd := exec.Command("brew", "update")
	if _, err := updateCmd.CombinedOutput(); err != nil {
		output.WriteString(fmt.Sprintf("   Warning: brew update failed: %v\n", err))
	} else {
		output.WriteString("   Homebrew updated\n")
	}
	output.WriteString("\n")

	// Install Go
	output.WriteString("3. Installing Go via Homebrew...\n")
	cmd := exec.Command("brew", "install", "go")
	cmdOutput, err := cmd.CombinedOutput()
	output.WriteString(string(cmdOutput))

	if err != nil {
		return output.String(), fmt.Errorf("installation failed: %w", err)
	}

	output.WriteString("\n✓ Go installation complete!\n")
	output.WriteString("\nGo should now be available in your PATH.\n")

	return output.String(), nil
}

// IsPythonInstalled checks if Python is installed and available in PATH
func (li *LanguageInstaller) IsPythonInstalled() bool {
	// Try python3 first
	cmd := exec.Command("python3", "--version")
	if err := cmd.Run(); err == nil {
		return true
	}

	// Try python
	cmd = exec.Command("python", "--version")
	return cmd.Run() == nil
}

// GetPythonVersion returns the installed Python version or empty string if not installed
func (li *LanguageInstaller) GetPythonVersion() string {
	// Try python3 first
	cmd := exec.Command("python3", "--version")
	output, err := cmd.Output()
	if err == nil {
		return strings.TrimSpace(string(output))
	}

	// Try python
	cmd = exec.Command("python", "--version")
	output, err = cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

// CheckLanguageForFile checks if the required language runtime is installed for a file type
func (li *LanguageInstaller) CheckLanguageForFile(fileType string) (bool, string) {
	switch fileType {
	case "go":
		if li.IsGoInstalled() {
			return true, li.GetGoVersion()
		}
		return false, "Go"
	case "python":
		if li.IsPythonInstalled() {
			return true, li.GetPythonVersion()
		}
		return false, "Python"
	default:
		// For bash, powershell, etc., assume they're available
		return true, ""
	}
}
