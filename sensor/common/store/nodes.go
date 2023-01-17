package store

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/net"
)

// NodeWrap adds address information to nodes to forward to central
type NodeWrap struct {
	*storage.Node
	Addresses []net.IPAddress
}
