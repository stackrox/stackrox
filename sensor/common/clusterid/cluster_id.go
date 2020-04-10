package clusterid

import (
	"log"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/clusterid"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once      sync.Once
	clusterID string
)

// Get returns the cluster id parsed from the service certficate
func Get() string {
	once.Do(func() {
		id, err := clusterid.ParseClusterIDFromServiceCert(storage.ServiceType_SENSOR_SERVICE)
		if err != nil {
			log.Panicf("Error parsing cluster id from certficate: %v", err)
		}
		clusterID = id
	})
	return clusterID
}
