# Database Configuration Module

`configure/database` 模块提供了基于 GORM 的数据库集成支持，具备多实例管理、依赖注入（DI）集成、连接池配置和自动迁移等功能。

## 特性

*   **多实例支持**：可以同时配置多个数据库连接（如主库、从库、日志库）。
*   **依赖注入集成**：自动将 `*gorm.DB` 注册到 DI 容器，支持命名注入 (`di:"name"`) 和可选注入 (`di:"?"`)。
*   **驱动解耦**：通过 `gorm.Dialector` 接口支持任意 GORM 兼容驱动（MySQL, PostgreSQL, SQLite, SQLServer 等），无需框架硬编码依赖。
*   **连接池管理**：内置 `SetMaxIdleConns`、`SetMaxOpenConns` 等配置。
*   **自动迁移**：支持在启动时自动执行 `AutoMigrate`。
*   **生命周期管理**：应用停止时自动关闭所有数据库连接。

## 安装依赖

由于采用了驱动解耦设计，你需要手动引入 GORM 核心库和你需要的驱动库：

```bash
go get gorm.io/gorm
go get gorm.io/driver/mysql    # 如果使用 MySQL
go get gorm.io/driver/postgres # 如果使用 PostgreSQL
# ... 其他驱动
```

## 使用示例

### 1. 配置数据库

在你的应用启动代码中（通常是 `main.go`）：

```go
package main

import (
    "github.com/gocrud/app/core"
    "github.com/gocrud/app/configure/database"
    "gorm.io/driver/mysql"
    "gorm.io/driver/postgres"
)

func main() {
    builder := core.NewApplicationBuilder()

    builder.Configure(database.Configure(func(b *database.Builder) {
        // 配置主库 (MySQL)
        b.Add("master", mysql.Open("user:pass@tcp(127.0.0.1:3306)/dbname?charset=utf8mb4&parseTime=True&loc=Local"), func(o *database.DatabaseOptions) {
            o.MaxOpenConns = 100
            o.MaxIdleConns = 10
            o.MaxLifetime = time.Hour
            // 注册需要自动迁移的模型
            o.AutoMigrate = []any{&User{}, &Order{}}
        })

        // 配置从库 (PostgreSQL)
        b.Add("slave", postgres.Open("host=localhost user=gorm password=gorm dbname=gorm port=9920 sslmode=disable"), nil)
    }))

    app := builder.Build()
    app.Run()
}
```

### 2. 在服务中使用（依赖注入）

定义你的服务结构体，使用 `di` 标签注入数据库实例：

```go
type UserRepository struct {
    // 注入名为 "master" 的数据库实例（必须存在）
    MasterDB *gorm.DB `di:"master"`
    
    // 注入名为 "slave" 的数据库实例（可选，如果未配置则为 nil）
    SlaveDB  *gorm.DB `di:"slave,?"`
}

func (r *UserRepository) GetUser(id uint) (*User, error) {
    var user User
    // 优先读从库，降级读主库
    db := r.SlaveDB
    if db == nil {
        db = r.MasterDB
    }
    
    if err := db.First(&user, id).Error; err != nil {
        return nil, err
    }
    return &user, nil
}
```

## 配置选项 (DatabaseOptions)

| 字段 | 类型 | 说明 | 默认值 |
| :--- | :--- | :--- | :--- |
| `Name` | `string` | 实例名称（用于 DI 注入标识） | (必填) |
| `Dialector` | `gorm.Dialector` | GORM 驱动实例 | (必填) |
| `GormConfig` | `*gorm.Config` | GORM 核心配置对象 | `{}` |
| `MaxIdleConns` | `int` | 最大空闲连接数 | `10` |
| `MaxOpenConns` | `int` | 最大打开连接数 | `100` |
| `MaxLifetime` | `time.Duration` | 连接最大存活时间 | `1h` |
| `AutoMigrate` | `[]any` | 需要自动迁移的模型列表 | `[]` |

## 常见问题

### 如何使用默认数据库（无名称注入）？

如果你配置了一个名为 `"default"` 的数据库：

```go
b.Add("default", mysql.Open(...), nil)
```

框架会自动将其注册为默认服务，你可以通过以下两种方式注入：

```go
// 方式 1：显式指定 default
DB *gorm.DB `di:"default"`

// 方式 2：留空（推荐）
DB *gorm.DB `di:""`
```

