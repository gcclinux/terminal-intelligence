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
	GenAIType string `json:"genai_type"`
	ModelID   string `json:"model_id"`
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

// ToJSON serializes a JSONConfig to a JSON byte slice.
func ToJSON(cfg *JSONConfig) ([]byte, error) {
	return json.Marshal(cfg)
}

// Validate checks that the JSONConfig has valid field values.
func Validate(cfg *JSONConfig) error {
	if cfg.GenAIType != "gemini" && cfg.GenAIType != "ollama" {
		return fmt.Errorf("invalid genai_type: must be \"gemini\" or \"ollama\"")
	}
	if cfg.GenAIType == "gemini" && cfg.GeminiAPI == "" {
		return fmt.Errorf("gemini_api is required when genai_type is \"gemini\"")
	}
	return nil
}

// ApplyToAppConfig merges a JSONConfig into an existing AppConfig,
// setting only the fields that are non-empty in the JSONConfig.
func ApplyToAppConfig(jcfg *JSONConfig, appCfg *types.AppConfig) {
	if jcfg.GenAIType != "" {
		appCfg.Provider = jcfg.GenAIType
	}
	if jcfg.ModelID != "" {
		appCfg.DefaultModel = jcfg.ModelID
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
	return &JSONConfig{
		GenAIType: appCfg.Provider,
		ModelID:   appCfg.DefaultModel,
		OllamaURL: appCfg.OllamaURL,
		GeminiAPI: appCfg.GeminiAPIKey,
		Workspace: appCfg.WorkspaceDir,
	}
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
		GenAIType: "ollama",
		ModelID:   "llama2",
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
