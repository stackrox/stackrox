package fake

import (
	"fmt"
)

type vmInfo struct {
	id       string
	vsockCID uint32
	name     string
}

type vmPool struct {
	vms []*vmInfo
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
