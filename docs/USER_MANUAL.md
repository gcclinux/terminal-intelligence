# User Manual

## Aide — AI-Powered Code Assistant

**Version:** See `go.mod` for current module version
**Supported Platforms:** Linux, macOS, Windows

---

## Table of Contents

1. [Overview](#overview)
2. [Requirements](#requirements)
3. [Installation](#installation)
4. [Configuration](#configuration)
5. [Starting the Application](#starting-the-application)
6. [User Interface](#user-interface)
7. [Working with Files and Projects](#working-with-files-and-projects)
8. [AI Conversations](#ai-conversations)
9. [Keyboard Shortcuts](#keyboard-shortcuts)
10. [Git Integration](#git-integration)
11. [Clipboard Support](#clipboard-support)
12. [Running Tests](#running-tests)
13. [Building from Source](#building-from-source)
14. [Troubleshooting](#troubleshooting)

---

## Overview

Aide is a terminal-based AI coding assistant that runs directly in your terminal. It uses Amazon Bedrock as its AI backend and provides an interactive, keyboard-driven interface built with the Charmbracelet TUI framework (Bubbletea, Bubbles, and Lipgloss). Aide understands Go, JavaScript/TypeScript, and Python projects, reads your source files and dependencies, and lets you have contextual conversations about your code without leaving the terminal.

Key capabilities:

- **Contextual code chat** — Aide reads your project files and passes relevant context to the AI model hosted on Amazon Bedrock.
- **Multi-language support** — Recognises Go modules (`go.mod`), Node.js packages (`package.json`), and Python requirements (`requirements.txt`) automatically.
- **Git awareness** — Integrates with your local Git repository via `go-git` to understand history, diffs, and current branch state.
- **Clipboard integration** — Copy AI responses or code snippets directly to your system clipboard.
- **Fully keyboard-driven** — No mouse required; all navigation and actions are accessible via keyboard shortcuts.

---

## Requirements

| Requirement | Minimum Version / Notes |
|---|---|
| Go toolchain | 1.21 or later (check `go.mod` for the `go` directive) |
| AWS account | With Amazon Bedrock access enabled in your target region |
| AWS credentials | Configured via environment variables, `~/.aws/credentials`, SSO, or EC2 instance metadata |
| Terminal emulator | Any modern terminal with UTF-8 and 256-colour support |
| Git | Optional but recommended for Git-aware features |
| Node.js / npm | Only required for JavaScript/TypeScript project analysis |
| Python 3 | Only required for Python project analysis |

---

## Installation

### Option 1 — Install with `go install`

```bash
go install github.com/<org>/aide@latest
```

Replace `<org>` with the actual GitHub organisation or user shown in `go.mod`.

### Option 2 — Download a Pre-built Release Binary

Pre-built binaries for Linux (amd64/arm64), macOS (amd64/arm64), and Windows (amd64) are published automatically via the release workflow defined in `.github/workflows/release.yml`.

1. Visit the **Releases** page on GitHub.
2. Download the archive for your platform and architecture.
3. Extract the binary and place it on your `PATH`:

```bash
# Example for Linux amd64
tar -xzf aide_linux_amd64.tar.gz
sudo mv aide /usr/local/bin/
```

### Option 3 — Build from Source

See [Building from Source](#building-from-source).

---

## Configuration

Aide uses AWS credentials and region configuration to connect to Amazon Bedrock. No separate application-level config file is required; all configuration is handled through standard AWS mechanisms.

### AWS Credentials

Aide uses `github.com/aws/aws-sdk-go-v2/config` and `github.com/aws/aws-sdk-go-v2/credentials` to load credentials in the following order of precedence:

1. **Environment variables**

   ```bash
   export AWS_ACCESS_KEY_ID=AKIA...
   export AWS_SECRET_ACCESS_KEY=...
   export AWS_SESSION_TOKEN=...          # if using temporary credentials
   export AWS_REGION=us-east-1
   ```

2. **AWS credentials file** — `~/.aws/credentials` with a named profile:

   ```ini
   [default]
   aws_access_key_id     = AKIA...
   aws_secret_access_key = ...
   ```

   To use a non-default profile, set:

   ```bash
   export AWS_PROFILE=my-profile
   ```

3. **AWS SSO** — Configure an SSO profile in `~/.aws/config` and authenticate with `aws sso login --profile my-sso-profile` before launching Aide.

4. **EC2 Instance Metadata (IMDS)** — When running on an EC2 instance or ECS task with an IAM role attached, credentials are fetched automatically via the instance metadata service.

### Required IAM Permissions

Your IAM identity must have at minimum the following permissions:

```json
{
  "Effect": "Allow",
  "Action": [
    "bedrock:InvokeModel",
    "bedrock:InvokeModelWithResponseStream",
    "bedrock:ListFoundationModels"
  ],
  "Resource": "*"
}
```

### Selecting an AWS Region

Amazon Bedrock model availability varies by region. Set your target region explicitly if the default does not have the models you need:

```bash
export AWS_REGION=us-west-2
```

---

## Starting the Application

Navigate to your project directory and run:

```bash
aide
```

Aide will automatically detect the project type by looking for:

| File | Detected Project Type |
|---|---|
| `go.mod` | Go module |
| `package.json` | JavaScript / TypeScript (npm/yarn) |
| `requirements.txt` | Python |

If none of these files are present, Aide still starts but operates without dependency context.

### Starting in a Specific Directory

```bash
aide /path/to/your/project
```

---

## User Interface

Aide's terminal UI is built with [Bubbletea](https://github.com/charmbracelet/bubbletea) and [Lipgloss](https://github.com/charmbracelet/lipgloss). The interface is divided into three main areas:

```
┌─────────────────────────────────────────────────┐
│  Header — project name, model, current branch   │
├─────────────────────────────────────────────────┤
│                                                 │
│  Conversation area (scrollable)                 │
│                                                 │
│  You:  <your message>                           │
│  AI:   <model response, streamed in real time>  │
│                                                 │
│  ...                                            │
│                                                 │
├─────────────────────────────────────────────────┤
│  Input bar — type your message here             │
├─────────────────────────────────────────────────┤
│  Status bar — shortcuts hint / status messages  │
└─────────────────────────────────────────────────┘
```

### Session States

Aide moves through the following states during a session:

| State | Description |
|---|---|
| **Initialising** | Loading project context, connecting to AWS |
| **Ready** | Waiting for user input |
| **Thinking** | Request sent to Bedrock; awaiting first token |
| **Streaming** | Model response is being streamed to the conversation area |
| **Error** | An error occurred; see the status bar for details |

---

## Working with Files and Projects

### Go Projects

Aide reads `go.mod` to understand module path and declared dependencies. It uses this context to answer questions about package usage, dependency versions, and module structure. All dependencies listed in the project context (e.g., `github.com/charmbracelet/bubbletea@v1.3.10`) are passed to the model so it can give version-accurate advice.

### JavaScript / TypeScript Projects

Aide reads `package.json` to extract `dependencies` and `devDependencies`. It recognises test frameworks (e.g., `jest@^29.5.0`) and web frameworks (e.g., `express@^4.18.2`) and uses this information to tailor its suggestions.

### Python Projects

Aide reads `requirements.txt` to understand installed packages. It recognises packages such as `pytest==7.4.0` and `requests==2.31.0` and provides advice consistent with those versions.

### Sample Fixture Projects

The repository ships with three sample projects under `tests/fixtures/docgen/` that demonstrate how each language is handled:

| Path | Type |
|---|---|
| `tests/fixtures/docgen/sample-go-project/` | Go (has its own `go.mod`) |
| `tests/fixtures/docgen/sample-js-project/` | JavaScript/TypeScript |
| `tests/fixtures/docgen/sample-python-project/` | Python |

These fixtures are used by the test suite but also serve as reference examples of valid project layouts.

---

## AI Conversations

### Sending a Message

1. Ensure the input bar at the bottom of the screen has focus (it does by default).
2. Type your question or instruction.
3. Press **Enter** to send.

The AI response streams into the conversation area in real time, token by token, via the Amazon Bedrock streaming API (`InvokeModelWithResponseStream`).

### Conversation Tips

- **Be specific about files.** For example: *"In `internal/ui/model.go`, why does the update function handle the `WindowSizeMsg` event twice?"*
- **Reference dependency versions.** Aide already knows your dependency versions from your manifest files, so you can ask: *"Show me how to use the bubbles `textarea` component from v1.0.0."*
- **Ask for diffs.** When asking for code changes, ask for a unified diff so you can apply changes precisely.
- **Multi-turn context.** Aide maintains the full conversation history within a session. You can refer back to earlier messages naturally.

### Clearing the Conversation

Press **Ctrl+L** to clear the current conversation history and start a fresh session while keeping the project context loaded.

---

## Keyboard Shortcuts

| Shortcut | Action |
|---|---|
| **Enter** | Send the current message |
| **Ctrl+C** | Quit the application |
| **Ctrl+L** | Clear conversation history |
| **Ctrl+Y** | Copy the last AI response to the system clipboard |
| **↑ / ↓** | Scroll through conversation history |
| **Page Up / Page Down** | Scroll conversation by page |
| **Tab** | Move focus between input bar and conversation area |
| **Ctrl+U** | Clear the input bar |
| **Esc** | Cancel a streaming response in progress |

> **Note:** Clipboard operations use `github.com/atotto/clipboard` and are supported on Linux (with `xclip` or `xsel` installed), macOS, and Windows.

---

## Git Integration

Aide integrates with your local Git repository using `github.com/go-git/go-git/v5`. When you open a project that is a Git repository, Aide can:

- **Display the current branch name** in the header.
- **Read staged and unstaged diffs** so you can ask questions like: *"Review my staged changes and suggest improvements."*
- **Access commit history** to provide context about recent changes.

Git operations are read-only; Aide never modifies your repository.

### SSH and HTTPS Authentication

`go-git` supports both HTTPS and SSH remotes. SSH keys are loaded via `github.com/xanzy/ssh-agent` (using your system SSH agent) and known hosts are validated using `github.com/skeema/knownhosts`. HTTPS credentials follow your system's standard Git credential helpers.

---

## Clipboard Support

Clipboard functionality is provided by `github.com/atotto/clipboard@v0.1.4`.

### Platform Requirements

| Platform | Requirement |
|---|---|
| **macOS** | No additional software needed (`pbcopy`/`pbpaste` used automatically) |
| **Linux (X11)** | `xclip` or `xsel` must be installed |
| **Linux (Wayland)** | `wl-clipboard` must be installed |
| **Windows** | No additional software needed |

### Installing Clipboard Utilities on Linux

```bash
# Debian / Ubuntu
sudo apt install xclip

# Fedora / RHEL
sudo dnf install xclip

# Arch Linux
sudo pacman -S xclip
```

---

## Running Tests

### Unit Tests

The `tests/unit/` directory contains focused unit tests covering file type handling, keyboard shortcut registration, UI component initialisation, error message formatting, and session state transitions.

```bash
go test ./tests/unit/...
```

### All Tests

```bash
go test ./...
```

### With Verbose Output

```bash
go test -v ./tests/unit/...
```

### With Race Detection

```bash
go test -race ./...
```

### Property-Based Tests

Aide uses both `github.com/leanovate/gopter@v0.2.11` and `pgregory.net/rapid@v1.2.0` for property-based testing of core logic (e.g., state transitions, text processing). These tests run as part of the standard `go test` command.

### JavaScript Tests (Sample Project)

```bash
cd tests/fixtures/docgen/sample-js-project
npm install
npx jest
```

### Python Tests (Sample Project)

```bash
cd tests/fixtures/docgen/sample-python-project
pip install -r requirements.txt
pytest
```

---

## Building from Source

### Prerequisites

- Go 1.21 or later
- `make` (optional but recommended)
- Git

### Clone the Repository

```bash
git clone https://github.com/<org>/aide.git
cd aide
```

### Build Using Make

```bash
make build
```

The compiled binary is placed in the project root or a `dist/` directory as configured in the `Makefile`.

### Build Manually

```bash
go build -o aide .
```

### Cross-Compile

```bash
# Linux amd64
GOOS=linux GOARCH=amd64 go build -o aide-linux-amd64 .

# macOS arm64 (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -o aide-darwin-arm64 .

# Windows amd64
GOOS=windows GOARCH=amd64 go build -o aide-windows-amd64.exe .
```

### Release Builds

Releases are automated via `.github/workflows/release.yml`. To trigger a release, push a version tag:

```bash
git tag v1.2.3
git push origin v1.2.3
```

---

## Troubleshooting

### `NoCredentialProviders: no valid providers in chain`

AWS credentials are not configured. Check the [Configuration](#configuration) section and ensure at least one credential source is available. Verify with:

```bash
aws sts get-caller-identity
```

### `Error: operation error Bedrock: InvokeModelWithResponseStream, ... UnauthorizedException`

Your IAM identity does not have permission to invoke the requested Bedrock model. Review the [Required IAM Permissions](#required-iam-permissions) section and ensure Amazon Bedrock model access is enabled in the AWS console for your account and region.

### Garbled or Missing Characters in the Terminal

Aide uses Unicode box-drawing characters and emoji. Ensure your terminal emulator supports UTF-8. On Linux, verify:

```bash
echo $LANG
# Expected output contains UTF-8, e.g.: en_US.UTF-8
```

### Clipboard Not Working on Linux

Install `xclip` or `xsel` (X11) or `wl-clipboard` (Wayland). See [Clipboard Support](#clipboard-support).

### SSH Git Remote Authentication Fails

Ensure your SSH agent is running and your key is loaded:

```bash
eval "$(ssh-agent -s)"
ssh-add ~/.ssh/id_ed25519
```

### `go: module ... not found` During Build

Run `go mod download` to fetch all dependencies listed in `go.sum`:

```bash
go mod download
```

### UI Renders Incorrectly in a Narrow Terminal

Aide is designed for terminal widths of at least **80 columns**. Resize your terminal window or reduce font size to meet this minimum.

### Streaming Response Stops Unexpectedly

Press **Esc** to cancel the current request and retry. If the issue persists, check your network connection and whether the Bedrock service endpoint is reachable from your environment.

---

## Support and Contributing

- **Bug reports and feature requests:** Open an issue on GitHub.
- **