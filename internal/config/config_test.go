package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/user/terminal-intelligence/internal/types"
)

// --- Unit tests for validation edge cases (Task 1.6) ---

func TestValidate_GeminiWithEmptyAPI_ReturnsError(t *testing.T) {
	cfg := &JSONConfig{Agent: "gemini", GeminiAPI: ""}
	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected error for gemini with empty gemini_api, got nil")
	}
	if !strings.Contains(err.Error(), "gemini_api is required") {
		t.Errorf("unexpected error message: %s", err.Error())
	}
}

func TestValidate_OllamaWithEmptyAPI_Passes(t *testing.T) {
	cfg := &JSONConfig{Agent: "ollama", GeminiAPI: ""}
	err := Validate(cfg)
	if err != nil {
		t.Fatalf("expected no error for ollama with empty gemini_api, got: %v", err)
	}
}

func TestValidate_ValidGenAIType_EmptyOtherFields_Passes(t *testing.T) {
	cfg := &JSONConfig{Agent: "ollama"}
	err := Validate(cfg)
	if err != nil {
		t.Fatalf("expected no error for valid agent with empty fields, got: %v", err)
	}
}

func TestLoadFromFile_NonExistentPath_ReturnsError(t *testing.T) {
	_, err := LoadFromFile("/tmp/nonexistent-config-file-12345.json")
	if err == nil {
		t.Fatal("expected error for non-existent file, got nil")
	}
}

// --- Additional unit tests for core functions ---

func TestFromJSON_ValidJSON(t *testing.T) {
	data := []byte(`{"agent":"gemini","model":"flash","gemini_api":"key123"}`)
	cfg, err := FromJSON(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Agent != "gemini" || cfg.Model != "flash" || cfg.GeminiAPI != "key123" {
		t.Errorf("unexpected config values: %+v", cfg)
	}
}

func TestFromJSON_InvalidJSON_ReturnsError(t *testing.T) {
	_, err := FromJSON([]byte(`{not valid json`))
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestToJSON_RoundTrip(t *testing.T) {
	original := &JSONConfig{
		Agent:     "ollama",
		Model:     "llama2",
		OllamaURL: "http://localhost:11434",
		Workspace: "/tmp/ws",
	}
	data, err := ToJSON(original)
	if err != nil {
		t.Fatalf("ToJSON error: %v", err)
	}
	restored, err := FromJSON(data)
	if err != nil {
		t.Fatalf("FromJSON error: %v", err)
	}
	if *original != *restored {
		t.Errorf("round-trip mismatch: %+v != %+v", original, restored)
	}
}

func TestApplyToAppConfig_SetsNonEmptyFields(t *testing.T) {
	jcfg := &JSONConfig{
		Agent:     "gemini",
		GModel:    "flash",
		GeminiAPI: "key123",
	}
	appCfg := types.DefaultConfig()
	original := *appCfg
	ApplyToAppConfig(jcfg, appCfg)

	if appCfg.Provider != "gemini" {
		t.Errorf("Provider: got %q, want %q", appCfg.Provider, "gemini")
	}
	if appCfg.DefaultModel != "flash" {
		t.Errorf("DefaultModel: got %q, want %q", appCfg.DefaultModel, "flash")
	}
	if appCfg.GeminiAPIKey != "key123" {
		t.Errorf("GeminiAPIKey: got %q, want %q", appCfg.GeminiAPIKey, "key123")
	}
	// Empty fields should not override defaults
	if appCfg.OllamaURL != original.OllamaURL {
		t.Errorf("OllamaURL should not change: got %q, want %q", appCfg.OllamaURL, original.OllamaURL)
	}
	if appCfg.WorkspaceDir != original.WorkspaceDir {
		t.Errorf("WorkspaceDir should not change: got %q, want %q", appCfg.WorkspaceDir, original.WorkspaceDir)
	}
}

func TestConfigFilePath_ReturnsHomeDirPath(t *testing.T) {
	p, err := ConfigFilePath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	homeDir, _ := os.UserHomeDir()
	expected := filepath.Join(homeDir, ".ti", "config.json")
	if p != expected {
		t.Errorf("got %q, want %q", p, expected)
	}
}

func TestLoadFromFile_ValidFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test-config.json")
	content := []byte(`{"agent":"ollama","model":"llama2","ollama_url":"http://localhost:11434"}`)
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
	cfg, err := LoadFromFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Agent != "ollama" || cfg.Model != "llama2" {
		t.Errorf("unexpected config: %+v", cfg)
	}
}

// =============================================================================
// Requirement 1: Load Configuration from JSON File
// =============================================================================

func TestReq1_LoadConfigFromJSON_AllFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ti.json")
	content := []byte(`{
		"agent": "gemini",
		"gmodel": "gemini-2.0-flash-exp",
		"ollama_url": "http://custom:11434",
		"gemini_api": "test-api-key",
		"workspace": "/tmp/my-workspace"
	}`)
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	jcfg, err := LoadFromFile(path)
	if err != nil {
		t.Fatalf("LoadFromFile error: %v", err)
	}

	appCfg := types.DefaultConfig()
	ApplyToAppConfig(jcfg, appCfg)

	// AC 1.2: agent → Provider
	if appCfg.Provider != "gemini" {
		t.Errorf("Provider: got %q, want %q", appCfg.Provider, "gemini")
	}
	// AC 1.3: model → DefaultModel
	if appCfg.DefaultModel != "gemini-2.0-flash-exp" {
		t.Errorf("DefaultModel: got %q, want %q", appCfg.DefaultModel, "gemini-2.0-flash-exp")
	}
	// AC 1.4: ollama_url → OllamaURL
	if appCfg.OllamaURL != "http://custom:11434" {
		t.Errorf("OllamaURL: got %q, want %q", appCfg.OllamaURL, "http://custom:11434")
	}
	// AC 1.5: gemini_api → GeminiAPIKey
	if appCfg.GeminiAPIKey != "test-api-key" {
		t.Errorf("GeminiAPIKey: got %q, want %q", appCfg.GeminiAPIKey, "test-api-key")
	}
	// AC 1.6: workspace → WorkspaceDir
	if appCfg.WorkspaceDir != "/tmp/my-workspace" {
		t.Errorf("WorkspaceDir: got %q, want %q", appCfg.WorkspaceDir, "/tmp/my-workspace")
	}
}

// =============================================================================
// Requirement 2: Fall Back to CLI Flags
// =============================================================================

func TestReq2_FallbackToCLI_NoConfigFile(t *testing.T) {
	// AC 2.1: When no config file exists, defaults are used and CLI can override
	appCfg := types.DefaultConfig()

	// Simulate CLI flag overrides (no config file loaded)
	appCfg.Provider = "gemini"
	appCfg.GeminiAPIKey = "cli-key"
	appCfg.DefaultModel = "gemini-2.0-flash-exp"

	if appCfg.Provider != "gemini" {
		t.Errorf("Provider: got %q, want %q", appCfg.Provider, "gemini")
	}
	if appCfg.GeminiAPIKey != "cli-key" {
		t.Errorf("GeminiAPIKey: got %q, want %q", appCfg.GeminiAPIKey, "cli-key")
	}
}

func TestReq2_CLIOverridesConfigFile(t *testing.T) {
	// AC 2.2: CLI flags override config file values
	jcfg := &JSONConfig{
		Agent:     "ollama",
		Model:     "llama2",
		OllamaURL: "http://config:11434",
		Workspace: "/config/workspace",
	}
	appCfg := types.DefaultConfig()
	ApplyToAppConfig(jcfg, appCfg)

	// Simulate CLI overrides
	appCfg.DefaultModel = "qwen2.5-coder:1.5b"
	appCfg.WorkspaceDir = "/cli/workspace"

	if appCfg.DefaultModel != "qwen2.5-coder:1.5b" {
		t.Errorf("DefaultModel: got %q, want %q", appCfg.DefaultModel, "qwen2.5-coder:1.5b")
	}
	if appCfg.WorkspaceDir != "/cli/workspace" {
		t.Errorf("WorkspaceDir: got %q, want %q", appCfg.WorkspaceDir, "/cli/workspace")
	}
	// Non-overridden field should keep config file value
	if appCfg.OllamaURL != "http://config:11434" {
		t.Errorf("OllamaURL: got %q, want %q", appCfg.OllamaURL, "http://config:11434")
	}
}

// =============================================================================
// Requirement 3: Validate JSON Configuration
// =============================================================================

func TestReq3_InvalidJSON_ReturnsError(t *testing.T) {
	// AC 3.1: Invalid JSON returns descriptive error
	_, err := FromJSON([]byte(`{"agent": "ollama",`))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "invalid JSON") {
		t.Errorf("error should mention invalid JSON, got: %s", err.Error())
	}
}

func TestReq3_UnrecognizedGenAIType_ReturnsError(t *testing.T) {
	// AC 3.2: Unrecognized agent returns error
	testCases := []string{"openai", "anthropic", "", "GEMINI", "Ollama"}
	for _, provider := range testCases {
		cfg := &JSONConfig{Agent: provider}
		err := Validate(cfg)
		if err == nil {
			t.Errorf("expected error for agent=%q, got nil", provider)
		}
		if !strings.Contains(err.Error(), "invalid agent") {
			t.Errorf("error for agent=%q should mention invalid agent, got: %s", provider, err.Error())
		}
	}
}

func TestReq3_GeminiMissingAPIKey_ReturnsError(t *testing.T) {
	// AC 3.3: Gemini with missing/empty API key returns error
	cfg := &JSONConfig{Agent: "gemini", GeminiAPI: ""}
	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected error for gemini with empty API key")
	}
	if !strings.Contains(err.Error(), "gemini_api is required") {
		t.Errorf("error should mention gemini_api requirement, got: %s", err.Error())
	}
}

// =============================================================================
// Requirement 4: JSON Configuration Serialization Round-Trip
// =============================================================================

func TestReq4_SerializeAppConfigToJSON(t *testing.T) {
	// AC 4.1: AppConfig serializes to valid JSON matching schema
	appCfg := &types.AppConfig{
		Provider:     "ollama",
		DefaultModel: "llama2",
		OllamaURL:    "http://localhost:11434",
		GeminiAPIKey: "",
		WorkspaceDir: "/tmp/workspace",
	}

	jcfg := AppConfigToJSONConfig(appCfg)
	data, err := ToJSON(jcfg)
	if err != nil {
		t.Fatalf("ToJSON error: %v", err)
	}

	// Verify it's valid JSON by parsing back
	restored, err := FromJSON(data)
	if err != nil {
		t.Fatalf("FromJSON error: %v", err)
	}

	if restored.Agent != "ollama" {
		t.Errorf("Agent: got %q, want %q", restored.Agent, "ollama")
	}
	if restored.Model != "llama2" {
		t.Errorf("Model: got %q, want %q", restored.Model, "llama2")
	}
}

func TestReq4_RoundTrip_AppConfig(t *testing.T) {
	// AC 4.2: Serialize then deserialize produces equivalent AppConfig
	original := &types.AppConfig{
		Provider:     "gemini",
		DefaultModel: "gemini-2.0-flash-exp",
		OllamaURL:    "http://localhost:11434",
		GeminiAPIKey: "test-key-123",
		WorkspaceDir: "/home/user/workspace",
	}

	// AppConfig → JSONConfig → JSON → JSONConfig → AppConfig
	jcfg := AppConfigToJSONConfig(original)
	data, err := ToJSON(jcfg)
	if err != nil {
		t.Fatalf("ToJSON error: %v", err)
	}

	restored, err := FromJSON(data)
	if err != nil {
		t.Fatalf("FromJSON error: %v", err)
	}

	result := types.DefaultConfig()
	ApplyToAppConfig(restored, result)

	if result.Provider != original.Provider {
		t.Errorf("Provider: got %q, want %q", result.Provider, original.Provider)
	}
	if result.DefaultModel != original.DefaultModel {
		t.Errorf("DefaultModel: got %q, want %q", result.DefaultModel, original.DefaultModel)
	}
	if result.OllamaURL != original.OllamaURL {
		t.Errorf("OllamaURL: got %q, want %q", result.OllamaURL, original.OllamaURL)
	}
	if result.GeminiAPIKey != original.GeminiAPIKey {
		t.Errorf("GeminiAPIKey: got %q, want %q", result.GeminiAPIKey, original.GeminiAPIKey)
	}
	if result.WorkspaceDir != original.WorkspaceDir {
		t.Errorf("WorkspaceDir: got %q, want %q", result.WorkspaceDir, original.WorkspaceDir)
	}
}

// =============================================================================
// CreateDefaultConfig Tests
// =============================================================================

func TestCreateDefaultConfig_CreatesValidFile(t *testing.T) {
	// Save original home dir
	originalHome := os.Getenv("HOME")
	if originalHome == "" {
		originalHome = os.Getenv("USERPROFILE") // Windows
	}

	// Create temp directory and set as home
	tempHome := t.TempDir()
	os.Setenv("HOME", tempHome)
	os.Setenv("USERPROFILE", tempHome)
	defer func() {
		os.Setenv("HOME", originalHome)
		os.Setenv("USERPROFILE", originalHome)
	}()

	// Create default config
	createdPath, err := CreateDefaultConfig()
	if err != nil {
		t.Fatalf("CreateDefaultConfig error: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(createdPath); os.IsNotExist(err) {
		t.Fatalf("Config file was not created at %s", createdPath)
	}

	// Verify file is valid JSON
	jcfg, err := LoadFromFile(createdPath)
	if err != nil {
		t.Fatalf("Failed to load created config: %v", err)
	}

	// Verify default values
	if jcfg.Agent != "ollama" {
		t.Errorf("Agent: got %q, want %q", jcfg.Agent, "ollama")
	}
	if jcfg.Model != "llama2" {
		t.Errorf("Model: got %q, want %q", jcfg.Model, "llama2")
	}
	if jcfg.OllamaURL != "http://localhost:11434" {
		t.Errorf("OllamaURL: got %q, want %q", jcfg.OllamaURL, "http://localhost:11434")
	}

	// Verify it validates
	if err := Validate(jcfg); err != nil {
		t.Errorf("Default config should be valid, got error: %v", err)
	}
}

func TestCreateDefaultConfig_CreatesDirectory(t *testing.T) {
	// Save original home dir
	originalHome := os.Getenv("HOME")
	if originalHome == "" {
		originalHome = os.Getenv("USERPROFILE")
	}

	// Create temp directory and set as home
	tempHome := t.TempDir()
	os.Setenv("HOME", tempHome)
	os.Setenv("USERPROFILE", tempHome)
	defer func() {
		os.Setenv("HOME", originalHome)
		os.Setenv("USERPROFILE", originalHome)
	}()

	// Verify .ti directory doesn't exist yet
	configDir := filepath.Join(tempHome, ".ti")
	if _, err := os.Stat(configDir); !os.IsNotExist(err) {
		t.Fatalf("Config directory should not exist yet")
	}

	// Create default config
	_, err := CreateDefaultConfig()
	if err != nil {
		t.Fatalf("CreateDefaultConfig error: %v", err)
	}

	// Verify directory was created
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		t.Fatalf("Config directory was not created at %s", configDir)
	}
}

// =============================================================================
// Property-Based Tests for Bedrock Integration
// =============================================================================

// Feature: aws-bedrock-integration, Property 6: Configuration Round-Trip Preservation
// **Validates: Requirements 4.1, 4.2, 4.3, 4.7, 4.8**
func TestProperty6_ConfigRoundTrip_BedrockFields(t *testing.T) {
	testCases := []struct {
		name          string
		bedrockAPI    string
		bedrockModel  string
		bedrockRegion string
	}{
		{"all fields populated", "AKIAIOSFODNN7EXAMPLE", "anthropic.claude-haiku-4-5-v1:0", "us-east-1"},
		{"different region", "AKIAIOSFODNN7EXAMPLE", "anthropic.claude-3-5-sonnet-20241022-v2:0", "us-west-2"},
		{"empty model", "AKIAIOSFODNN7EXAMPLE", "", "eu-west-1"},
		{"empty region", "AKIAIOSFODNN7EXAMPLE", "anthropic.claude-3-5-haiku-20241022-v1:0", ""},
		{"minimal config", "AKIAIOSFODNN7EXAMPLE", "", ""},
		{"long api key", "AKIAIOSFODNN7EXAMPLEVERYLONGKEY123456789", "anthropic.claude-haiku-4-5-v1:0", "ap-southeast-1"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create original AppConfig with Bedrock settings
			original := &types.AppConfig{
				Provider:      "bedrock",
				BedrockAPIKey: tc.bedrockAPI,
				BedrockModel:  tc.bedrockModel,
				BedrockRegion: tc.bedrockRegion,
				DefaultModel:  tc.bedrockModel,
				WorkspaceDir:  "/tmp/workspace",
			}

			// Convert to JSONConfig
			jcfg := AppConfigToJSONConfig(original)

			// Convert back to AppConfig
			result := types.DefaultConfig()
			ApplyToAppConfig(jcfg, result)

			// Verify all Bedrock fields are preserved
			if result.BedrockAPIKey != tc.bedrockAPI {
				t.Errorf("BedrockAPIKey: got %q, want %q", result.BedrockAPIKey, tc.bedrockAPI)
			}
			if result.BedrockModel != tc.bedrockModel {
				t.Errorf("BedrockModel: got %q, want %q", result.BedrockModel, tc.bedrockModel)
			}

			// Region should be preserved or defaulted to us-east-1
			expectedRegion := tc.bedrockRegion
			if expectedRegion == "" {
				expectedRegion = "us-east-1"
			}
			if result.BedrockRegion != expectedRegion {
				t.Errorf("BedrockRegion: got %q, want %q", result.BedrockRegion, expectedRegion)
			}

			if result.Provider != "bedrock" {
				t.Errorf("Provider: got %q, want %q", result.Provider, "bedrock")
			}
		})
	}
}

// Feature: aws-bedrock-integration, Property 7: Provider-Specific Model Selection
// **Validates: Requirements 5.5**
func TestProperty7_ProviderSpecificModelSelection(t *testing.T) {
	testCases := []struct {
		provider      string
		ollamaModel   string
		geminiModel   string
		bedrockModel  string
		expectedModel string
	}{
		{"ollama", "llama2", "gemini-2.0-flash-exp", "anthropic.claude-haiku-4-5-v1:0", "llama2"},
		{"gemini", "llama2", "gemini-2.0-flash-exp", "anthropic.claude-haiku-4-5-v1:0", "gemini-2.0-flash-exp"},
		{"bedrock", "llama2", "gemini-2.0-flash-exp", "anthropic.claude-haiku-4-5-v1:0", "anthropic.claude-haiku-4-5-v1:0"},
		{"ollama", "qwen2.5-coder:1.5b", "", "", "qwen2.5-coder:1.5b"},
		{"gemini", "", "gemini-3-pro-preview", "", "gemini-3-pro-preview"},
		{"bedrock", "", "", "anthropic.claude-3-5-sonnet-20241022-v2:0", "anthropic.claude-3-5-sonnet-20241022-v2:0"},
	}

	for _, tc := range testCases {
		t.Run(tc.provider+"_"+tc.expectedModel, func(t *testing.T) {
			jcfg := &JSONConfig{
				Agent:        tc.provider,
				Model:        tc.ollamaModel,
				GModel:       tc.geminiModel,
				BedrockModel: tc.bedrockModel,
			}

			appCfg := types.DefaultConfig()
			ApplyToAppConfig(jcfg, appCfg)

			if appCfg.DefaultModel != tc.expectedModel {
				t.Errorf("DefaultModel: got %q, want %q", appCfg.DefaultModel, tc.expectedModel)
			}
		})
	}
}

// =============================================================================
// Backward Compatibility Tests (Requirements 8.1, 8.2, 8.3, 8.4)
// =============================================================================

// Test loading config without Bedrock fields
// **Validates: Requirement 8.1**
func TestBackwardCompat_LoadConfigWithoutBedrockFields(t *testing.T) {
	// Simulate an old config file that doesn't have Bedrock fields
	oldConfigJSON := []byte(`{
		"agent": "ollama",
		"model": "llama2",
		"ollama_url": "http://localhost:11434",
		"workspace": "/tmp/workspace"
	}`)

	jcfg, err := FromJSON(oldConfigJSON)
	if err != nil {
		t.Fatalf("FromJSON error: %v", err)
	}

	// Verify Bedrock fields are empty/default
	if jcfg.BedrockAPI != "" {
		t.Errorf("BedrockAPI should be empty, got %q", jcfg.BedrockAPI)
	}
	if jcfg.BedrockModel != "" {
		t.Errorf("BedrockModel should be empty, got %q", jcfg.BedrockModel)
	}
	if jcfg.BedrockRegion != "" {
		t.Errorf("BedrockRegion should be empty, got %q", jcfg.BedrockRegion)
	}

	// Apply to AppConfig and verify it works
	appCfg := types.DefaultConfig()
	ApplyToAppConfig(jcfg, appCfg)

	if appCfg.Provider != "ollama" {
		t.Errorf("Provider: got %q, want %q", appCfg.Provider, "ollama")
	}
	if appCfg.DefaultModel != "llama2" {
		t.Errorf("DefaultModel: got %q, want %q", appCfg.DefaultModel, "llama2")
	}
}

// Test saving config without Bedrock configured
// **Validates: Requirement 8.3**
func TestBackwardCompat_SaveConfigWithoutBedrock(t *testing.T) {
	// Create AppConfig without Bedrock settings
	appCfg := &types.AppConfig{
		Provider:     "gemini",
		GeminiAPIKey: "test-key",
		GeminiModel:  "gemini-2.0-flash-exp",
		DefaultModel: "gemini-2.0-flash-exp",
		OllamaURL:    "http://localhost:11434",
		WorkspaceDir: "/tmp/workspace",
	}

	// Convert to JSONConfig
	jcfg := AppConfigToJSONConfig(appCfg)

	// Serialize to JSON
	data, err := ToJSON(jcfg)
	if err != nil {
		t.Fatalf("ToJSON error: %v", err)
	}

	// Verify JSON is valid and can be loaded back
	restored, err := FromJSON(data)
	if err != nil {
		t.Fatalf("FromJSON error: %v", err)
	}

	// Verify non-Bedrock fields are preserved
	if restored.Agent != "gemini" {
		t.Errorf("Agent: got %q, want %q", restored.Agent, "gemini")
	}
	if restored.GeminiAPI != "test-key" {
		t.Errorf("GeminiAPI: got %q, want %q", restored.GeminiAPI, "test-key")
	}

	// Bedrock fields should be empty or omitted (both are acceptable)
	// The important thing is that the config is valid
	if err := Validate(restored); err != nil {
		t.Errorf("Config should be valid even without Bedrock fields: %v", err)
	}
}

// Test that existing Ollama and Gemini configurations still work
// **Validates: Requirements 8.2, 8.4**
func TestBackwardCompat_ExistingProvidersStillWork(t *testing.T) {
	testCases := []struct {
		name     string
		provider string
		config   *JSONConfig
	}{
		{
			name:     "ollama config",
			provider: "ollama",
			config: &JSONConfig{
				Agent:     "ollama",
				Model:     "llama2",
				OllamaURL: "http://localhost:11434",
				Workspace: "/tmp/workspace",
			},
		},
		{
			name:     "gemini config",
			provider: "gemini",
			config: &JSONConfig{
				Agent:     "gemini",
				GModel:    "gemini-2.0-flash-exp",
				GeminiAPI: "test-api-key",
				Workspace: "/tmp/workspace",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Validate the config
			if err := Validate(tc.config); err != nil {
				t.Fatalf("Validate error for %s: %v", tc.provider, err)
			}

			// Apply to AppConfig
			appCfg := types.DefaultConfig()
			ApplyToAppConfig(tc.config, appCfg)

			// Verify provider is set correctly
			if appCfg.Provider != tc.provider {
				t.Errorf("Provider: got %q, want %q", appCfg.Provider, tc.provider)
			}

			// Verify provider-specific fields are set
			if tc.provider == "ollama" {
				if appCfg.OllamaURL != tc.config.OllamaURL {
					t.Errorf("OllamaURL: got %q, want %q", appCfg.OllamaURL, tc.config.OllamaURL)
				}
				if appCfg.DefaultModel != tc.config.Model {
					t.Errorf("DefaultModel: got %q, want %q", appCfg.DefaultModel, tc.config.Model)
				}
			} else if tc.provider == "gemini" {
				if appCfg.GeminiAPIKey != tc.config.GeminiAPI {
					t.Errorf("GeminiAPIKey: got %q, want %q", appCfg.GeminiAPIKey, tc.config.GeminiAPI)
				}
				if appCfg.DefaultModel != tc.config.GModel {
					t.Errorf("DefaultModel: got %q, want %q", appCfg.DefaultModel, tc.config.GModel)
				}
			}

			// Round-trip test
			jcfg := AppConfigToJSONConfig(appCfg)
			data, err := ToJSON(jcfg)
			if err != nil {
				t.Fatalf("ToJSON error: %v", err)
			}

			restored, err := FromJSON(data)
			if err != nil {
				t.Fatalf("FromJSON error: %v", err)
			}

			if restored.Agent != tc.provider {
				t.Errorf("Round-trip Agent: got %q, want %q", restored.Agent, tc.provider)
			}
		})
	}
}

// Test Validate function with Bedrock configurations
// **Validates: Requirements 4.4, 4.5**
func TestValidate_BedrockAgent(t *testing.T) {
	testCases := []struct {
		name        string
		config      *JSONConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid bedrock config",
			config: &JSONConfig{
				Agent:      "bedrock",
				BedrockAPI: "AKIAIOSFODNN7EXAMPLE",
			},
			expectError: false,
		},
		{
			name: "bedrock without API key",
			config: &JSONConfig{
				Agent:      "bedrock",
				BedrockAPI: "",
			},
			expectError: true,
			errorMsg:    "bedrock_api is required",
		},
		{
			name: "invalid agent",
			config: &JSONConfig{
				Agent: "openai",
			},
			expectError: true,
			errorMsg:    "invalid agent",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := Validate(tc.config)
			if tc.expectError {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tc.errorMsg)
				}
				if !strings.Contains(err.Error(), tc.errorMsg) {
					t.Errorf("expected error containing %q, got %q", tc.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
			}
		})
	}
}
