package connection

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/version"
)

var (
	clusterUpdateCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Cluster)))
)

type upgradeController struct {
	injector   common.MessageInjector
	clusterID  string
	errorSig   concurrency.ErrorSignal
	clusterMgr ClusterManager
}

func newUpgradeController(initialCtx context.Context, injector common.MessageInjector, clusterID string, clusterMgr ClusterManager) *upgradeController {
	u := &upgradeController{
		injector:   injector,
		errorSig:   concurrency.NewErrorSignal(),
		clusterID:  clusterID,
		clusterMgr: clusterMgr,
	}
	go u.handleInitialContext(initialCtx)
	return u
}

func (u *upgradeController) errorSignal() concurrency.ReadOnlyErrorSignal {
	return &u.errorSig
}

func (u *upgradeController) upgradeClusterStatusOrTerminate(status *storage.ClusterUpgradeStatus) {
	err := u.clusterMgr.UpdateClusterUpgradeStatus(clusterUpdateCtx, u.clusterID, status)
	if err != nil {
		u.errorSig.SignalWithError(errors.Wrap(err, "failed to write cluster upgrade status"))
	}
}

func (u *upgradeController) determineUpgradabilityFromVersionInfo(versionInfo *centralsensor.SensorVersionInfo) storage.ClusterUpgradeStatus_Upgradability {
	if versionInfo == nil {
		log.Infof("Sensor from cluster %s is from an old version that doesn't support auto-upgrade", u.clusterID)
		return storage.ClusterUpgradeStatus_MANUAL_UPGRADE_REQUIRED
	}

	if versionInfo.MainVersion == version.GetMainVersion() {
		log.Infof("Sensor from cluster %s is running the same version as Central (%s)", u.clusterID, versionInfo.MainVersion)
		return storage.ClusterUpgradeStatus_UP_TO_DATE
	}
	cmp := version.CompareReleaseVersions(versionInfo.MainVersion, version.GetMainVersion())
	// The sensor is newer! See comments on the below enum value in the proto file
	// for more details on how we handle this case.
	if cmp > 0 {
		log.Infof("Sensor from cluster %s is running a newer version! (%s)", u.clusterID, versionInfo.MainVersion)
		return storage.ClusterUpgradeStatus_SENSOR_VERSION_HIGHER
	}
	// We don't differentiate between cmp == -1 and cmp == 0.
	// The former means we definitely know sensor is an older version.
	// The latter means we don't know (ex: we're on a development version)
	// In such a case, it seems reasonable to assume that the sensor is older.
	// Ideally, we would panic if cmp == 0 on release builds, since that should
	// only happen if the versions are exactly equal (which is checked above),
	// but panic-ing on release builds doesn't help anyone with on-prem software, so...
	log.Infof("Sensor from cluster %s is running an older version (%s). Auto upgrading is possible", u.clusterID, versionInfo.MainVersion)
	// TODO(viswa): Change this to auto-upgrade possible when it's, well, possible.
	return storage.ClusterUpgradeStatus_MANUAL_UPGRADE_REQUIRED
}

func (u *upgradeController) handleInitialContext(initialCtx context.Context) {
	versionInfo, err := centralsensor.DeriveSensorVersionInfo(initialCtx)
	if err != nil {
		// This ONLY happens when the sensor gives an inconsistent version.
		errorhelpers.PanicOnDevelopment(err)
		u.errorSig.SignalWithErrorf("couldn't derive version info from context: %v", err)
		return
	}

	u.upgradeClusterStatusOrTerminate(&storage.ClusterUpgradeStatus{Upgradability: u.determineUpgradabilityFromVersionInfo(versionInfo)})
}
