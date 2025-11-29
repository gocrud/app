# 数据库 (Database)

数据库模块基于 [GORM](https://gorm.io) 封装，支持 MySQL, PostgreSQL, SQLite, SQLServer 等多种数据库。

## 启用数据库

```go
import (
    "github.com/gocrud/app/database"
    "gorm.io/driver/mysql"
)

app.Run(
    database.New(
        // 注册名为 "default" 的数据库连接
        database.WithDatabase("default", mysql.Open("user:pass@tcp(127.0.0.1:3306)/dbname")),
    ),
)
```

## 注入使用

### 默认注入

如果只注册了一个数据库，或者名为 "default"，可以直接注入 `*gorm.DB`。

```go
type UserRepo struct {
    DB *gorm.DB `di:""`
}
```

### 命名注入

如果有多个数据库连接：

```go
type AnalyticsRepo struct {
    // 注入名为 "analytics" 的连接
    DB *gorm.DB `di:"name=analytics"`
}
```

## 最佳实践

### Repository 模式

建议将数据库操作封装在 Repository 层。

```go
type UserRepo struct {
    DB *gorm.DB `di:""`
}

func (r *UserRepo) Create(u *User) error {
    return r.DB.Create(u).Error
}

func (r *UserRepo) FindByID(id uint) (*User, error) {
    var u User
    err := r.DB.First(&u, id).Error
    return &u, err
}
```

### 事务处理 (Transactions)

GORM 提供了强大的事务支持。你可以在 Repository 中处理事务，或者将其提升到 Service 层（通过传递 `*gorm.DB` 或使用闭包）。

**简单事务 (Closure)**:

```go
func (s *UserService) Transfer(fromID, toID uint, amount int) error {
    return s.Repo.DB.Transaction(func(tx *gorm.DB) error {
        // 在事务中执行数据库操作
        if err := tx.Model(&User{}).Where("id = ?", fromID).Update("balance", gorm.Expr("balance - ?", amount)).Error; err != nil {
            return err
        }

        if err := tx.Model(&User{}).Where("id = ?", toID).Update("balance", gorm.Expr("balance + ?", amount)).Error; err != nil {
            return err
        }

        return nil
    })
}
```

### 迁移 (Migration)

建议在应用启动时的 `OnStart` 钩子中执行自动迁移。

```go
func WithAutoMigrate() core.Option {
    return func(rt *core.Runtime) error {
        rt.Lifecycle.OnStart(func(ctx context.Context) error {
            // 获取 DB 实例
            var db *gorm.DB
            if err := rt.Invoke(func(d *gorm.DB) { db = d }); err != nil {
                return err
            }
            
            // 执行迁移
            return db.AutoMigrate(&User{}, &Product{})
        })
        return nil
    }
}
```

