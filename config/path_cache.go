package config

import (
	"strings"
	"sync"
)

// PathCache 缓存配置路径解析结果
type PathCache struct {
	cache sync.Map
}

// GetPathSegments 获取路径片段，如果缓存不存在则解析并缓存
func (c *PathCache) GetPathSegments(path string) []string {
	if v, ok := c.cache.Load(path); ok {
		return v.([]string)
	}

	// 解析路径：支持 : 和 . 作为分隔符
	parts := strings.Split(strings.ReplaceAll(path, ":", "."), ".")
	c.cache.Store(path, parts)
	return parts
}

// globalPathCache 全局路径缓存实例
var globalPathCache = &PathCache{}
