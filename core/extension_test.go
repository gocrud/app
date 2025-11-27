package core

import (
	"testing"
)

// 定义各种 Extension 实现用于测试

// EmptyExtension 未实现任何接口
type EmptyExtension struct{}

func (e *EmptyExtension) Name() string { return "Empty" }

// ServiceOnlyExtension 仅实现 ServiceConfigurator
type ServiceOnlyExtension struct{}

func (e *ServiceOnlyExtension) Name() string { return "ServiceOnly" }
func (e *ServiceOnlyExtension) ConfigureServices(s *ServiceCollection) {}

// AppOnlyExtension 仅实现 AppConfigurator
type AppOnlyExtension struct{}

func (e *AppOnlyExtension) Name() string { return "AppOnly" }
func (e *AppOnlyExtension) ConfigureBuilder(ctx *BuildContext) {}

// FullExtension 同时实现 ServiceConfigurator 和 AppConfigurator
type FullExtension struct{}

func (e *FullExtension) Name() string { return "Full" }
func (e *FullExtension) ConfigureServices(s *ServiceCollection) {}
func (e *FullExtension) ConfigureBuilder(ctx *BuildContext) {}

func TestAddExtension_Panic_WhenNoInterfaceImplemented(t *testing.T) {
	builder := NewApplicationBuilder()

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic as expected for EmptyExtension")
		} else {
			// 验证 panic 消息包含 Extension 名称
			errStr := r.(string)
			expectedPart := "Extension 'Empty' does not implement any supported interfaces"
			if len(errStr) < len(expectedPart) || errStr[:len(expectedPart)] != expectedPart {
				// 简单的包含检查
				importStrings := "Extension 'Empty' does not implement any supported interfaces"
				if !contains(errStr, importStrings) {
					t.Errorf("Panic message not match. Got: %v", errStr)
				}
			}
		}
	}()

	builder.AddExtension(&EmptyExtension{})
}

func TestAddExtension_Success_ServiceOnly(t *testing.T) {
	builder := NewApplicationBuilder()
	builder.AddExtension(&ServiceOnlyExtension{})

	if len(builder.serviceConfigurators) != 1 {
		t.Errorf("Expected 1 service configurator, got %d", len(builder.serviceConfigurators))
	}
	if len(builder.configurators) != 0 {
		t.Errorf("Expected 0 app configurators, got %d", len(builder.configurators))
	}
}

func TestAddExtension_Success_AppOnly(t *testing.T) {
	builder := NewApplicationBuilder()
	builder.AddExtension(&AppOnlyExtension{})

	if len(builder.serviceConfigurators) != 0 {
		t.Errorf("Expected 0 service configurators, got %d", len(builder.serviceConfigurators))
	}
	if len(builder.configurators) != 1 {
		t.Errorf("Expected 1 app configurator, got %d", len(builder.configurators))
	}
}

func TestAddExtension_Success_Full(t *testing.T) {
	builder := NewApplicationBuilder()
	builder.AddExtension(&FullExtension{})

	if len(builder.serviceConfigurators) != 1 {
		t.Errorf("Expected 1 service configurator, got %d", len(builder.serviceConfigurators))
	}
	if len(builder.configurators) != 1 {
		t.Errorf("Expected 1 app configurator, got %d", len(builder.configurators))
	}
}

func TestAddExtension_Multiple(t *testing.T) {
	builder := NewApplicationBuilder()
	builder.AddExtension(&ServiceOnlyExtension{})
	builder.AddExtension(&AppOnlyExtension{})
	builder.AddExtension(&FullExtension{})

	if len(builder.serviceConfigurators) != 2 { // ServiceOnly + Full
		t.Errorf("Expected 2 service configurators, got %d", len(builder.serviceConfigurators))
	}
	if len(builder.configurators) != 2 { // AppOnly + Full
		t.Errorf("Expected 2 app configurators, got %d", len(builder.configurators))
	}
}

// 简单的字符串包含辅助函数
func contains(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

