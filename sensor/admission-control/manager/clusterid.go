package manager

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/clusterid"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	clusterID     string
	clusterIDInit sync.Once
)

func getClusterID() string {
	clusterIDInit.Do(func() {
		var err error
		clusterID, err = clusterid.ParseClusterIDFromServiceCert(storage.ServiceType_ADMISSION_CONTROL_SERVICE)
		utils.Should(err) // use an empty cluster ID in release builds, better than crashing.
	})
	return clusterID
}
