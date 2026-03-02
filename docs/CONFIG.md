# Configuration Guide

Terminal Intelligence supports configuration through a JSON file located at `~/.ti/config.json` (or `%USERPROFILE%\.ti\config.json` on Windows).

## Configuration File

On first run, if no config file exists, the application will automatically create a default `config.json` with example values at `~/.ti/config.json`.

You can edit this file to customize your settings. The application will load it automatically on subsequent runs.

## Configuration Examples

### Ollama Configuration (Default)

```json
{
  "agent": "ollama",
  "model": "qwen2.5-coder:3b",
  "ollama_url": "http://localhost:11434",
  "gemini_api": "",
  "gemini_model": "",
  "bedrock_api": "",
  "bedrock_model": "",
  "bedrock_region": "",
  "workspace": "/home/user/ti-workspace"
}
```

### Gemini Configuration

```json
{
  "agent": "gemini",
  "model": "",
  "ollama_url": "",
  "gemini_api": "your-gemini-api-key-here",
  "gemini_model": "gemini-3.1-flash-lite",
  "bedrock_api": "",
  "bedrock_model": "",
  "bedrock_region": "",
  "workspace": "/home/user/project-workspace"
}
```

### AWS Bedrock Configuration

```json
{
  "agent": "bedrock",
  "model": "",
  "ollama_url": "",
  "gemini_api": "",
  "gemini_model": "",
  "bedrock_api": "ACCESS_KEY_ID:SECRET_ACCESS_KEY",
  "bedrock_model": "anthropic.claude-sonnet-4-6",
  "bedrock_region": "us-east-1",
  "workspace": "/home/user/project-workspace"
}
```

## Configuration Fields

### Core Settings

- **`agent`** (string, required): AI provider to use
  - Valid values: `"ollama"`, `"gemini"`, `"bedrock"`
  - Determines which AI service will handle requests

- **`workspace`** (string, required): Workspace directory path
  - Absolute path to your workspace folder
  - Example: `/home/user/ti-workspace` or `C:\Users\user\ti-workspace`

### Ollama Settings

- **`ollama_url`** (string): Ollama server URL
  - Default: `http://localhost:11434`
  - Only used when `agent` is set to `"ollama"`
  - Change if running Ollama on a different host/port

- **`model`** (string): Ollama model name
  - Examples: `"qwen2.5-coder:3b"`, `"deepseek-coder-v2:16b"`, `"llama2"`
  - Only used when `agent` is set to `"ollama"`

### Gemini Settings

- **`gemini_api`** (string): Google Gemini API key
  - Required when `agent` is set to `"gemini"`
  - Get your API key from [Google AI Studio](https://makersuite.google.com/app/apikey)
  - Format: `"AIza..."`

- **`gemini_model`** (string): Gemini model name
  - Examples: `"gemini-3.1-flash-lite"`, `"gemini-3.1-pro-preview"`, `"gemini-3-flash-preview"`
  - Only used when `agent` is set to `"gemini"`

### AWS Bedrock Settings

- **`bedrock_api`** (string): AWS credentials
  - Required when `agent` is set to `"bedrock"`
  - Format: `"ACCESS_KEY_ID:SECRET_ACCESS_KEY"`
  - Example: `"AKIAIOSFODNN7EXAMPLE:wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"`
  - Get credentials from [AWS IAM Console](https://console.aws.amazon.com/iam/)

- **`bedrock_model`** (string): Bedrock model identifier
  - Examples: `"anthropic.claude-sonnet-4-6"`, `"anthropic.claude-haiku-4-6"`, `"anthropic.claude-opus-4-6"`
  - Only used when `agent` is set to `"bedrock"`
  - Models automatically use inference profiles for optimal performance

- **`bedrock_region`** (string): AWS region
  - Default: `"us-east-1"`
  - Examples: `"us-east-1"`, `"us-west-2"`, `"eu-west-1"`
  - Only used when `agent` is set to `"bedrock"`
  - Choose a region where Bedrock is available

## Tested LLMs

Terminal Intelligence has been tested and verified to work with the following AI models:

| Provider | Model | Notes |
|----------|-------|-------|
| **Ollama** | `qwen2.5-coder:3b` | Recommended for coding tasks |
| | `qwen2.5-coder:1.5b` | Lightweight coding model |
| | `deepseek-coder-v2:16b` | Advanced coding capabilities |
| **Gemini** | `gemini-3-flash-preview` | Fast responses |
| | `gemini-3.1-pro-preview` | Advanced reasoning |
| | `gemini-3.1-flash-lite` | Lightweight and fast |
| **AWS Bedrock** | `anthropic.claude-sonnet-4-6` | Best coding performance |
| | `anthropic.claude-haiku-4-6` | Fast and cost-effective |
| | `anthropic.claude-opus-4-6` | Highest intelligence |

**Note:** AWS Bedrock models automatically use inference profiles for optimal performance and cross-region failover.

## Command-Line Overrides

Configuration file values can be overridden using command-line flags:

```bash
# Override agent and model
ti --agent gemini --model gemini-3.1-flash-lite

# Override workspace
ti --workspace /path/to/workspace

# Override Ollama URL
ti --ollama-url http://remote-host:11434

# Override Gemini API key
ti --gemini-api YOUR_API_KEY

# Override Bedrock settings
ti --bedrock-api ACCESS_KEY:SECRET_KEY --bedrock-region us-west-2
```

**Note:** Command-line flags take precedence over config file values.

## Configuration Best Practices

### Security

1. **Protect API Keys**: Never commit `config.json` with real API keys to version control
2. **Use Environment Variables**: Consider using environment variables for sensitive credentials
3. **File Permissions**: Ensure `~/.ti/config.json` has restricted permissions (chmod 600 on Unix)

### Performance

1. **Choose Appropriate Models**: 
   - Use smaller models (1.5b-3b) for faster responses
   - Use larger models (16b+) for complex coding tasks
2. **Local vs Cloud**: 
   - Ollama: No latency, runs locally, requires local resources
   - Gemini/Bedrock: Cloud-based, requires internet, no local resources needed

### Workspace Organization

1. **Dedicated Workspace**: Use a dedicated directory for TI projects
2. **Absolute Paths**: Always use absolute paths for the workspace setting
3. **Backup**: Regularly backup your workspace directory

## Troubleshooting

### Ollama Connection Issues

```
Error: Failed to connect to Ollama
```

**Solutions:**
- Verify Ollama is running: `ollama list`
- Check the `ollama_url` in config matches your Ollama server
- Ensure firewall allows connections to Ollama port

### Gemini API Issues

```
Error: Invalid Gemini API key
```

**Solutions:**
- Verify your API key is correct in `gemini_api` field
- Check API key has not expired
- Ensure you have API quota remaining

### Bedrock Authentication Issues

```
Error: Bedrock Generate (status 403): Invalid credentials
```

**Solutions:**
- Verify AWS credentials format: `ACCESS_KEY_ID:SECRET_ACCESS_KEY`
- Check IAM permissions include `bedrock:InvokeModel`
- Ensure the model is available in your selected region
- Verify your AWS account has Bedrock access enabled

### Bedrock Model Not Found

```
Error: Bedrock Generate (status 404): Model may be invalid
```

**Solutions:**
- Check model ID is correct (e.g., `anthropic.claude-sonnet-4-6`)
- Verify model is available in your region
- Request model access in AWS Bedrock console if needed

## Interactive Configuration Editor

You can also edit configuration interactively from within TI:

1. Press `Ctrl+C` to open the chat panel
2. Type `/config` and press Enter
3. Use the visual editor to modify settings
4. Changes are saved automatically to `~/.ti/config.json`

This provides a user-friendly way to update configuration without manually editing JSON files.
