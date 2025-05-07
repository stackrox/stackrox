package manager

import (
	"github.com/stackrox/rox/pkg/net"
	"github.com/stackrox/rox/sensor/common/clusterentities"
)

// EntityStore interface to the clusterentities.Store
//
//go:generate mockgen-wrapper
type EntityStore interface {
	LookupByContainerID(string) (metadata clusterentities.ContainerMetadata, found bool, isHistorical bool)
	LookupByEndpoint(net.NumericEndpoint) []clusterentities.LookupResult
	DumpEndpointStore()
	RegisterPublicIPsListener(clusterentities.PublicIPsListener) bool
	UnregisterPublicIPsListener(clusterentities.PublicIPsListener) bool
	RecordTick()
}
