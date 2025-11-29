# GoCRUD App Framework

一个基于 Go 语言的现代化、微内核、插件化应用开发框架。它旨在通过**依赖注入 (DI)**、**模块化设计**和**声明式配置**来简化构建可维护、可测试的后端服务。

## 📚 文档

**[📖 点击查看完整文档 (Documentation)](docs/README.md)**

*   [核心概念](docs/core.md)
*   [Web 开发指南](docs/web.md)
*   [配置系统](docs/config.md)
*   [数据库与事务](docs/database.md)
*   [常用组件 (Redis, Cron...)](docs/components.md)

## 🚀 快速开始

### 1. 安装

```bash
go get github.com/gocrud/app
```

### 2. Hello World

```go
package main

import (
	"github.com/gin-gonic/gin"
	"github.com/gocrud/app"
	"github.com/gocrud/app/web"
)

type HelloController struct{}

func (c *HelloController) MountRoutes(r gin.IRouter) {
	r.GET("/hello", func(ctx *gin.Context) {
		ctx.JSON(200, gin.H{"message": "Hello, GoCRUD!"})
	})
}

func main() {
	app.Run(
		web.New(
			web.WithControllers(&HelloController{}),
			web.WithPort(8080),
		),
	)
}
```

运行并访问 `http://localhost:8080/hello`。

## ✨ 核心特性

*   **微内核架构**: 极简核心，一切皆插件。
*   **依赖注入**: 支持构造函数注入、字段注入、接口绑定。
*   **生命周期**: 自动管理组件启动与关闭顺序。
*   **模块化**: 轻松拆分业务模块。

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

## 📄 License

MIT
