package testutils

import (
	"fmt"
	"net"
	"net/http"
	"sync/atomic"

	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

// Starting with 10000 up to 30000. Linux will usually use ports >30000 for clients.
// By starting with 10000, we are trying to avoid port congestion in the upper port range.
const minFreePortRange = 10000
const maxFreePortRange = 30000

var (
	once        sync.Once
	portCounter atomic.Uint64
)

// GetFreeTestPort returns next available free port. It panics if range of free ports is exhausted.
func GetFreeTestPort() uint64 {
	once.Do(func() {
		portCounter.Store(minFreePortRange)
	})

	for {
		freePort := portCounter.Add(1)
		if freePort > maxFreePortRange {
			panic("port number is out of range")
		}

		// If there is an error: net.Listen - will always return nil listener. (from code, no docs)
		listener, err := net.Listen("tcp4", fmt.Sprintf(":%d", freePort))
		if err != nil {
			continue
		}
		utils.IgnoreError(listener.Close)

		// Ensure that port is free on IPv4 and IPv6.
		listener, err = net.Listen("tcp6", fmt.Sprintf(":%d", freePort))
		if err != nil {
			continue
		}
		utils.IgnoreError(listener.Close)

		return freePort
	}
}

// SafeClientClose safely closes an open client connection.
func SafeClientClose(resp *http.Response) {
	if resp == nil || resp.Body == nil {
		return
	}

	utils.IgnoreError(resp.Body.Close)
}

// GetUsedPortsList returns list of ports returned by GetFreeTestPort.
func GetUsedPortsList() []uint64 {
	lastPort := portCounter.Add(0)

	var ports []uint64
	for port := uint64(minFreePortRange); port <= lastPort; port++ {
		ports = append(ports, port)
	}

	return ports
}
