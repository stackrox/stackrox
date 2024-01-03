package tree

import (
	"bytes"
	"net"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/networkgraph"
)

var (
	ipv4InternetCIDR = "0.0.0.0/0"
	ipv6InternetCIDR = "::ffff:0:0/0"
)

func ipNetEqual(a, b *net.IPNet) bool {
	return a.IP.Equal(b.IP) && bytes.Equal(a.Mask, b.Mask)
}

func rmDescIfInternet(entity *storage.NetworkEntityInfo) {
	// Throughout the codebase, internet and internal nodes are expected only with ID and Type.
	if networkgraph.IsConstantID(entity.GetId()) {
		entity.Desc = nil
	}
}
