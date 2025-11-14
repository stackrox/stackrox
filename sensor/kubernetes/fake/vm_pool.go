package fake

import (
	"fmt"
	"sync"
)

type vmInfo struct {
	id       string
	vsockCID uint32
	name     string
}

type vmPool struct {
	vms    []*vmInfo
	nextVM int
	mu     sync.Mutex
}

func newVMPool(numVMs int) *vmPool {
	pool := &vmPool{
		vms: make([]*vmInfo, numVMs),
	}
	for i := 0; i < numVMs; i++ {
		pool.vms[i] = &vmInfo{
			id:       fmt.Sprintf("vm-%d", i),
			vsockCID: uint32(1000 + i),
			name:     fmt.Sprintf("fake-vm-%d", i),
		}
	}
	return pool
}

func (p *vmPool) getRoundRobin() *vmInfo {
	p.mu.Lock()
	defer p.mu.Unlock()
	vm := p.vms[p.nextVM]
	p.nextVM = (p.nextVM + 1) % len(p.vms)
	return vm
}
