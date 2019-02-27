package clusterstatus

import (
	"github.com/stackrox/rox/generated/internalapi/central"
)

// An Updater can update central on Cluster Status.
type Updater interface {
	Start()
	Stop()
	Updates() <-chan *central.ClusterStatusUpdate
}
