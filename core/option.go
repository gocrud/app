package core

// Option 定义了修改 Runtime 状态的函数签名
// 这是框架唯一的扩展点
type Option func(rt *Runtime) error
