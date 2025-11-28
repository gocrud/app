package core

// BaseBuilder 提供基础的构建上下文能力
// 所有模块的 Builder 都应该嵌入此结构体
type BaseBuilder struct {
	ctx *BuildContext
}

// NewBaseBuilder 创建基础构建器
func NewBaseBuilder(ctx *BuildContext) BaseBuilder {
	return BaseBuilder{ctx: ctx}
}

// ConfigContext 获取构建上下文（受限接口）
func (b *BaseBuilder) ConfigContext() ConfigurationContext {
	return b.ctx
}

// RegisterCleanup 允许 Builder 注册清理函数（受保护的代理方法）
// 这样 Builder 内部可以注册清理，但通过 Context() 获取的接口无法注册
func (b *BaseBuilder) RegisterCleanup(key string, cleanup func()) {
	b.ctx.SetCleanup(key, cleanup)
}
