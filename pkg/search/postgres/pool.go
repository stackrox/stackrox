package postgres

import (
	"github.com/stackrox/rox/central/metrics"
	"google.golang.org/grpc/mem"
)

// BufferPool is a pool of buffers that can be shared and reused, resulting in
// decreased memory allocation.
// See: google.golang.org/grpc/mem/buffer_pool.go
type BufferPool interface {
	// Get returns a buffer with specified length from the pool.
	Get(length int) *[]byte

	// Put returns a buffer to the pool.
	Put(...*[]byte)
}

// TODO: adjust to real data usage
var defaultBufferPoolSizes = []int{
	256,
	4 << 10,  // 4KB (go page size)
	16 << 10, // 16KB (max HTTP/2 frame size used by gRPC)
	32 << 10, // 32KB (default buffer size for io.Copy)
	1 << 20,  // 1MB
}

var defaultBufferPool BufferPool = &poolWithMetric{mem.NewTieredBufferPool(defaultBufferPoolSizes...)}

type poolWithMetric struct {
	mem.BufferPool
}

// Get returns a buffer with specified length from the pool.
func (p *poolWithMetric) Get(length int) *[]byte {
	metrics.ObserveSerializedSize(length)
	return p.BufferPool.Get(length)
}

// Put returns a buffer to the pool.
func (p *poolWithMetric) Put(buffers ...*[]byte) {
	for _, buffer := range buffers {
		p.BufferPool.Put(buffer)
	}
}
