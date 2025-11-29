# Web 开发 (Gin)

Web 模块基于 [Gin](https://github.com/gin-gonic/gin) 框架封装，提供了自动化的 Controller 注册和生命周期管理。

## 启用 Web 模块

```go
app.Run(
    web.New(
        web.WithPort(8080),
        web.WithControllers(NewUserController, NewOrderController),
    ),
)
```

## 定义 Controller

Controller 是一个普通的 Go 结构体，只需实现 `Register(gin.IRouter)` 方法（或 `MountRoutes`，根据版本）。

**建议使用构造函数注入**：

```go
type UserController struct {
    Service *UserService
}

// 构造函数：DI 容器会自动注入 UserService
func NewUserController(svc *UserService) *UserController {
    return &UserController{Service: svc}
}

// 注册路由
func (c *UserController) MountRoutes(r gin.IRouter) {
    // 基础路由
    r.GET("/users", c.List)
    
    // 路由组
    v1 := r.Group("/api/v1")
    {
        v1.POST("/users", c.Create)
        v1.GET("/users/:id", c.Get)
    }
}

func (c *UserController) List(ctx *gin.Context) {
    ctx.JSON(200, gin.H{"data": "list"})
}
```

## 中间件 (Middleware)

### 全局中间件

在 `web.New` 中通过配置函数添加：

```go
web.New(
    func(b *web.Builder) {
        b.Use(gin.Logger())
        b.Use(gin.Recovery())
        b.Use(MyCustomMiddleware())
    },
)
```

### 路由组中间件

在 `Controller` 中定义：

```go
func (c *AdminController) MountRoutes(r gin.IRouter) {
    // 创建带鉴权的组
    admin := r.Group("/admin", AuthMiddleware())
    
    admin.GET("/dashboard", c.Dashboard)
}
```

## 请求处理

完全兼容 Gin 的所有功能。

### 参数绑定

```go
type CreateUserReq struct {
    Username string `json:"username" binding:"required"`
    Email    string `json:"email" binding:"required,email"`
}

func (c *UserController) Create(ctx *gin.Context) {
    var req CreateUserReq
    if err := ctx.ShouldBindJSON(&req); err != nil {
        ctx.JSON(400, gin.H{"error": err.Error()})
        return
    }
    
    // 调用业务逻辑
    // c.Service.Create(req.Username, req.Email)
}
```

## 高级配置

### 自定义 Gin Engine

如果你需要完全控制 Gin Engine（例如设置 mode，添加自定义 Render 等）：

```go
web.New(
    func(b *web.Builder) {
        engine := b.Engine()
        engine.SetTrustedProxies([]string{"127.0.0.1"})
        // ...
    },
)
```

