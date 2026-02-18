package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/user/terminal-intelligence/internal/types"
)

// JSONConfig represents the JSON config file schema.
type JSONConfig struct {
	Agent     string `json:"agent"`
	Model     string `json:"model"`
	GModel    string `json:"gmodel"`
	OllamaURL string `json:"ollama_url"`
	GeminiAPI string `json:"gemini_api"`
	Workspace string `json:"workspace"`
}

// LoadFromFile reads and parses a JSON config file at the given path.
func LoadFromFile(path string) (*JSONConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	return FromJSON(data)
}

// FromJSON deserializes a JSON byte slice into a JSONConfig.
func FromJSON(data []byte) (*JSONConfig, error) {
	var cfg JSONConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	return &cfg, nil
}

// ToJSON serializes a JSONConfig to a JSON byte slice with pretty formatting.
func ToJSON(cfg *JSONConfig) ([]byte, error) {
	return json.MarshalIndent(cfg, "", "  ")
}

// Validate checks that the JSONConfig has valid field values.
func Validate(cfg *JSONConfig) error {
	if cfg.Agent != "gemini" && cfg.Agent != "ollama" {
		return fmt.Errorf("invalid agent: must be \"gemini\" or \"ollama\"")
	}
	if cfg.Agent == "gemini" && cfg.GeminiAPI == "" {
		return fmt.Errorf("gemini_api is required when agent is \"gemini\"")
	}
	return nil
}

// ApplyToAppConfig merges a JSONConfig into an existing AppConfig,
// setting only the fields that are non-empty in the JSONConfig.
func ApplyToAppConfig(jcfg *JSONConfig, appCfg *types.AppConfig) {
	if jcfg.Agent != "" {
		appCfg.Provider = jcfg.Agent
	}
	// Use gmodel for gemini, model for ollama
	switch jcfg.Agent {
	case "gemini":
		if jcfg.GModel != "" {
			appCfg.DefaultModel = jcfg.GModel
		}
	default:
		if jcfg.Model != "" {
			appCfg.DefaultModel = jcfg.Model
		}
	}
	if jcfg.OllamaURL != "" {
		appCfg.OllamaURL = jcfg.OllamaURL
	}
	if jcfg.GeminiAPI != "" {
		appCfg.GeminiAPIKey = jcfg.GeminiAPI
	}
	if jcfg.Workspace != "" {
		appCfg.WorkspaceDir = jcfg.Workspace
	}
}

// AppConfigToJSONConfig converts an AppConfig into a JSONConfig for serialization.
func AppConfigToJSONConfig(appCfg *types.AppConfig) *JSONConfig {
	jcfg := &JSONConfig{
		Agent:     appCfg.Provider,
		OllamaURL: appCfg.OllamaURL,
		GeminiAPI: appCfg.GeminiAPIKey,
		Workspace: appCfg.WorkspaceDir,
	}
	if appCfg.Provider == "gemini" {
		jcfg.GModel = appCfg.DefaultModel
	} else {
		jcfg.Model = appCfg.DefaultModel
	}
	return jcfg
}

// ConfigFilePath returns the expected config file path in the user's home directory.
// Returns ~/.ti/config.json on Linux/macOS or %USERPROFILE%\.ti\config.json on Windows.
func ConfigFilePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}
	return filepath.Join(homeDir, ".ti", "config.json"), nil
}

// CreateDefaultConfig creates a default config.json file with example values.
// Returns the path where the file was created.
func CreateDefaultConfig() (string, error) {
	configPath, err := ConfigFilePath()
	if err != nil {
		return "", err
	}

	// Create the directory if it doesn't exist
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create config directory: %w", err)
	}

	// Create default config with example values
	homeDir, _ := os.UserHomeDir()
	defaultConfig := &JSONConfig{
		Agent:     "ollama",
		Model:     "llama2",
		GModel:    "gemini-3-pro-preview",
		OllamaURL: "http://localhost:11434",
		GeminiAPI: "",
		Workspace: filepath.Join(homeDir, "ti-workspace"),
	}

	// Marshal to pretty JSON
	data, err := json.MarshalIndent(defaultConfig, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal default config: %w", err)
	}

	// Write to file
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write config file: %w", err)
	}

	return configPath, nil
}
