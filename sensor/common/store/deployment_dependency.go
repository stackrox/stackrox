package store

import (
	"github.com/mitchellh/hashstructure/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/sensor/common/service"
)

// Dependencies are properties that belong to a storage.Deployment object, but don't come directly from the
// k8s deployment spec. They need to be enhanced from other resources, like RBACs and Services.
type Dependencies struct {
	PermissionLevel storage.PermissionLevel
	Exposures       []map[service.PortRef][]*storage.PortConfig_ExposureInfo
	LocalImages     set.StringSet
}

func (d *Dependencies) GetHash() (uint64, error) {
	return hashstructure.Hash(d, hashstructure.FormatV2, &hashstructure.HashOptions{
		ZeroNil:         true,
		IgnoreZeroValue: true,
		SlicesAsSets:    true,
	})
}
