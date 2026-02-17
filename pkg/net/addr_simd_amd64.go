//go:build amd64 && goexperiment.simd

package net

import (
	"net"

	"github.com/stackrox/rox/pkg/net/internal/simdutil"
)

// isPublic implementation for AMD64 with SIMD support.
// Uses vectorized operations to check multiple subnet masks in parallel.
func (d ipv4data) isPublic() bool {
	return simdutil.CheckIPv4Public(d)
}

func (d ipv6data) isPublic() bool {
	// IPv6 SIMD optimization is more complex due to 128-bit addresses.
	// Start with scalar fallback; can be enhanced with 256-bit or 512-bit vectors.
	return isPublicIPv6Scalar(net.IP(d.bytes()))
}
