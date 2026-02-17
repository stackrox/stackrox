//go:build !amd64 || !goexperiment.simd

package net

import (
	"net"

	"github.com/stackrox/rox/pkg/net/internal/simdutil"
)

// isPublic stub implementation for non-SIMD builds.
// Falls back to optimized scalar implementation using bitwise operations.
func (d ipv4data) isPublic() bool {
	return simdutil.CheckIPv4Public(d)
}

func (d ipv6data) isPublic() bool {
	return isPublicIPv6Scalar(net.IP(d.bytes()))
}
