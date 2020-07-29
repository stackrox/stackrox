package upgradecontroller

import (
	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/pkg/version"
)

func updateTagToMainVersionAndGetFullName(imageName *storage.ImageName) string {
	imageName.Tag = version.GetMainVersion()
	return utils.NormalizeImageFullNameNoSha(imageName).GetFullName()
}

func newCertRotationProcess() *storage.ClusterUpgradeStatus_UpgradeProcessStatus {
	return baseNewUpgradeProcess(storage.ClusterUpgradeStatus_UpgradeProcessStatus_CERT_ROTATION)
}

func baseNewUpgradeProcess(typ storage.ClusterUpgradeStatus_UpgradeProcessStatus_UpgradeProcessType) *storage.ClusterUpgradeStatus_UpgradeProcessStatus {
	return &storage.ClusterUpgradeStatus_UpgradeProcessStatus{
		Active:      true,
		Id:          uuid.NewV4().String(),
		InitiatedAt: types.TimestampNow(),
		Progress:    &storage.UpgradeProgress{},
		Type:        typ,
	}
}

func newUpgradeProcess(cluster *storage.Cluster) (*storage.ClusterUpgradeStatus_UpgradeProcessStatus, error) {
	imageName, _, err := utils.GenerateImageNameFromString(cluster.GetMainImage())
	if err != nil {
		return nil, errors.Wrapf(err, "config for cluster %s has unparseable main image name", cluster.GetId())
	}
	if imageName.GetTag() != "" {
		return nil, errors.Errorf("config for cluster %s specifies a hard-coded tag %q, remove it to use the auto-upgrade feature", cluster.GetId(), imageName.GetTag())
	}
	upgraderImage := updateTagToMainVersionAndGetFullName(imageName)

	process := baseNewUpgradeProcess(storage.ClusterUpgradeStatus_UpgradeProcessStatus_UPGRADE)
	process.UpgraderImage = upgraderImage
	process.TargetVersion = version.GetMainVersion()
	return process, nil
}

func (u *upgradeController) newUpgradeProcess() (*storage.ClusterUpgradeStatus_UpgradeProcessStatus, error) {
	return newUpgradeProcess(u.getCluster())
}
