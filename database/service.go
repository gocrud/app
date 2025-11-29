package database

import (
	"fmt"
	"sync"
	"time"

	"gorm.io/gorm"
)

// DatabaseOptions 数据库配置选项
type DatabaseOptions struct {
	Name         string
	Dialector    gorm.Dialector
	GormConfig   *gorm.Config
	MaxIdleConns int
	MaxOpenConns int
	MaxLifetime  time.Duration
	AutoMigrate  []any // 需要自动迁移的模型
}

// NewDefaultOptions 创建默认配置
func NewDefaultOptions(name string, dialector gorm.Dialector) *DatabaseOptions {
	return &DatabaseOptions{
		Name:         name,
		Dialector:    dialector,
		GormConfig:   &gorm.Config{},
		MaxIdleConns: 10,
		MaxOpenConns: 100,
		MaxLifetime:  time.Hour,
		AutoMigrate:  make([]any, 0),
	}
}

// Validate 验证配置
func (o *DatabaseOptions) Validate() error {
	if o.Name == "" {
		return fmt.Errorf("database name is required")
	}
	if o.Dialector == nil {
		return fmt.Errorf("database dialector is required")
	}
	return nil
}

// DatabaseFactory 数据库客户端工厂
type DatabaseFactory struct {
	dbs map[string]*gorm.DB
	mu  sync.RWMutex
}

// NewDatabaseFactory 创建数据库工厂
func NewDatabaseFactory() *DatabaseFactory {
	return &DatabaseFactory{
		dbs: make(map[string]*gorm.DB),
	}
}

// Register 注册数据库实例
func (f *DatabaseFactory) Register(opts DatabaseOptions) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if _, exists := f.dbs[opts.Name]; exists {
		return fmt.Errorf("database '%s' already registered", opts.Name)
	}

	// 打开数据库连接
	db, err := gorm.Open(opts.Dialector, opts.GormConfig)
	if err != nil {
		return fmt.Errorf("failed to open database '%s': %w", opts.Name, err)
	}

	// 配置连接池
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get sql.DB for '%s': %w", opts.Name, err)
	}

	sqlDB.SetMaxIdleConns(opts.MaxIdleConns)
	sqlDB.SetMaxOpenConns(opts.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(opts.MaxLifetime)

	// 执行自动迁移
	if len(opts.AutoMigrate) > 0 {
		if err := db.AutoMigrate(opts.AutoMigrate...); err != nil {
			return fmt.Errorf("auto migrate failed for '%s': %w", opts.Name, err)
		}
	}

	f.dbs[opts.Name] = db
	return nil
}

// Each 遍历所有数据库实例
func (f *DatabaseFactory) Each(fn func(name string, db *gorm.DB)) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	for name, db := range f.dbs {
		fn(name, db)
	}
}

// Close 关闭所有数据库连接
func (f *DatabaseFactory) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	var errs []error
	for name, db := range f.dbs {
		sqlDB, err := db.DB()
		if err != nil {
			// 如果获取不到 sqlDB，可能连接已经有问题，记录错误但继续
			errs = append(errs, fmt.Errorf("failed to get sql.DB for '%s': %w", name, err))
			continue
		}
		if err := sqlDB.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close database '%s': %w", name, err))
		}
	}

	f.dbs = make(map[string]*gorm.DB)

	if len(errs) > 0 {
		return fmt.Errorf("errors closing databases: %v", errs)
	}
	return nil
}
