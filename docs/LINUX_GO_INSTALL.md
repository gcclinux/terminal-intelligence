# Automatic Go Installation on Linux

Terminal Intelligence includes a fully automated Go installer for Linux that downloads and installs Go directly from golang.org, eliminating the need for distribution-specific package managers.

## How It Works

The Linux Go installer performs the following steps automatically:

### 1. Version Detection
Fetches the latest stable Go version from `https://go.dev/VERSION?m=text`

### 2. Architecture Detection
Automatically detects your system architecture:
- `amd64` (x86_64) - Most common
- `arm64` (aarch64) - ARM 64-bit (Raspberry Pi 4, Apple Silicon via Rosetta)
- `386` (i386) - 32-bit x86

### 3. Download
Downloads the appropriate Go binary from `https://go.dev/dl/`

Example: `go1.21.5.linux-amd64.tar.gz`

### 4. Installation
- Removes old Go installation from `/usr/local/go` (if exists)
- Extracts new Go to `/usr/local/go`
- Requires sudo password for system-wide installation

### 5. PATH Configuration
Automatically updates your shell configuration files:
- `~/.bashrc` - For Bash shell
- `~/.profile` - For login shells
- `~/.zshrc` - For Zsh shell (if exists)

Adds: `export PATH=$PATH:/usr/local/go/bin`

### 6. Verification
Verifies installation by running `/usr/local/go/bin/go version`

## Installation Process

### Step-by-Step

1. **Save a Go file** in Terminal Intelligence
2. **Dialog appears** if Go is not detected
3. **Press Y** to confirm installation
4. **Enter sudo password** when prompted
5. **Watch progress** in the AI pane:

```
Starting Go installation for Linux...

1. Fetching latest Go version...
   Latest version: go1.21.5

2. Detected architecture: amd64

3. Download URL: https://go.dev/dl/go1.21.5.linux-amd64.tar.gz

4. Downloading Go...
   Download complete!

5. Removing old Go installation (if exists)...
   No old installation found

6. Extracting Go to /usr/local...
   Extraction complete!

7. Updating PATH in shell configuration...
   Updated .bashrc
   Updated .profile

8. Cleaning up...
   Temporary files removed

9. Verifying installation...
   go version go1.21.5 linux/amd64

✓ Go installation complete!

IMPORTANT: Please restart Terminal Intelligence or run:
  source ~/.bashrc
to update your PATH.
```

## Requirements

### System Requirements
- **Linux distribution**: Any (Ubuntu, Debian, Fedora, Arch, CentOS, etc.)
- **Architecture**: amd64, arm64, or 386
- **sudo access**: Required for system-wide installation
- **Internet connection**: To download Go from golang.org
- **Disk space**: ~150MB for Go installation

### Pre-installed Tools
These are typically available on all Linux systems:
- `curl` or `wget` - For downloading (Go's http.Get is used)
- `tar` - For extraction
- `sudo` - For system-wide installation

## Installation Locations

### Go Binary
```
/usr/local/go/bin/go
```

### Go Root
```
/usr/local/go/
```

### Shell Configuration
```
~/.bashrc
~/.profile
~/.zshrc (if exists)
```

### Temporary Download
```
/tmp/go1.21.5.linux-amd64.tar.gz (removed after installation)
```

## Post-Installation

### Verify Installation

After installation, verify Go is available:

```bash
# Restart Terminal Intelligence or source your shell config
source ~/.bashrc

# Check Go version
go version

# Check Go environment
go env
```

### Set GOPATH (Optional)

If you want to customize your Go workspace:

```bash
# Add to ~/.bashrc
export GOPATH=$HOME/go
export PATH=$PATH:$GOPATH/bin
```

### Test Installation

Create a simple Go program:

```go
package main

import "fmt"

func main() {
    fmt.Println("Hello from Go!")
}
```

Run it:
```bash
go run hello.go
```

## Troubleshooting

### "sudo: command not found"

**Problem**: Your system doesn't have sudo installed.

**Solution**: Install as root or use your distribution's package manager:
```bash
# As root
su -
tar -C /usr/local -xzf /tmp/go*.tar.gz
```

### "Permission denied" when extracting

**Problem**: Insufficient permissions to write to `/usr/local`.

**Solution**: Ensure you enter the correct sudo password, or install to a user directory:
```bash
# Install to home directory instead
tar -C $HOME -xzf /tmp/go*.tar.gz
export PATH=$PATH:$HOME/go/bin
```

### "Failed to fetch Go version"

**Problem**: No internet connection or golang.org is unreachable.

**Solution**: 
- Check your internet connection
- Try again later
- Manual installation: Download from https://go.dev/dl/

### "Unsupported architecture"

**Problem**: Your CPU architecture is not supported (rare).

**Solution**: Check supported architectures at https://go.dev/dl/ and install manually.

### Go not found after installation

**Problem**: PATH not updated in current session.

**Solution**:
```bash
# Restart Terminal Intelligence, or
source ~/.bashrc

# Or add to current session
export PATH=$PATH:/usr/local/go/bin
```

### Old Go version still showing

**Problem**: Multiple Go installations or PATH priority.

**Solution**:
```bash
# Check which Go is being used
which go

# Should show: /usr/local/go/bin/go

# If not, check your PATH
echo $PATH

# Remove other Go installations or adjust PATH priority
```

## Security Considerations

### Download Verification
- Downloads directly from official golang.org
- Uses HTTPS for secure transfer
- No third-party repositories or mirrors

### Installation Safety
- Only modifies `/usr/local/go` (standard Go location)
- Only updates user's shell configuration files
- Requires explicit sudo password (not stored)
- No arbitrary code execution

### Permissions
- System-wide installation requires sudo
- User shell configs modified with user permissions
- No changes to system-wide shell configs

## Comparison with Package Managers

### TI Auto-Install vs apt/yum/dnf

| Feature | TI Auto-Install | Package Managers |
|---------|----------------|------------------|
| Latest Version | ✓ Always | ✗ Often outdated |
| All Distros | ✓ Yes | ✗ Distro-specific |
| No Dependencies | ✓ Yes | ✗ May have deps |
| Official Binary | ✓ Yes | ~ Varies |
| Auto PATH Setup | ✓ Yes | ~ Sometimes |
| User-Friendly | ✓ Very | ~ Varies |

### When to Use Package Managers

Consider using your distribution's package manager if:
- You need system-wide package management
- You want automatic security updates
- You prefer distribution-tested versions
- You're managing multiple systems with automation

### When to Use TI Auto-Install

Use TI's auto-installer when:
- You want the latest Go version
- You're on a distribution with outdated Go packages
- You want a simple, one-click installation
- You're new to Linux and want an easy setup

## Advanced Usage

### Custom Installation Location

To install Go to a custom location, you'll need to modify the installer or install manually:

```bash
# Download
wget https://go.dev/dl/go1.21.5.linux-amd64.tar.gz

# Extract to custom location
tar -C $HOME/custom -xzf go1.21.5.linux-amd64.tar.gz

# Update PATH
export PATH=$PATH:$HOME/custom/go/bin
```

### Multiple Go Versions

To manage multiple Go versions, consider using:
- **gvm** (Go Version Manager): https://github.com/moovweb/gvm
- **goenv**: https://github.com/syndbg/goenv
- **asdf**: https://asdf-vm.com/

### Uninstalling

To remove Go installed by TI:

```bash
# Remove Go installation
sudo rm -rf /usr/local/go

# Remove PATH entries from shell configs
# Edit ~/.bashrc, ~/.profile, ~/.zshrc and remove:
# export PATH=$PATH:/usr/local/go/bin
```

## Technical Implementation

### Download Method
Uses Go's built-in `net/http` package for downloading:
```go
resp, err := http.Get("https://go.dev/dl/go1.21.5.linux-amd64.tar.gz")
```

### Extraction Method
Uses Go's `archive/tar` and `compress/gzip` packages:
```go
gzr, _ := gzip.NewReader(file)
tr := tar.NewReader(gzr)
```

### Shell Config Update
Checks if PATH export already exists before appending:
```go
if !strings.Contains(content, "export PATH=$PATH:/usr/local/go/bin") {
    // Append to file
}
```

## Future Enhancements

Planned improvements:
- Version selection (install specific Go version)
- Offline installation support
- Checksum verification
- Progress bar for downloads
- Rollback on failure
- User-directory installation option (no sudo)

## See Also

- [Automatic Language Installation](AUTO_INSTALL.md) - Overview of auto-install feature
- [Go Language Support](GO_SUPPORT.md) - Using Go in Terminal Intelligence
- [Official Go Installation](https://go.dev/doc/install) - Manual installation guide

---

[← Back to Auto-Install Documentation](AUTO_INSTALL.md)
