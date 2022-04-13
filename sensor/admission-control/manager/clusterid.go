package manager

import (
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/clusterid"
	"github.com/stackrox/stackrox/pkg/sync"
	"github.com/stackrox/stackrox/pkg/utils"
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
