package mongodb

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gocrud/mgo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// MongoOptions MongoDB 客户端配置选项
type MongoOptions struct {
	Name        string
	Uri         string
	Username    string
	Password    string
	MaxPoolSize uint64
	MinPoolSize uint64
	Timeout     time.Duration
}

// NewDefaultOptions 创建默认配置
func NewDefaultOptions(name string, uri string) *MongoOptions {
	return &MongoOptions{
		Name:        name,
		Uri:         uri,
		MaxPoolSize: 100,
		MinPoolSize: 5,
		Timeout:     10 * time.Second,
	}
}

// Validate 验证配置
func (o *MongoOptions) Validate() error {
	if o.Name == "" {
		return fmt.Errorf("mongo client name is required")
	}
	if o.Uri == "" {
		return fmt.Errorf("mongo uri is required")
	}
	return nil
}

// MongoFactory MongoDB 客户端工厂
type MongoFactory struct {
	clients map[string]*mgo.Client
	mu      sync.RWMutex
}

// NewMongoFactory 创建客户端工厂
func NewMongoFactory() *MongoFactory {
	return &MongoFactory{
		clients: make(map[string]*mgo.Client),
	}
}

// Register 注册 MongoDB 客户端
func (f *MongoFactory) Register(opts MongoOptions) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if _, exists := f.clients[opts.Name]; exists {
		return fmt.Errorf("mongo client '%s' already registered", opts.Name)
	}

	// 构建配置
	clientOpts := options.Client()
	if opts.Username != "" || opts.Password != "" {
		clientOpts.SetAuth(options.Credential{
			Username: opts.Username,
			Password: opts.Password,
		})
	}
	if opts.MaxPoolSize > 0 {
		clientOpts.SetMaxPoolSize(opts.MaxPoolSize)
	}
	if opts.MinPoolSize > 0 {
		clientOpts.SetMinPoolSize(opts.MinPoolSize)
	}
	if opts.Timeout > 0 {
		clientOpts.SetConnectTimeout(opts.Timeout)
	}

	// 创建连接上下文
	ctx, cancel := context.WithTimeout(context.Background(), opts.Timeout)
	defer cancel()

	// 创建客户端
	// 注意：这里假设 mgo.NewClient 的签名是 (ctx, uri, ...options)
	client, err := mgo.NewClient(ctx, opts.Uri, clientOpts)
	if err != nil {
		return fmt.Errorf("failed to create mongo client '%s': %w", opts.Name, err)
	}

	f.clients[opts.Name] = client
	return nil
}

// Each 遍历所有客户端
func (f *MongoFactory) Each(fn func(name string, client *mgo.Client)) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	for name, client := range f.clients {
		fn(name, client)
	}
}

// Close 关闭所有客户端
func (f *MongoFactory) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	var errs []error
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for name, client := range f.clients {
		if err := client.Disconnect(ctx); err != nil {
			errs = append(errs, fmt.Errorf("failed to close client '%s': %w", name, err))
		}
	}

	f.clients = make(map[string]*mgo.Client)

	if len(errs) > 0 {
		return fmt.Errorf("errors closing mongo clients: %v", errs)
	}
	return nil
}
