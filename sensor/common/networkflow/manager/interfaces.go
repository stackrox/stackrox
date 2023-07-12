package manager

import (
	"github.com/stackrox/rox/pkg/net"
	"github.com/stackrox/rox/sensor/common/clusterentities"
)

//go:generate mockgen-wrapper
type EntityStore interface {
	LookupByContainerID(string) (clusterentities.ContainerMetadata, bool)
	LookupByEndpoint(net.NumericEndpoint) []clusterentities.LookupResult
	RegisterPublicIPsListener(clusterentities.PublicIPsListener) bool
	UnregisterPublicIPsListener(clusterentities.PublicIPsListener) bool
}
