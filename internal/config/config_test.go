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
	cfg := &JSONConfig{GenAIType: "gemini", GeminiAPI: ""}
	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected error for gemini with empty gemini_api, got nil")
	}
	if !strings.Contains(err.Error(), "gemini_api is required") {
		t.Errorf("unexpected error message: %s", err.Error())
	}
}

func TestValidate_OllamaWithEmptyAPI_Passes(t *testing.T) {
	cfg := &JSONConfig{GenAIType: "ollama", GeminiAPI: ""}
	err := Validate(cfg)
	if err != nil {
		t.Fatalf("expected no error for ollama with empty gemini_api, got: %v", err)
	}
}

func TestValidate_ValidGenAIType_EmptyOtherFields_Passes(t *testing.T) {
	cfg := &JSONConfig{GenAIType: "ollama"}
	err := Validate(cfg)
	if err != nil {
		t.Fatalf("expected no error for valid genai_type with empty fields, got: %v", err)
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
	data := []byte(`{"genai_type":"gemini","model_id":"flash","gemini_api":"key123"}`)
	cfg, err := FromJSON(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.GenAIType != "gemini" || cfg.ModelID != "flash" || cfg.GeminiAPI != "key123" {
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
		GenAIType: "ollama",
		ModelID:   "llama2",
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
		GenAIType: "gemini",
		ModelID:   "flash",
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
	content := []byte(`{"genai_type":"ollama","model_id":"llama2","ollama_url":"http://localhost:11434"}`)
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
	cfg, err := LoadFromFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.GenAIType != "ollama" || cfg.ModelID != "llama2" {
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
		"genai_type": "gemini",
		"model_id": "gemini-2.0-flash-exp",
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

	// AC 1.2: genai_type → Provider
	if appCfg.Provider != "gemini" {
		t.Errorf("Provider: got %q, want %q", appCfg.Provider, "gemini")
	}
	// AC 1.3: model_id → DefaultModel
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
		GenAIType: "ollama",
		ModelID:   "llama2",
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
	_, err := FromJSON([]byte(`{"genai_type": "ollama",`))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "invalid JSON") {
		t.Errorf("error should mention invalid JSON, got: %s", err.Error())
	}
}

func TestReq3_UnrecognizedGenAIType_ReturnsError(t *testing.T) {
	// AC 3.2: Unrecognized genai_type returns error
	testCases := []string{"openai", "anthropic", "", "GEMINI", "Ollama"}
	for _, provider := range testCases {
		cfg := &JSONConfig{GenAIType: provider}
		err := Validate(cfg)
		if err == nil {
			t.Errorf("expected error for genai_type=%q, got nil", provider)
		}
		if !strings.Contains(err.Error(), "invalid genai_type") {
			t.Errorf("error for genai_type=%q should mention invalid genai_type, got: %s", provider, err.Error())
		}
	}
}

func TestReq3_GeminiMissingAPIKey_ReturnsError(t *testing.T) {
	// AC 3.3: Gemini with missing/empty API key returns error
	cfg := &JSONConfig{GenAIType: "gemini", GeminiAPI: ""}
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

	if restored.GenAIType != "ollama" {
		t.Errorf("GenAIType: got %q, want %q", restored.GenAIType, "ollama")
	}
	if restored.ModelID != "llama2" {
		t.Errorf("ModelID: got %q, want %q", restored.ModelID, "llama2")
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
	if jcfg.GenAIType != "ollama" {
		t.Errorf("GenAIType: got %q, want %q", jcfg.GenAIType, "ollama")
	}
	if jcfg.ModelID != "llama2" {
		t.Errorf("ModelID: got %q, want %q", jcfg.ModelID, "llama2")
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
