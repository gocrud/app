package logging

import (
	"bytes"
	"sync"
)

// BufferPool 简单的字节缓冲池，用于复用 buffer 减少 GC
type BufferPool struct {
	pool sync.Pool
}

// NewBufferPool 创建新的缓冲池
func NewBufferPool() *BufferPool {
	return &BufferPool{
		pool: sync.Pool{
			New: func() interface{} {
				return new(bytes.Buffer)
			},
		},
	}
}

// Get 获取一个 buffer
func (p *BufferPool) Get() *bytes.Buffer {
	return p.pool.Get().(*bytes.Buffer)
}

// Put 归还一个 buffer
func (p *BufferPool) Put(b *bytes.Buffer) {
	b.Reset()
	p.pool.Put(b)
}

// GlobalBufferPool 全局缓冲池实例
var GlobalBufferPool = NewBufferPool()

