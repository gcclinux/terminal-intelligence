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
	Agent         string `json:"agent"`
	Model         string `json:"model"`
	GModel        string `json:"gmodel"`
	OllamaURL     string `json:"ollama_url"`
	GeminiAPI     string `json:"gemini_api"`
	BedrockAPI    string `json:"bedrock_api"`
	BedrockModel  string `json:"bedrock_model"`
	BedrockRegion string `json:"bedrock_region"`
	Workspace     string `json:"workspace"`
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
	if cfg.Agent != "gemini" && cfg.Agent != "ollama" && cfg.Agent != "bedrock" {
		return fmt.Errorf("invalid agent: must be \"gemini\", \"ollama\", or \"bedrock\"")
	}
	if cfg.Agent == "gemini" && cfg.GeminiAPI == "" {
		return fmt.Errorf("gemini_api is required when agent is \"gemini\"")
	}
	if cfg.Agent == "bedrock" && cfg.BedrockAPI == "" {
		return fmt.Errorf("bedrock_api is required when agent is \"bedrock\"")
	}
	return nil
}

// ApplyToAppConfig merges a JSONConfig into an existing AppConfig,
// setting only the fields that are non-empty in the JSONConfig.
func ApplyToAppConfig(jcfg *JSONConfig, appCfg *types.AppConfig) {
	if jcfg.Agent != "" {
		appCfg.Provider = jcfg.Agent
	}

	// Update stored models if present in config
	if jcfg.GModel != "" {
		appCfg.GeminiModel = jcfg.GModel
	}
	if jcfg.Model != "" {
		appCfg.OllamaModel = jcfg.Model
	}
	if jcfg.BedrockModel != "" {
		appCfg.BedrockModel = jcfg.BedrockModel
	}

	// Set active model based on provider
	if appCfg.Provider == "gemini" {
		if appCfg.GeminiModel != "" {
			appCfg.DefaultModel = appCfg.GeminiModel
		}
	} else if appCfg.Provider == "bedrock" {
		if appCfg.BedrockModel != "" {
			appCfg.DefaultModel = appCfg.BedrockModel
		}
	} else {
		// Default to ollama
		if appCfg.OllamaModel != "" {
			appCfg.DefaultModel = appCfg.OllamaModel
		}
	}

	if jcfg.OllamaURL != "" {
		appCfg.OllamaURL = jcfg.OllamaURL
	}
	if jcfg.GeminiAPI != "" {
		appCfg.GeminiAPIKey = jcfg.GeminiAPI
	}
	if jcfg.BedrockAPI != "" {
		appCfg.BedrockAPIKey = jcfg.BedrockAPI
	}
	if jcfg.BedrockRegion != "" {
		appCfg.BedrockRegion = jcfg.BedrockRegion
	} else if appCfg.Provider == "bedrock" && appCfg.BedrockRegion == "" {
		// Default to us-east-1 if bedrock is the provider and region is not set
		appCfg.BedrockRegion = "us-east-1"
	}
	if jcfg.Workspace != "" {
		appCfg.WorkspaceDir = jcfg.Workspace
	}
}

// AppConfigToJSONConfig converts an AppConfig into a JSONConfig for serialization.
func AppConfigToJSONConfig(appCfg *types.AppConfig) *JSONConfig {
	jcfg := &JSONConfig{
		Agent:         appCfg.Provider,
		OllamaURL:     appCfg.OllamaURL,
		GeminiAPI:     appCfg.GeminiAPIKey,
		BedrockAPI:    appCfg.BedrockAPIKey,
		BedrockRegion: appCfg.BedrockRegion,
		Workspace:     appCfg.WorkspaceDir,
		Model:         appCfg.OllamaModel,
		GModel:        appCfg.GeminiModel,
		BedrockModel:  appCfg.BedrockModel,
	}

	// Ensure active model is synced to the correct field if stored model is empty
	if appCfg.Provider == "gemini" {
		if jcfg.GModel == "" {
			jcfg.GModel = appCfg.DefaultModel
		}
	} else if appCfg.Provider == "bedrock" {
		if jcfg.BedrockModel == "" {
			jcfg.BedrockModel = appCfg.DefaultModel
		}
	} else {
		if jcfg.Model == "" {
			jcfg.Model = appCfg.DefaultModel
		}
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
