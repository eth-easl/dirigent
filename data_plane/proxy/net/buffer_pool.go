package net

import (
	"net/http/httputil"
	"sync"
)

type bufferPool struct {
	pool *sync.Pool
}

// NewBufferPool creates a new BufferPool. This is only safe to use in the context
// of a httputil.ReverseProxy, as the buffers returned via Put are not cleaned
// explicitly.
func NewBufferPool() httputil.BufferPool {
	return &bufferPool{
		// We don't use the New function of sync.Pool here to avoid an unnecessary
		// allocation when creating the slices. They are implicitly created in the
		// Get function below.
		pool: &sync.Pool{},
	}
}

func (b *bufferPool) Get() []byte {
	buf := b.pool.Get()
	if buf == nil {
		// Use the default buffer size as defined in the ReverseProxy itself.
		return make([]byte, 32*1024)
	}

	return *buf.(*[]byte)
}

// Put returns the given Buffer to the bufferPool.
func (b *bufferPool) Put(buffer []byte) {
	b.pool.Put(&buffer)
}
