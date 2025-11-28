package etcd

import (
	"fmt"
	"sync"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

// EtcdClientOptions etcd 客户端配置选项
type EtcdClientOptions struct {
	Name               string        // 客户端名称
	Endpoints          []string      // etcd 服务器地址列表
	DialTimeout        time.Duration // 连接超时时间
	Username           string        // 用户名（可选）
	Password           string        // 密码（可选）
	AutoSyncInterval   time.Duration // 自动同步间隔（可选）
	MaxCallSendMsgSize int           // 最大发送消息大小（可选）
	MaxCallRecvMsgSize int           // 最大接收消息大小（可选）
}

// NewDefaultOptions 创建默认配置
func NewDefaultOptions(name string) *EtcdClientOptions {
	return &EtcdClientOptions{
		Name:        name,
		Endpoints:   []string{"localhost:2379"},
		DialTimeout: 5 * time.Second,
	}
}

// Validate 验证配置
func (o *EtcdClientOptions) Validate() error {
	if o.Name == "" {
		return fmt.Errorf("etcd client name is required")
	}
	if len(o.Endpoints) == 0 {
		return fmt.Errorf("etcd endpoints are required")
	}
	if o.DialTimeout <= 0 {
		return fmt.Errorf("etcd dial timeout must be positive")
	}
	return nil
}

// EtcdClientFactory etcd 客户端工厂
type EtcdClientFactory struct {
	clients map[string]*clientv3.Client
	mu      sync.RWMutex
}

// NewEtcdClientFactory 创建客户端工厂
func NewEtcdClientFactory() *EtcdClientFactory {
	return &EtcdClientFactory{
		clients: make(map[string]*clientv3.Client),
	}
}

// Register 注册 etcd 客户端
func (f *EtcdClientFactory) Register(opts EtcdClientOptions) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	// 检查是否已存在
	if _, exists := f.clients[opts.Name]; exists {
		return fmt.Errorf("etcd client '%s' already registered", opts.Name)
	}

	// 创建客户端配置
	config := clientv3.Config{
		Endpoints:   opts.Endpoints,
		DialTimeout: opts.DialTimeout,
	}

	// 设置认证信息
	if opts.Username != "" {
		config.Username = opts.Username
		config.Password = opts.Password
	}

	// 设置自动同步间隔
	if opts.AutoSyncInterval > 0 {
		config.AutoSyncInterval = opts.AutoSyncInterval
	}

	// 设置消息大小限制
	if opts.MaxCallSendMsgSize > 0 {
		config.MaxCallSendMsgSize = opts.MaxCallSendMsgSize
	}
	if opts.MaxCallRecvMsgSize > 0 {
		config.MaxCallRecvMsgSize = opts.MaxCallRecvMsgSize
	}

	// 创建客户端
	client, err := clientv3.New(config)
	if err != nil {
		return fmt.Errorf("failed to create etcd client: %w", err)
	}

	// 保存客户端
	f.clients[opts.Name] = client

	return nil
}

// Each 遍历所有客户端
func (f *EtcdClientFactory) Each(fn func(name string, client *clientv3.Client)) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	for name, client := range f.clients {
		fn(name, client)
	}
}

// Close 关闭所有 etcd 客户端
func (f *EtcdClientFactory) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	var errs []error
	for name, client := range f.clients {
		if err := client.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close client '%s': %w", name, err))
		}
	}

	// 清空客户端列表
	f.clients = make(map[string]*clientv3.Client)

	if len(errs) > 0 {
		return fmt.Errorf("errors closing etcd clients: %v", errs)
	}

	return nil
}
