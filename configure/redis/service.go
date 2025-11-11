package redis

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisClientOptions Redis 客户端配置选项
type RedisClientOptions struct {
	Name         string        // 客户端名称
	Addr         string        // Redis 服务器地址 (host:port)
	Password     string        // 密码（可选）
	DB           int           // 数据库编号
	DialTimeout  time.Duration // 连接超时时间
	ReadTimeout  time.Duration // 读取超时时间
	WriteTimeout time.Duration // 写入超时时间
	PoolSize     int           // 连接池大小
	MinIdleConns int           // 最小空闲连接数
	MaxRetries   int           // 最大重试次数
}

// NewDefaultOptions 创建默认配置
func NewDefaultOptions(name string) *RedisClientOptions {
	return &RedisClientOptions{
		Name:         name,
		Addr:         "localhost:6379",
		DB:           0,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     10,
		MinIdleConns: 5,
		MaxRetries:   3,
	}
}

// Validate 验证配置
func (o *RedisClientOptions) Validate() error {
	if o.Name == "" {
		return fmt.Errorf("redis client name is required")
	}
	if o.Addr == "" {
		return fmt.Errorf("redis address is required")
	}
	if o.DB < 0 {
		return fmt.Errorf("redis database number must be non-negative")
	}
	if o.DialTimeout <= 0 {
		return fmt.Errorf("redis dial timeout must be positive")
	}
	return nil
}

// RedisClientFactory Redis 客户端工厂
type RedisClientFactory struct {
	clients map[string]*redis.Client
	mu      sync.RWMutex
}

// NewRedisClientFactory 创建客户端工厂
func NewRedisClientFactory() *RedisClientFactory {
	return &RedisClientFactory{
		clients: make(map[string]*redis.Client),
	}
}

// Register 注册 Redis 客户端
func (f *RedisClientFactory) Register(opts RedisClientOptions) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	// 检查是否已存在
	if _, exists := f.clients[opts.Name]; exists {
		return fmt.Errorf("redis client '%s' already registered", opts.Name)
	}

	// 创建客户端配置
	config := &redis.Options{
		Addr:         opts.Addr,
		Password:     opts.Password,
		DB:           opts.DB,
		DialTimeout:  opts.DialTimeout,
		ReadTimeout:  opts.ReadTimeout,
		WriteTimeout: opts.WriteTimeout,
		PoolSize:     opts.PoolSize,
		MinIdleConns: opts.MinIdleConns,
		MaxRetries:   opts.MaxRetries,
	}

	// 创建客户端
	client := redis.NewClient(config)

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), opts.DialTimeout)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return fmt.Errorf("failed to connect to redis: %w", err)
	}

	// 保存客户端
	f.clients[opts.Name] = client

	return nil
}

// Get 获取指定名称的 Redis 客户端
func (f *RedisClientFactory) Get(name string) (*redis.Client, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	client, exists := f.clients[name]
	if !exists {
		return nil, fmt.Errorf("redis client '%s' not found", name)
	}

	return client, nil
}

// Close 关闭所有 Redis 客户端
func (f *RedisClientFactory) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	var errs []error
	for name, client := range f.clients {
		if err := client.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close client '%s': %w", name, err))
		}
	}

	// 清空客户端列表
	f.clients = make(map[string]*redis.Client)

	if len(errs) > 0 {
		return fmt.Errorf("errors closing redis clients: %v", errs)
	}

	return nil
}
