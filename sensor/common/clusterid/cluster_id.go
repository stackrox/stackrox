package clusterid

import (
	"log"

	"github.com/stackrox/rox/pkg/sensor/clusterid"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once      sync.Once
	clusterID string
)

// Get returns the cluster id parsed from the service certficate
func Get() string {
	once.Do(func() {
		id, err := clusterid.ParseClusterIDFromServiceCert()
		if err != nil {
			log.Panicf("Error parsing cluster id from certficate: %v", err)
		}
		clusterID = id
	})
	return clusterID
}
