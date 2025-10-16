package upgradecontroller

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/pkg/version"
)

func updateTagToMainVersionAndGetFullName(imageName *storage.ImageName) string {
	imageName.SetTag(version.GetMainVersion())
	return utils.NormalizeImageFullNameNoSha(imageName).GetFullName()
}

func newCertRotationProcess() *storage.ClusterUpgradeStatus_UpgradeProcessStatus {
	return baseNewUpgradeProcess(storage.ClusterUpgradeStatus_UpgradeProcessStatus_CERT_ROTATION)
}

func baseNewUpgradeProcess(typ storage.ClusterUpgradeStatus_UpgradeProcessStatus_UpgradeProcessType) *storage.ClusterUpgradeStatus_UpgradeProcessStatus {
	cu := &storage.ClusterUpgradeStatus_UpgradeProcessStatus{}
	cu.SetActive(true)
	cu.SetId(uuid.NewV4().String())
	cu.SetInitiatedAt(protocompat.TimestampNow())
	cu.SetProgress(&storage.UpgradeProgress{})
	cu.SetType(typ)
	return cu
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
	process.SetUpgraderImage(upgraderImage)
	process.SetTargetVersion(version.GetMainVersion())
	return process, nil
}

func (u *upgradeController) newUpgradeProcess() (*storage.ClusterUpgradeStatus_UpgradeProcessStatus, error) {
	return newUpgradeProcess(u.getCluster())
}
