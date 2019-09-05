package upgradecontroller

import (
	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/pkg/version"
)

func newUpgradeProcess(cluster *storage.Cluster) (*storage.ClusterUpgradeStatus_UpgradeProcessStatus, error) {
	imageName, _, err := utils.GenerateImageNameFromString(cluster.GetMainImage())
	if err != nil {
		return nil, errors.Wrapf(err, "config for cluster %s has unparseable main image name", cluster.GetId())
	}
	if imageName.GetTag() != "" {
		return nil, errors.Errorf("config for cluster %s specifies a hard-coded tag %q, remove it to use the auto-upgrade feature", cluster.GetId(), imageName.GetTag())
	}
	imageName.Tag = version.GetMainVersion()
	imageName = utils.NormalizeImageFullNameNoSha(imageName)

	return &storage.ClusterUpgradeStatus_UpgradeProcessStatus{
		Active:        true,
		Id:            uuid.NewV4().String(),
		TargetVersion: version.GetMainVersion(),
		UpgraderImage: imageName.GetFullName(),
		InitiatedAt:   types.TimestampNow(),
		Progress:      &storage.UpgradeProgress{},
	}, nil
}

func (u *upgradeController) newUpgradeProcess() (*storage.ClusterUpgradeStatus_UpgradeProcessStatus, error) {
	return newUpgradeProcess(u.getCluster())
}
