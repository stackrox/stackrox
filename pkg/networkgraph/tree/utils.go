package tree

import (
	"bytes"
	"net"

	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/networkgraph"
)

var (
	ipv4InternetCIDR = "0.0.0.0/0"
	ipv6InternetCIDR = "::ffff:0:0/0"
)

func ipNetEqual(a, b *net.IPNet) bool {
	return a.IP.Equal(b.IP) && bytes.Equal(a.Mask, b.Mask)
}

func rmDescIfInternet(entity *storage.NetworkEntityInfo) {
	// Throughout the codebase, internet node is expected only with ID and Type.
	if entity.GetId() == networkgraph.InternetExternalSourceID {
		entity.Desc = nil
	}
}
