package utils

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestGetConfigPath_EnvOverride(t *testing.T) {
	want := "/custom/path/.purrconfig"
	t.Setenv("PURR_CONFIG_PATH", want)

	got, err := GetConfigPath()
	if err != nil {
		t.Fatalf("GetConfigPath() unexpected error: %v", err)
	}
	if got != want {
		t.Errorf("GetConfigPath() = %q, want %q", got, want)
	}
}

func TestGetConfigPath_Default(t *testing.T) {
	// Ensure env is unset so the default path is used
	t.Setenv("PURR_CONFIG_PATH", "")

	got, err := GetConfigPath()
	if err != nil {
		t.Fatalf("GetConfigPath() unexpected error: %v", err)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("os.UserHomeDir() unexpected error: %v", err)
	}
	want := filepath.Join(homeDir, ".purrconfig")
	if got != want {
		t.Errorf("GetConfigPath() = %q, want %q", got, want)
	}
}

func TestReadConfig_NonExistentFile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("PURR_CONFIG_PATH", filepath.Join(tmpDir, ".purrconfig"))

	cfg, err := ReadConfig()
	if err != nil {
		t.Fatalf("ReadConfig() unexpected error: %v", err)
	}
	if cfg == nil {
		t.Fatal("ReadConfig() returned nil config, want empty PurrConfig")
	}
	if cfg.UserName != "" || cfg.UserEmail != "" {
		t.Errorf("ReadConfig() = %+v, want empty PurrConfig", cfg)
	}
}

func TestReadConfig_ValidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".purrconfig")
	t.Setenv("PURR_CONFIG_PATH", configPath)

	data := []byte(`{"user_name":"alice","user_email":"alice@example.com"}`)
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := ReadConfig()
	if err != nil {
		t.Fatalf("ReadConfig() unexpected error: %v", err)
	}
	if cfg.UserName != "alice" {
		t.Errorf("UserName = %q, want %q", cfg.UserName, "alice")
	}
	if cfg.UserEmail != "alice@example.com" {
		t.Errorf("UserEmail = %q, want %q", cfg.UserEmail, "alice@example.com")
	}
}

func TestReadConfig_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".purrconfig")
	t.Setenv("PURR_CONFIG_PATH", configPath)

	if err := os.WriteFile(configPath, []byte("{not valid json!!!"), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	_, err := ReadConfig()
	if err == nil {
		t.Fatal("ReadConfig() expected error for invalid JSON, got nil")
	}
}

func TestWriteConfig_CreatesFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".purrconfig")
	t.Setenv("PURR_CONFIG_PATH", configPath)

	cfg := &PurrConfig{
		UserName:  "bob",
		UserEmail: "bob@example.com",
	}
	if err := WriteConfig(cfg); err != nil {
		t.Fatalf("WriteConfig() unexpected error: %v", err)
	}

	// Verify file was created
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config file after write: %v", err)
	}

	var got PurrConfig
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("failed to unmarshal written config: %v", err)
	}
	if got.UserName != "bob" {
		t.Errorf("UserName = %q, want %q", got.UserName, "bob")
	}
	if got.UserEmail != "bob@example.com" {
		t.Errorf("UserEmail = %q, want %q", got.UserEmail, "bob@example.com")
	}
}

func TestReadWriteConfig_Roundtrip(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".purrconfig")
	t.Setenv("PURR_CONFIG_PATH", configPath)

	original := &PurrConfig{
		UserName:  "charlie",
		UserEmail: "charlie@example.com",
	}

	if err := WriteConfig(original); err != nil {
		t.Fatalf("WriteConfig() unexpected error: %v", err)
	}

	loaded, err := ReadConfig()
	if err != nil {
		t.Fatalf("ReadConfig() unexpected error: %v", err)
	}

	if loaded.UserName != original.UserName {
		t.Errorf("UserName = %q, want %q", loaded.UserName, original.UserName)
	}
	if loaded.UserEmail != original.UserEmail {
		t.Errorf("UserEmail = %q, want %q", loaded.UserEmail, original.UserEmail)
	}
}

func TestWriteConfig_Overwrites(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".purrconfig")
	t.Setenv("PURR_CONFIG_PATH", configPath)

	// Write first config
	first := &PurrConfig{
		UserName:  "first_user",
		UserEmail: "first@example.com",
	}
	if err := WriteConfig(first); err != nil {
		t.Fatalf("WriteConfig(first) unexpected error: %v", err)
	}

	// Write second config over the same file
	second := &PurrConfig{
		UserName:  "second_user",
		UserEmail: "second@example.com",
	}
	if err := WriteConfig(second); err != nil {
		t.Fatalf("WriteConfig(second) unexpected error: %v", err)
	}

	// Read back and verify the latest values
	loaded, err := ReadConfig()
	if err != nil {
		t.Fatalf("ReadConfig() unexpected error: %v", err)
	}
	if loaded.UserName != "second_user" {
		t.Errorf("UserName = %q, want %q", loaded.UserName, "second_user")
	}
	if loaded.UserEmail != "second@example.com" {
		t.Errorf("UserEmail = %q, want %q", loaded.UserEmail, "second@example.com")
	}
}
