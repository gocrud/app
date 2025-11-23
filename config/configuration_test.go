package config

import (
	"testing"
	"sync"
)

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

