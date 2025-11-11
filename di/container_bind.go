package di

// BindWith 绑定接口到实现（容器实例版本）
// 使用示例: di.BindWith[Logger](container, &ConsoleLogger{})
func BindWith[T any](c *Container, impl any) {
	c.ProvideType(TypeProvider{
		Provide: TypeOf[T](),
		UseType: impl,
	})
}

// BindToWith 创建别名（容器实例版本）
// 使用示例: di.BindToWith[AliasType](container, existingValue)
func BindToWith[T any](c *Container, existing any) {
	c.ProvideExisting(ExistingProvider{
		Provide:  TypeOf[T](),
		Existing: existing,
	})
}
