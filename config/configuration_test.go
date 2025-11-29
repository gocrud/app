package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfiguration(t *testing.T) {
	// 1. Test NewConfiguration
	cfg := NewConfiguration()
	if cfg == nil {
		t.Fatal("NewConfiguration returned nil")
	}

	// 2. Test LoadEnv
	os.Setenv("TEST_APP_NAME", "Gocrud")
	os.Setenv("TEST_DB_PORT", "5432")
	// 测试嵌套转换
	os.Setenv("TEST_NESTED__KEY", "nested_value")

	cfg.LoadEnv("TEST_")

	if val := cfg.Get("app.name"); val != "Gocrud" {
		t.Errorf("Expected app.name=Gocrud, got %s", val)
	}
	if val, _ := cfg.GetInt("db.port"); val != 5432 {
		t.Errorf("Expected db.port=5432, got %d", val)
	}
	if val := cfg.Get("nested.key"); val != "nested_value" {
		t.Errorf("Expected nested.key=nested_value, got %s", val)
	}

	// 3. Test LoadFile (JSON)
	jsonContent := `{"server": {"host": "localhost", "port": 8080}}`
	jsonFile := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(jsonFile, []byte(jsonContent), 0644); err != nil {
		t.Fatal(err)
	}

	if err := cfg.LoadFile(jsonFile); err != nil {
		t.Fatalf("LoadFile JSON failed: %v", err)
	}

	if val := cfg.Get("server.host"); val != "localhost" {
		t.Errorf("Expected server.host=localhost, got %s", val)
	}
	if val, _ := cfg.GetInt("server.port"); val != 8080 {
		t.Errorf("Expected server.port=8080, got %d", val)
	}

	// 4. Test LoadFile (YAML) - Merge
	yamlContent := `
server:
  timeout: 30s
log:
  level: info
`
	yamlFile := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(yamlFile, []byte(yamlContent), 0644); err != nil {
		t.Fatal(err)
	}

	if err := cfg.LoadFile(yamlFile); err != nil {
		t.Fatalf("LoadFile YAML failed: %v", err)
	}

	// Verify merge: old values kept, new values added
	if val, _ := cfg.GetInt("server.port"); val != 8080 {
		t.Errorf("server.port should remain 8080, got %d", val)
	}
	if val := cfg.Get("server.timeout"); val != "30s" {
		t.Errorf("Expected server.timeout=30s, got %s", val)
	}
	if val := cfg.Get("log.level"); val != "info" {
		t.Errorf("Expected log.level=info, got %s", val)
	}

	// 5. Test Bind
	type ServerConfig struct {
		Host    string `json:"host"`
		Port    int    `json:"port"`
		Timeout string `json:"timeout"`
	}
	var serverCfg ServerConfig
	if err := cfg.Bind("server", &serverCfg); err != nil {
		t.Fatalf("Bind failed: %v", err)
	}

	if serverCfg.Host != "localhost" || serverCfg.Port != 8080 || serverCfg.Timeout != "30s" {
		t.Errorf("Bind mismatch: %+v", serverCfg)
	}
}
