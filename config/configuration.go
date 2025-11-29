package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

// Configuration 配置接口
type Configuration interface {
	// Get 获取配置值
	Get(key string) string
	// GetWithDefault 获取配置值，如果不存在则返回默认值
	GetWithDefault(key, defaultValue string) string
	// GetInt 获取整数配置值
	GetInt(key string) (int, error)
	// GetBool 获取布尔配置值
	GetBool(key string) (bool, error)
	// GetSection 获取配置节
	GetSection(key string) Configuration
	// Bind 绑定配置到结构体
	Bind(key string, target any) error
	// GetAll 获取所有配置
	GetAll() map[string]any

	// LoadFile 加载文件
	LoadFile(path string) error
	// LoadEnv 加载环境变量
	LoadEnv(prefix ...string)
}

// configuration 配置实现
type configuration struct {
	data map[string]any
	mu   sync.RWMutex
}

// NewConfiguration 创建新的配置实例
func NewConfiguration() Configuration {
	return &configuration{
		data: make(map[string]any),
	}
}

// LoadFile 加载配置文件 (支持 .json, .yaml, .yml)
func (c *configuration) LoadFile(path string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	ext := strings.ToLower(filepath.Ext(path))
	var loadedData map[string]any

	switch ext {
	case ".json":
		if err := json.Unmarshal(data, &loadedData); err != nil {
			return fmt.Errorf("failed to parse JSON: %w", err)
		}
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &loadedData); err != nil {
			return fmt.Errorf("failed to parse YAML: %w", err)
		}
	default:
		return fmt.Errorf("unsupported config file extension: %s", ext)
	}

	mergeMaps(c.data, loadedData)
	return nil
}

// LoadEnv 加载环境变量
func (c *configuration) LoadEnv(prefix ...string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	envPrefix := ""
	if len(prefix) > 0 {
		envPrefix = prefix[0]
	}

	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key, value := parts[0], parts[1]

		// 检查前缀
		if envPrefix != "" {
			if !strings.HasPrefix(key, envPrefix) {
				continue
			}
			key = strings.TrimPrefix(key, envPrefix)
		}

		// 转换为小写
		key = strings.ToLower(key)
		// 将 __ 转换为 :
		key = strings.ReplaceAll(key, "__", ":")
		// 将 _ 转换为 .
		key = strings.ReplaceAll(key, "_", ".")

		setNestedValue(c.data, key, value)
	}
}

// Get 获取配置值
func (c *configuration) Get(key string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	value := c.getByPath(key)
	if value == nil {
		return ""
	}

	switch v := value.(type) {
	case string:
		return v
	case int, int64, float64:
		return fmt.Sprintf("%v", v)
	case bool:
		return strconv.FormatBool(v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// GetWithDefault 获取配置值，如果不存在则返回默认值
func (c *configuration) GetWithDefault(key, defaultValue string) string {
	value := c.Get(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// GetInt 获取整数配置值
func (c *configuration) GetInt(key string) (int, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	value := c.getByPath(key)
	if value == nil {
		return 0, fmt.Errorf("key %s not found", key)
	}

	switch v := value.(type) {
	case int:
		return v, nil
	case int64:
		return int(v), nil
	case float64:
		return int(v), nil
	case string:
		return strconv.Atoi(v)
	default:
		return 0, fmt.Errorf("cannot convert %v to int", value)
	}
}

// GetBool 获取布尔配置值
func (c *configuration) GetBool(key string) (bool, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	value := c.getByPath(key)
	if value == nil {
		return false, fmt.Errorf("key %s not found", key)
	}

	switch v := value.(type) {
	case bool:
		return v, nil
	case string:
		return strconv.ParseBool(v)
	default:
		return false, fmt.Errorf("cannot convert %v to bool", value)
	}
}

// GetSection 获取配置节
func (c *configuration) GetSection(key string) Configuration {
	c.mu.RLock()
	defer c.mu.RUnlock()

	value := c.getByPath(key)
	if value == nil {
		return &configuration{data: make(map[string]any)}
	}

	if m, ok := value.(map[string]any); ok {
		return &configuration{data: m}
	}

	return &configuration{data: make(map[string]any)}
}

// Bind 绑定配置到结构体
func (c *configuration) Bind(key string, target any) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var data any
	if key == "" {
		data = c.data
	} else {
		data = c.getByPath(key)
	}

	if data == nil {
		return fmt.Errorf("key %s not found", key)
	}

	// 使用 YAML 序列化/反序列化进行绑定 (比 JSON 更宽容，支持 map[interface]interface)
	// 但为了兼容性，先试用 JSON。
	// 注意：gopkg.in/yaml.v3 Unmarshal 出来的 map key 可能是 string。
	// 我们这里 data 本身已经是 map[string]any。
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	if err := json.Unmarshal(jsonData, target); err != nil {
		return fmt.Errorf("failed to unmarshal data: %w", err)
	}

	return nil
}

// GetAll 获取所有配置
func (c *configuration) GetAll() map[string]any {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// 返回副本
	result := make(map[string]any)
	mergeMaps(result, c.data)
	return result
}

// getByPath 通过路径获取值（支持 "a:b:c" 或 "a.b.c"）
func (c *configuration) getByPath(path string) any {
	if path == "" {
		return c.data
	}

	// 支持 : 和 . 作为分隔符
	parts := strings.Split(strings.ReplaceAll(path, ":", "."), ".")

	current := any(c.data)
	for _, part := range parts {
		if m, ok := current.(map[string]any); ok {
			current = m[part]
		} else {
			return nil
		}
	}

	return current
}

// mergeMaps 合并两个 map
func mergeMaps(dst, src map[string]any) {
	for k, v := range src {
		if dstMap, ok := dst[k].(map[string]any); ok {
			if srcMap, ok := v.(map[string]any); ok {
				mergeMaps(dstMap, srcMap)
				continue
			}
		}
		dst[k] = v
	}
}

// setNestedValue 设置嵌套值
func setNestedValue(data map[string]any, path string, value any) {
	parts := strings.Split(strings.ReplaceAll(path, ".", ":"), ":")
	current := data

	for i := 0; i < len(parts)-1; i++ {
		part := parts[i]
		if _, exists := current[part]; !exists {
			current[part] = make(map[string]any)
		}
		if m, ok := current[part].(map[string]any); ok {
			current = m
		} else {
			return
		}
	}

	// 尝试转换字符串值为合适的类型
	if strValue, ok := value.(string); ok {
		if intValue, err := strconv.Atoi(strValue); err == nil {
			value = intValue
		} else if floatValue, err := strconv.ParseFloat(strValue, 64); err == nil {
			value = floatValue
		} else if boolValue, err := strconv.ParseBool(strValue); err == nil {
			value = boolValue
		}
	}

	current[parts[len(parts)-1]] = value
}
