package config

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"gopkg.in/yaml.v3"
)

// Configuration 配置接口（类似于 .NET Core IConfiguration）
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
}

// ConfigurationBuilder 配置构建器
type ConfigurationBuilder struct {
	sources []ConfigurationSource
	mu      sync.RWMutex
}

// ConfigurationSource 配置源接口
type ConfigurationSource interface {
	Load() (map[string]any, error)
	Name() string
}

// NewConfigurationBuilder 创建配置构建器
func NewConfigurationBuilder() *ConfigurationBuilder {
	return &ConfigurationBuilder{
		sources: make([]ConfigurationSource, 0),
	}
}

// Add 添加配置源
func (b *ConfigurationBuilder) Add(source ConfigurationSource) *ConfigurationBuilder {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.sources = append(b.sources, source)
	return b
}

// AddJsonFile 添加 JSON 文件配置源
func (b *ConfigurationBuilder) AddJsonFile(path string, optional ...bool) *ConfigurationBuilder {
	isOptional := len(optional) > 0 && optional[0]
	return b.Add(&JsonFileSource{Path: path, Optional: isOptional})
}

// AddYamlFile 添加 YAML 文件配置源
func (b *ConfigurationBuilder) AddYamlFile(path string, optional ...bool) *ConfigurationBuilder {
	isOptional := len(optional) > 0 && optional[0]
	return b.Add(&YamlFileSource{Path: path, Optional: isOptional})
}

// AddEnvironmentVariables 添加环境变量配置源
func (b *ConfigurationBuilder) AddEnvironmentVariables(prefix string) *ConfigurationBuilder {
	return b.Add(&EnvironmentVariableSource{Prefix: prefix})
}

// AddInMemory 添加内存配置源
func (b *ConfigurationBuilder) AddInMemory(data map[string]any) *ConfigurationBuilder {
	return b.Add(&InMemorySource{Data: data})
}

// EtcdOptions etcd 配置选项
type EtcdOptions struct {
	Endpoints   []string      // etcd 服务器地址列表
	Username    string        // 用户名（可选）
	Password    string        // 密码（可选）
	Prefix      string        // 键前缀（可选）
	Timeout     time.Duration // 连接超时时间（默认 5 秒）
	DialTimeout time.Duration // 拨号超时时间（默认 5 秒）
}

// AddEtcd 添加 etcd 配置源
func (b *ConfigurationBuilder) AddEtcd(opts EtcdOptions) *ConfigurationBuilder {
	if opts.Timeout == 0 {
		opts.Timeout = 5 * time.Second
	}
	if opts.DialTimeout == 0 {
		opts.DialTimeout = 5 * time.Second
	}
	return b.Add(&EtcdSource{Options: opts})
}

// Build 构建配置
func (b *ConfigurationBuilder) Build() (Configuration, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	config := &configuration{
		data: make(map[string]any),
	}

	// 按顺序加载所有配置源（后面的会覆盖前面的）
	for _, source := range b.sources {
		data, err := source.Load()
		if err != nil {
			return nil, fmt.Errorf("failed to load config source %s: %w", source.Name(), err)
		}

		// 合并配置
		mergeMaps(config.data, data)
	}

	return config, nil
}

// configuration 配置实现
type configuration struct {
	data map[string]any
	mu   sync.RWMutex
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

	// 使用 JSON 序列化/反序列化进行绑定
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

// JsonFileSource JSON 文件配置源
type JsonFileSource struct {
	Path     string
	Optional bool
}

func (s *JsonFileSource) Name() string {
	return fmt.Sprintf("JsonFile(%s)", s.Path)
}

func (s *JsonFileSource) Load() (map[string]any, error) {
	data, err := os.ReadFile(s.Path)
	if err != nil {
		if s.Optional && os.IsNotExist(err) {
			return make(map[string]any), nil
		}
		return nil, err
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return result, nil
}

// YamlFileSource YAML 文件配置源
type YamlFileSource struct {
	Path     string
	Optional bool
}

func (s *YamlFileSource) Name() string {
	return fmt.Sprintf("YamlFile(%s)", s.Path)
}

func (s *YamlFileSource) Load() (map[string]any, error) {
	data, err := os.ReadFile(s.Path)
	if err != nil {
		if s.Optional && os.IsNotExist(err) {
			return make(map[string]any), nil
		}
		return nil, err
	}

	var result map[string]any
	if err := yaml.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	return result, nil
}

// EnvironmentVariableSource 环境变量配置源
type EnvironmentVariableSource struct {
	Prefix string
}

func (s *EnvironmentVariableSource) Name() string {
	return fmt.Sprintf("EnvironmentVariables(%s)", s.Prefix)
}

func (s *EnvironmentVariableSource) Load() (map[string]any, error) {
	result := make(map[string]any)

	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key, value := parts[0], parts[1]

		// 检查前缀
		if s.Prefix != "" && !strings.HasPrefix(key, s.Prefix) {
			continue
		}

		// 移除前缀
		if s.Prefix != "" {
			key = strings.TrimPrefix(key, s.Prefix)
		}

		// 转换为小写（保持与 JSON 配置一致）
		key = strings.ToLower(key)

		// 将 _ 转换为 :
		key = strings.ReplaceAll(key, "_", ":") // 设置嵌套值
		setNestedValue(result, key, value)
	}

	return result, nil
}

// InMemorySource 内存配置源
type InMemorySource struct {
	Data map[string]any
}

func (s *InMemorySource) Name() string {
	return "InMemory"
}

func (s *InMemorySource) Load() (map[string]any, error) {
	// 返回副本
	result := make(map[string]any)
	mergeMaps(result, s.Data)
	return result, nil
}

// setNestedValue 设置嵌套值
func setNestedValue(data map[string]any, path string, value any) {
	parts := strings.Split(path, ":")
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
		// 尝试转换为整数
		if intValue, err := strconv.Atoi(strValue); err == nil {
			value = intValue
		} else if floatValue, err := strconv.ParseFloat(strValue, 64); err == nil {
			// 尝试转换为浮点数
			value = floatValue
		} else if boolValue, err := strconv.ParseBool(strValue); err == nil {
			// 尝试转换为布尔值
			value = boolValue
		}
		// 否则保持为字符串
	}

	current[parts[len(parts)-1]] = value
}

// EtcdSource etcd 配置源
type EtcdSource struct {
	Options EtcdOptions
}

func (s *EtcdSource) Name() string {
	return fmt.Sprintf("Etcd(%v)", s.Options.Endpoints)
}

func (s *EtcdSource) Load() (map[string]any, error) {
	// 创建 etcd 客户端
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   s.Options.Endpoints,
		Username:    s.Options.Username,
		Password:    s.Options.Password,
		DialTimeout: s.Options.DialTimeout,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create etcd client: %w", err)
	}
	defer cli.Close()

	// 创建超时上下文
	ctx, cancel := context.WithTimeout(context.Background(), s.Options.Timeout)
	defer cancel()

	// 获取指定前缀下的所有配置
	prefix := s.Options.Prefix
	if prefix == "" {
		prefix = "/"
	}

	resp, err := cli.Get(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		return nil, fmt.Errorf("failed to get config from etcd: %w", err)
	}

	result := make(map[string]any)

	// 处理每个键值对
	for _, kv := range resp.Kvs {
		key := string(kv.Key)
		value := string(kv.Value)

		// 移除前缀
		if s.Options.Prefix != "" {
			key = strings.TrimPrefix(key, s.Options.Prefix)
		}

		// 移除开头的斜杠
		key = strings.TrimPrefix(key, "/")

		if key == "" {
			continue
		}

		// 将路径分隔符 / 转换为 :
		key = strings.ReplaceAll(key, "/", ":")

		// 尝试解析为 JSON
		var jsonValue any
		if err := json.Unmarshal([]byte(value), &jsonValue); err == nil {
			// 成功解析为 JSON
			if m, ok := jsonValue.(map[string]any); ok {
				// 如果是 JSON 对象，需要展开
				setNestedValue(result, key, m)
			} else {
				// 普通 JSON 值
				setNestedValue(result, key, jsonValue)
			}
		} else {
			// 尝试解析为 YAML
			var yamlValue any
			if err := yaml.Unmarshal([]byte(value), &yamlValue); err == nil {
				if m, ok := yamlValue.(map[string]any); ok {
					// 如果是 YAML 对象，需要展开
					setNestedValue(result, key, m)
				} else {
					setNestedValue(result, key, yamlValue)
				}
			} else {
				// 作为普通字符串处理
				setNestedValue(result, key, value)
			}
		}
	}

	return result, nil
}
