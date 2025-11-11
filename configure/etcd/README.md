# Etcd 配置模块

Etcd 配置模块提供了对 etcd 分布式键值存储的集成支持，支持多客户端配置、依赖注入和自动资源管理。

## 功能特性

- ✅ 支持配置多个 etcd 客户端
- ✅ 自动注入到 DI 容器
- ✅ 支持连接认证（用户名/密码）
- ✅ 灵活的客户端配置选项
- ✅ 应用关闭时自动清理资源
- ✅ 基于工厂模式管理多客户端

## 使用示例

### 1. 基本使用（单个 Etcd 客户端）

```go
package main

import (
	"github.com/gocrud/app/configure/etcd"
	"github.com/gocrud/app/core"
)

func main() {
	builder := core.NewApplicationBuilder()
	
	// 配置 Etcd
	builder.Configure(etcd.Configure(func(b *etcd.Builder) {
		b.AddClient("default", func(opts *etcd.EtcdClientOptions) {
			opts.Endpoints = []string{"localhost:2379"}
			opts.DialTimeout = 5 * time.Second
		})
	}))
	
	app := builder.Build()
	app.Run()
}
```

### 2. 多个 Etcd 客户端

```go
builder.Configure(etcd.Configure(func(b *etcd.Builder) {
	// 默认客户端
	b.AddClient("default", func(opts *etcd.EtcdClientOptions) {
		opts.Endpoints = []string{"localhost:2379"}
		opts.DialTimeout = 5 * time.Second
	})
	
	// 配置中心客户端
	b.AddClient("config", func(opts *etcd.EtcdClientOptions) {
		opts.Endpoints = []string{"etcd-config:2379", "etcd-config:2380"}
		opts.DialTimeout = 10 * time.Second
		opts.AutoSyncInterval = 30 * time.Second
	})
	
	// 服务发现客户端（带认证）
	b.AddClient("discovery", func(opts *etcd.EtcdClientOptions) {
		opts.Endpoints = []string{"etcd-discovery:2379"}
		opts.Username = "admin"
		opts.Password = "secret"
		opts.DialTimeout = 5 * time.Second
	})
}))
```

### 3. 在服务中使用 Etcd

```go
package myservice

import (
	"context"
	"time"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type ConfigService struct {
	etcdClient *clientv3.Client
}

// 通过 DI 注入默认 Etcd 客户端
func NewConfigService(client *clientv3.Client) *ConfigService {
	return &ConfigService{
		etcdClient: client,
	}
}

func (s *ConfigService) Put(key, value string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	_, err := s.etcdClient.Put(ctx, key, value)
	return err
}

func (s *ConfigService) Get(key string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	resp, err := s.etcdClient.Get(ctx, key)
	if err != nil {
		return "", err
	}
	
	if len(resp.Kvs) == 0 {
		return "", nil
	}
	
	return string(resp.Kvs[0].Value), nil
}

func (s *ConfigService) Watch(key string) clientv3.WatchChan {
	return s.etcdClient.Watch(context.Background(), key)
}
```

### 4. 使用 EtcdClientFactory 管理多个客户端

```go
package myservice

import (
	"context"
	"time"
	"github.com/gocrud/app/configure/etcd"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type DistributedService struct {
	factory *etcd.EtcdClientFactory
}

// 注入 EtcdClientFactory
func NewDistributedService(factory *etcd.EtcdClientFactory) *DistributedService {
	return &DistributedService{
		factory: factory,
	}
}

func (s *DistributedService) SaveConfig(key, value string) error {
	// 使用配置专用客户端
	configClient, err := s.factory.Get("config")
	if err != nil {
		return err
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	_, err = configClient.Put(ctx, key, value)
	return err
}

func (s *DistributedService) RegisterService(name, addr string) error {
	// 使用服务发现客户端
	discoveryClient, err := s.factory.Get("discovery")
	if err != nil {
		return err
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	// 创建租约
	lease, err := discoveryClient.Grant(ctx, 10)
	if err != nil {
		return err
	}
	
	// 注册服务
	key := "/services/" + name
	_, err = discoveryClient.Put(ctx, key, addr, clientv3.WithLease(lease.ID))
	return err
}
```

### 5. 分布式锁实现

```go
package myservice

import (
	"context"
	"time"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
)

type LockService struct {
	etcdClient *clientv3.Client
}

func NewLockService(client *clientv3.Client) *LockService {
	return &LockService{
		etcdClient: client,
	}
}

func (s *LockService) AcquireLock(lockKey string, ttl int) (*concurrency.Mutex, error) {
	ctx := context.Background()
	
	// 创建会话
	session, err := concurrency.NewSession(s.etcdClient, concurrency.WithTTL(ttl))
	if err != nil {
		return nil, err
	}
	
	// 创建互斥锁
	mutex := concurrency.NewMutex(session, lockKey)
	
	// 获取锁
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	
	if err := mutex.Lock(ctx); err != nil {
		session.Close()
		return nil, err
	}
	
	return mutex, nil
}

func (s *LockService) ReleaseLock(mutex *concurrency.Mutex) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	return mutex.Unlock(ctx)
}
```

## 配置选项

### EtcdClientOptions

| 字段                | 类型            | 必填 | 默认值                | 说明                     |
|---------------------|----------------|------|----------------------|-------------------------|
| Name                | string         | 是   | -                    | 客户端名称               |
| Endpoints           | []string       | 是   | ["localhost:2379"]   | etcd 服务器地址列表      |
| DialTimeout         | time.Duration  | 否   | 5s                   | 连接超时时间             |
| Username            | string         | 否   | ""                   | 用户名（认证）           |
| Password            | string         | 否   | ""                   | 密码（认证）             |
| AutoSyncInterval    | time.Duration  | 否   | 0                    | 自动同步间隔             |
| MaxCallSendMsgSize  | int            | 否   | 0                    | 最大发送消息大小（字节）  |
| MaxCallRecvMsgSize  | int            | 否   | 0                    | 最大接收消息大小（字节）  |

## 常见用例

### 配置中心

```go
type ConfigCenter struct {
	client *clientv3.Client
}

func (c *ConfigCenter) LoadConfig(prefix string) (map[string]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	resp, err := c.client.Get(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}
	
	configs := make(map[string]string)
	for _, kv := range resp.Kvs {
		configs[string(kv.Key)] = string(kv.Value)
	}
	
	return configs, nil
}

func (c *ConfigCenter) WatchConfig(key string, callback func(string)) {
	watchChan := c.client.Watch(context.Background(), key)
	
	for watchResp := range watchChan {
		for _, event := range watchResp.Events {
			callback(string(event.Kv.Value))
		}
	}
}
```

### 服务发现

```go
type ServiceDiscovery struct {
	client *clientv3.Client
}

func (s *ServiceDiscovery) Register(serviceName, serviceAddr string, ttl int64) error {
	ctx := context.Background()
	
	// 创建租约
	lease, err := s.client.Grant(ctx, ttl)
	if err != nil {
		return err
	}
	
	// 注册服务
	key := "/services/" + serviceName
	_, err = s.client.Put(ctx, key, serviceAddr, clientv3.WithLease(lease.ID))
	if err != nil {
		return err
	}
	
	// 保持租约
	keepAliveChan, err := s.client.KeepAlive(ctx, lease.ID)
	if err != nil {
		return err
	}
	
	// 处理续约响应
	go func() {
		for range keepAliveChan {
			// 续约成功
		}
	}()
	
	return nil
}

func (s *ServiceDiscovery) Discover(serviceName string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	prefix := "/services/" + serviceName
	resp, err := s.client.Get(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}
	
	var services []string
	for _, kv := range resp.Kvs {
		services = append(services, string(kv.Value))
	}
	
	return services, nil
}
```

## 最佳实践

### 1. 使用上下文超时

始终为 etcd 操作设置超时，避免永久阻塞：

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

_, err := client.Put(ctx, key, value)
```

### 2. 错误处理

正确处理 etcd 错误：

```go
import "google.golang.org/grpc/codes"
import "google.golang.org/grpc/status"

_, err := client.Get(ctx, key)
if err != nil {
	if status.Code(err) == codes.DeadlineExceeded {
		// 超时处理
	} else if status.Code(err) == codes.Unavailable {
		// 服务不可用
	}
}
```

### 3. 使用租约管理临时数据

对于需要自动过期的数据，使用租约：

```go
lease, err := client.Grant(ctx, 60) // 60秒租约
if err != nil {
	return err
}

_, err = client.Put(ctx, key, value, clientv3.WithLease(lease.ID))
```

### 4. 批量操作

使用事务进行批量操作：

```go
txn := client.Txn(ctx)
txn.Then(
	clientv3.OpPut("key1", "value1"),
	clientv3.OpPut("key2", "value2"),
	clientv3.OpPut("key3", "value3"),
)
_, err := txn.Commit()
```

## 依赖项

需要在 `go.mod` 中添加 etcd 客户端依赖：

```bash
go get go.etcd.io/etcd/client/v3
```

## 注意事项

1. **连接池管理**：etcd 客户端内部维护连接池，无需手动管理
2. **资源清理**：应用关闭时会自动关闭所有 etcd 客户端连接
3. **集群配置**：生产环境建议配置多个 Endpoints 实现高可用
4. **认证安全**：使用认证时，避免在代码中硬编码密码，建议从环境变量或配置文件读取
5. **Watch 性能**：避免创建过多的 Watch，可能影响性能

## 相关链接

- [etcd 官方文档](https://etcd.io/docs/)
- [etcd Go 客户端](https://github.com/etcd-io/etcd/tree/main/client/v3)
- [etcd API 参考](https://etcd.io/docs/v3.5/learning/api/)
