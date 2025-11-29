# 常用组件 (Components)

框架内置了对常用基础设施组件的支持。

## Redis

基于 [go-redis/v9](https://github.com/redis/go-redis) 封装。

### 启用

```go
import "github.com/gocrud/app/redis"

app.Run(
    redis.New(
        redis.WithClient("cache", &redis.Options{
            Addr: "localhost:6379",
            Password: "",
            DB: 0,
        }),
    ),
)
```

### 使用

```go
type CacheService struct {
    // 注入名为 "cache" 的客户端
    Redis *redis.Client `di:"name=cache"`
}

func (s *CacheService) Set(key string, val string) {
    s.Redis.Set(context.Background(), key, val, 0)
}
```

---

## Cron (定时任务)

基于 [robfig/cron/v3](https://github.com/robfig/cron) 封装。会自动随应用启动和停止。

### 启用

```go
import "github.com/gocrud/app/cron"

app.Run(
    cron.New(
        // 添加任务
        cron.WithJob("@every 1m", NewHealthCheckJob),
    ),
)
```

### 定义任务

任务可以是任何实现了 `cron.Job` 接口的对象，或者是 `func()`。
如果是结构体，支持依赖注入。

```go
type HealthCheckJob struct {
    DB *gorm.DB `di:""`
}

func NewHealthCheckJob(db *gorm.DB) *HealthCheckJob {
    return &HealthCheckJob{DB: db}
}

func (j *HealthCheckJob) Run() {
    // 执行任务逻辑
    fmt.Println("Checking DB health...")
}
```

---

## Etcd

基于 [etcd/client/v3](https://github.com/etcd-io/etcd/tree/main/client/v3) 封装。

### 启用

```go
import "github.com/gocrud/app/etcd"

app.Run(
    etcd.New(
        etcd.WithClient("registry", etcd.Config{
            Endpoints: []string{"localhost:2379"},
        }),
    ),
)
```

### 使用

```go
type DiscoveryService struct {
    Client *clientv3.Client `di:"name=registry"`
}
```

---

## MongoDB

基于 [mgo](github.com/gocrud/mgo) (官方 driver 包装) 封装。

### 启用

```go
import "github.com/gocrud/app/mongodb"

app.Run(
    mongodb.New(
        mongodb.WithDatabase("main", "mongodb://localhost:27017", "mydb"),
    ),
)
```

### 使用

```go
type DocRepo struct {
    DB *mongo.Database `di:"name=main"`
}
```

