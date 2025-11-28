package config

// Load 加载并绑定指定节的配置到结构体 T
// 这是一个泛型辅助函数，简化了 Configuration.GetSection().Bind() 的调用
func Load[T any](cfg Configuration, section string) (T, error) {
	var t T
	// 如果 section 为空，尝试从根节点绑定
	// 注意：Configuration 接口的 Bind 方法入参是 key，如果 key 为空则绑定全部。
	// GetSection 方法入参也是 key，如果 key 为空则返回根节点配置。

	if section == "" {
		// 直接绑定整个配置到 T
		// Bind 的 key 参数如果是 ""，则绑定整个 data
		err := cfg.Bind("", &t)
		return t, err
	}

	// 绑定指定 section
	// 方式 1: 使用 GetSection 然后 Bind
	// err := cfg.GetSection(section).Bind("", &t)

	// 方式 2: 直接使用 Bind(key, target)
	err := cfg.Bind(section, &t)
	return t, err
}
