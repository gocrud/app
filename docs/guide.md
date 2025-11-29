# 业务开发指南 (Development Guide)

本指南介绍如何使用框架构建可维护、可测试的业务应用。

## 分层架构 (Layered Architecture)

推荐采用经典的 **Controller-Service-Repository** 三层架构。

1.  **Controller (Presentation Layer)**: 处理 HTTP 请求，参数校验，响应格式化。
2.  **Service (Business Logic Layer)**: 核心业务逻辑，事务控制，领域模型操作。
3.  **Repository (Data Access Layer)**: 数据库 CRUD 操作，屏蔽存储细节。

### 目录结构示例

```
my-app/
├── cmd/
│   └── server/
│       └── main.go        # 入口文件
├── config/
│   └── config.yaml        # 配置文件
├── internal/
│   ├── model/             # 领域实体 (POJO)
│   │   └── user.go
│   ├── repository/        # 数据访问层
│   │   └── user_repo.go
│   ├── service/           # 业务逻辑层
│   │   └── user_service.go
│   ├── controller/        # 接口层
│   │   └── user_controller.go
│   └── worker/            # 后台任务
│       └── cleanup_worker.go
└── go.mod
```

### 代码实现示例

#### 1. Model

```go
// internal/model/user.go
package model

import "gorm.io/gorm"

type User struct {
    gorm.Model
    Username string
    Email    string
}
```

#### 2. Repository

```go
// internal/repository/user_repo.go
package repository

import (
    "gorm.io/gorm"
    "my-app/internal/model"
)

type UserRepo struct {
    DB *gorm.DB `di:""`
}

func NewUserRepo(db *gorm.DB) *UserRepo {
    return &UserRepo{DB: db}
}

func (r *UserRepo) Create(user *model.User) error {
    return r.DB.Create(user).Error
}
```

#### 3. Service

```go
// internal/service/user_service.go
package service

import (
    "errors"
    "my-app/internal/model"
    "my-app/internal/repository"
)

type UserService struct {
    Repo *repository.UserRepo `di:""`
}

func NewUserService(repo *repository.UserRepo) *UserService {
    return &UserService{Repo: repo}
}

func (s *UserService) Register(username, email string) error {
    if username == "" {
        return errors.New("username required")
    }
    return s.Repo.Create(&model.User{Username: username, Email: email})
}
```

#### 4. Controller

```go
// internal/controller/user_controller.go
package controller

import (
    "github.com/gin-gonic/gin"
    "my-app/internal/service"
)

type UserController struct {
    Service *service.UserService
}

func NewUserController(svc *service.UserService) *UserController {
    return &UserController{Service: svc}
}

func (c *UserController) MountRoutes(r gin.IRouter) {
    r.POST("/users", c.Register)
}

func (c *UserController) Register(ctx *gin.Context) {
    // ... Bind JSON ...
    // err := c.Service.Register(req.Username, req.Email)
    // ... Response ...
}
```

#### 5. Main

```go
// cmd/server/main.go
package main

import (
    "context"
    "time"
    "fmt"
    "github.com/gocrud/app"
    "github.com/gocrud/app/core"
    "github.com/gocrud/app/web"
    "my-app/internal/controller"
    "my-app/internal/repository"
    "my-app/internal/service"
    "my-app/internal/worker"
)

func main() {
    app.Run(
        // ... Config & DB ...
        
        web.New(
            web.WithControllers(controller.NewUserController),
        ),
        
        // 注册标准 HostedService
        core.WithHostedService(worker.NewCleanupWorker),
        
        // 注册简单函数 Worker
        core.WithWorker(func(ctx context.Context) error {
            for {
                select {
                case <-ctx.Done():
                    return nil
                case <-time.After(5 * time.Second):
                    fmt.Println("Working...")
                }
            }
        }),
        
        // 注册业务服务
        func(rt *core.Runtime) error {
            rt.Provide(repository.NewUserRepo)
            rt.Provide(service.NewUserService)
            return nil
        },
    )
}
```

## 后台任务 (Worker)

框架提供了统一的机制来管理后台服务的生命周期。无论是 Web 服务器、消息队列消费者还是定时任务，都通过 `core.HostedService` 接口进行抽象。

框架会自动管理这些服务的启动（Goroutine 调度）和停止（优雅关闭）。

### 方式一：实现 HostedService (推荐)

适用于有状态、需要依赖注入或复杂启动/停止逻辑的服务。

**接口定义**:
```go
type HostedService interface {
    // Start 启动服务
    // 框架会在独立的 Goroutine 中调用此方法，因此【建议阻塞】运行主循环。
    // 如果此方法返回 error，App 会记录错误并触发整个应用的优雅关闭。
    Start(ctx context.Context) error

    // Stop 停止服务
    // 在应用关闭时调用。通常用于通知 Start 方法退出 (如关闭 channel)。
    // 必须支持通过 ctx 进行超时控制。
    Stop(ctx context.Context) error
}
```

**实现示例**:

```go
package worker

import (
    "context"
    "fmt"
    "time"
    
    "github.com/gocrud/app/core"
    "gorm.io/gorm"
)

// 确保实现了 HostedService 接口
var _ core.HostedService = (*CleanupWorker)(nil)

type CleanupWorker struct {
    DB     *gorm.DB `di:""` // 支持自动注入
    stopCh chan struct{}
}

func NewCleanupWorker() *CleanupWorker {
    return &CleanupWorker{
        stopCh: make(chan struct{}),
    }
}

// Start 启动服务 (阻塞模式)
func (w *CleanupWorker) Start(ctx context.Context) error {
    fmt.Println("CleanupWorker started.")
    ticker := time.NewTicker(1 * time.Hour)
    defer ticker.Stop()

    for {
        select {
        case <-w.stopCh: // 收到停止信号
            fmt.Println("CleanupWorker stopped.")
            return nil
        case <-ctx.Done(): // 收到上下文取消信号 (双重保险)
            return nil
        case <-ticker.C:
            fmt.Println("Performing cleanup...")
            // w.DB.Exec(...)
        }
    }
}

// Stop 停止服务
func (w *CleanupWorker) Stop(ctx context.Context) error {
    fmt.Println("Stopping CleanupWorker...")
    close(w.stopCh) // 通知 Start 退出
    return nil
}
```

**注册方式**:

使用 `core.WithHostedService` 注册构造函数。

```go
// main.go
app.Run(
    // ... 其他模块 ...
    core.WithHostedService(worker.NewCleanupWorker),
)
```

### 方式二：使用 WithWorker (简单函数)

对于不需要复杂状态管理的简单后台任务，可以直接注册一个函数。该函数会在独立的 Goroutine 中运行。

```go
// main.go
app.Run(
    core.WithWorker(func(ctx context.Context) error {
        fmt.Println("Simple worker started")
        ticker := time.NewTicker(5 * time.Second)
        defer ticker.Stop()
        
        for {
            select {
            case <-ctx.Done(): // 监听退出信号
                fmt.Println("Simple worker stopping")
                return nil
            case <-ticker.C:
                fmt.Println("Tick...")
            }
        }
    }),
)
```
