package config

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestConfiguration_Basic(t *testing.T) {
	builder := NewConfigurationBuilder()
	builder.AddInMemory(map[string]any{
		"app": map[string]any{
			"name": "test",
			"port": 8080,
		},
		"debug": true,
	})

	config, err := builder.Build()
	if err != nil {
		t.Fatalf("Failed to build config: %v", err)
	}

	if val := config.Get("app:name"); val != "test" {
		t.Errorf("Expected 'test', got '%s'", val)
	}
	if val := config.Get("app.name"); val != "test" {
		t.Errorf("Expected 'test' with dot separator, got '%s'", val)
	}

	port, err := config.GetInt("app:port")
	if err != nil || port != 8080 {
		t.Errorf("Expected 8080, got %d (err: %v)", port, err)
	}

	debug, err := config.GetBool("debug")
	if err != nil || !debug {
		t.Errorf("Expected true, got %v (err: %v)", debug, err)
	}
}

func TestConfiguration_CommandLine(t *testing.T) {
	args := []string{
		"--app:name=cmd-app",
		"--app.port=9090",
		"--debug=false",
		"--nested:key:val=123",
	}

	builder := NewConfigurationBuilder()
	builder.AddCommandLine(args)

	config, err := builder.Build()
	if err != nil {
		t.Fatalf("Failed to build config: %v", err)
	}

	if val := config.Get("app:name"); val != "cmd-app" {
		t.Errorf("Expected 'cmd-app', got '%s'", val)
	}
	if port, _ := config.GetInt("app:port"); port != 9090 {
		t.Errorf("Expected 9090, got %d", port)
	}
	if debug, _ := config.GetBool("debug"); debug {
		t.Error("Expected debug=false")
	}
	if val, _ := config.GetInt("nested:key:val"); val != 123 {
		t.Errorf("Expected 123, got %d", val)
	}
}

func TestConfiguration_EnvironmentVariables(t *testing.T) {
	os.Setenv("TESTAPP_HOST", "localhost")
	os.Setenv("TESTAPP_PORT", "8080")
	os.Setenv("TESTAPP_DB__NAME", "mydb") // Double underscore for nested

	defer func() {
		os.Unsetenv("TESTAPP_HOST")
		os.Unsetenv("TESTAPP_PORT")
		os.Unsetenv("TESTAPP_DB__NAME")
	}()

	builder := NewConfigurationBuilder()
	builder.AddEnvironmentVariables("TESTAPP_")

	config, err := builder.Build()
	if err != nil {
		t.Fatalf("Failed to build config: %v", err)
	}

	if val := config.Get("host"); val != "localhost" {
		t.Errorf("Expected localhost, got %s", val)
	}
	if port, _ := config.GetInt("port"); port != 8080 {
		t.Errorf("Expected 8080, got %d", port)
	}
	if val := config.Get("db:name"); val != "mydb" {
		t.Errorf("Expected mydb, got %s", val)
	}
}

func TestConfiguration_Hierarchy(t *testing.T) {
	// Priority: CommandLine > Env > InMemory > Json
	builder := NewConfigurationBuilder()

	// 1. Base: InMemory
	builder.AddInMemory(map[string]any{
		"key": "memory",
		"val": 1,
	})

	// 2. Override: Env
	os.Setenv("TEST_KEY", "env")
	defer os.Unsetenv("TEST_KEY")
	builder.AddEnvironmentVariables("TEST_")

	// 3. Override: CommandLine
	builder.AddCommandLine([]string{"--key=cmd"})

	config, _ := builder.Build()

	// key should be from CommandLine
	if val := config.Get("key"); val != "cmd" {
		t.Errorf("Expected 'cmd', got '%s'", val)
	}

	// val should be from InMemory (not overridden)
	if val, _ := config.GetInt("val"); val != 1 {
		t.Errorf("Expected 1, got %d", val)
	}
}

func TestConfiguration_JsonFile(t *testing.T) {
	content := `{"app": {"name": "json-app"}}`
	tmpfile := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(tmpfile, []byte(content), 0666); err != nil {
		t.Fatal(err)
	}

	builder := NewConfigurationBuilder()
	builder.AddJsonFile(tmpfile)

	config, err := builder.Build()
	if err != nil {
		t.Fatalf("Failed to build config: %v", err)
	}

	if val := config.Get("app:name"); val != "json-app" {
		t.Errorf("Expected 'json-app', got '%s'", val)
	}
}

func TestConfiguration_YamlFile(t *testing.T) {
	content := `
app:
  name: yaml-app
list:
  - item1
  - item2
`
	tmpfile := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(tmpfile, []byte(content), 0666); err != nil {
		t.Fatal(err)
	}

	builder := NewConfigurationBuilder()
	builder.AddYamlFile(tmpfile)

	config, err := builder.Build()
	if err != nil {
		t.Fatalf("Failed to build config: %v", err)
	}

	if val := config.Get("app:name"); val != "yaml-app" {
		t.Errorf("Expected 'yaml-app', got '%s'", val)
	}
}

func TestConfiguration_Bind(t *testing.T) {
	type Config struct {
		Host string `json:"host"`
		Port int    `json:"port"`
	}

	builder := NewConfigurationBuilder()
	builder.AddInMemory(map[string]any{
		"server": map[string]any{
			"host": "localhost",
			"port": 8080,
		},
	})

	config, _ := builder.Build()

	var cfg Config
	if err := config.Bind("server", &cfg); err != nil {
		t.Fatalf("Bind failed: %v", err)
	}

	if cfg.Host != "localhost" || cfg.Port != 8080 {
		t.Errorf("Bind result mismatch: %+v", cfg)
	}
}

func TestValueStore(t *testing.T) {
	store := NewValueStore()

	data := map[string]any{"key": "value"}
	store.Store(data)

	loaded := store.Load()
	if loaded["key"] != "value" {
		t.Error("Load failed")
	}

	// Test concurrency
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			store.Load()
		}()
	}
	wg.Wait()
}

func TestPathCache(t *testing.T) {
	cache := &PathCache{}

	path := "a:b.c"
	parts := cache.GetPathSegments(path)

	if len(parts) != 3 {
		t.Errorf("Expected 3 parts, got %d", len(parts))
	}
	if parts[0] != "a" || parts[1] != "b" || parts[2] != "c" {
		t.Error("Parse failed")
	}

	// Test cache hit
	parts2 := cache.GetPathSegments(path)
	if len(parts2) != 3 {
		t.Errorf("Expected 3 parts on second call, got %d", len(parts2))
	}
}

func BenchmarkConfigGet(b *testing.B) {
	// Setup config
	builder := NewConfigurationBuilder()
	builder.AddInMemory(map[string]any{
		"server": map[string]any{
			"host": "localhost",
			"port": 8080,
		},
	})
	config, _ := builder.BuildReloadable()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		config.Get("server:host")
	}
}
