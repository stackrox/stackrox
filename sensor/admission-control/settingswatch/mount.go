package settingswatch

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/admissioncontrol"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/fileutils"
	"github.com/stackrox/rox/pkg/gziputil"
	"github.com/stackrox/rox/pkg/k8scfgwatch"
)

const (
	settingsMountPath = `/run/config/stackrox.io/admission-control/config/`
)

// WatchMountPathForSettingsUpdateAsync watches the config map mount path for updates to admission control settings.
func WatchMountPathForSettingsUpdateAsync(ctx concurrency.Waitable, outC chan<- *sensor.AdmissionControlSettings) error {
	w := &mountSettingsWatch{
		ctx:  ctx,
		outC: outC,
	}

	return w.start()
}

type mountSettingsWatch struct {
	ctx  concurrency.Waitable
	outC chan<- *sensor.AdmissionControlSettings
}

func (m *mountSettingsWatch) OnChange(dir string) (interface{}, error) {
	configPath := filepath.Join(dir, admissioncontrol.ConfigGZDataKey)
	deployTimePoliciesPath := filepath.Join(dir, admissioncontrol.DeployTimePoliciesGZDataKey)
	runTimePoliciesPath := filepath.Join(dir, admissioncontrol.RunTimePoliciesGZDataKey)
	timestampPath := filepath.Join(dir, admissioncontrol.LastUpdateTimeDataKey)
	cacheVersionPath := filepath.Join(dir, admissioncontrol.CacheVersionDataKey)
	centralEndpointPath := filepath.Join(dir, admissioncontrol.CentralEndpointDataKey)
	clusterIDPath := filepath.Join(dir, admissioncontrol.ClusterIDDataKey)

	if noneExists, err := fileutils.NoneExists(configPath, deployTimePoliciesPath, timestampPath); err != nil {
		return nil, errors.Wrapf(err, "error checking the existence of files in %s", dir)
	} else if noneExists {
		return nil, nil
	}

	configDataGZ, err := os.ReadFile(configPath)
	if err != nil {
		return nil, errors.Wrapf(err, "reading cluster config from file %s", configPath)
	}
	configData, err := gziputil.Decompress(configDataGZ)
	if err != nil {
		return nil, errors.Wrapf(err, "decompressing cluster config in file %s", configPath)
	}

	var clusterConfig storage.DynamicClusterConfig
	if err := proto.Unmarshal(configData, &clusterConfig); err != nil {
		return nil, errors.Wrapf(err, "unmarshaling decompressed cluster config data from file %s", configPath)
	}

	deployTimePolicies, err := getPoliciesFromFile(deployTimePoliciesPath)
	if err != nil {
		return nil, errors.Wrapf(err, "getting deploy-time policies from file %s", deployTimePoliciesPath)
	}

	runTimePolicies, err := getPoliciesFromFile(runTimePoliciesPath)
	if err != nil {
		return nil, errors.Wrapf(err, "getting run-time policies from file %s", runTimePoliciesPath)
	}

	timestampBytes, err := os.ReadFile(timestampPath)
	if err != nil {
		return nil, errors.Wrapf(err, "reading last update timestamp from file %s", timestampPath)
	}

	timestamp, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(string(timestampBytes)))
	if err != nil {
		return nil, errors.Wrapf(err, "parsing last update timestamp from file %s", timestampPath)
	}

	tsProto, err := types.TimestampProto(timestamp)
	if err != nil {
		return nil, errors.Wrapf(err, "timestamp in file %s is invalid", timestampPath)
	}

	var cacheVersion string
	cacheVersionBytes, err := os.ReadFile(cacheVersionPath)
	if err != nil && !os.IsNotExist(err) {
		log.Errorf("Failed to read cache version from file %s: %v", cacheVersionPath, err)
	} else {
		cacheVersion = string(cacheVersionBytes)
	}

	var centralEndpoint string
	centralEndpointBytes, err := os.ReadFile(centralEndpointPath)
	if err != nil && !os.IsNotExist(err) {
		log.Errorf("Failed to read central endpoint from file %s: %v", centralEndpointPath, err)
	} else {
		centralEndpoint = string(centralEndpointBytes)
	}

	var clusterID string
	clusterIDBytes, err := os.ReadFile(clusterIDPath)
	if err != nil && !os.IsNotExist(err) {
		log.Errorf("Failed to read cluster ID from file %s: %v", clusterIDPath, err)
	} else {
		clusterID = string(clusterIDBytes)
	}

	return &sensor.AdmissionControlSettings{
		ClusterConfig:              &clusterConfig,
		EnforcedDeployTimePolicies: deployTimePolicies,
		RuntimePolicies:            runTimePolicies,
		Timestamp:                  tsProto,
		CacheVersion:               cacheVersion,
		CentralEndpoint:            centralEndpoint,
		ClusterId:                  clusterID,
	}, nil
}

func (m *mountSettingsWatch) OnStableUpdate(val interface{}, err error) {
	if err != nil {
		log.Errorf("Failed to update admission controller settings from config mount: %v", err)
		return
	}

	settings, _ := val.(*sensor.AdmissionControlSettings)
	if settings == nil {
		return
	}

	select {
	case <-m.ctx.Done():
		return
	case m.outC <- settings:
	}

	log.Infof("Detected and propagated updated admission controller settings via config map mount, timestamp: %v", settings.GetTimestamp())
}

func (m *mountSettingsWatch) OnWatchError(err error) {
	log.Errorf("Error watching config map mount directory %s: %v", settingsMountPath, err)
}

func (m *mountSettingsWatch) start() error {
	if err := k8scfgwatch.WatchConfigMountDir(m.ctx, settingsMountPath, k8scfgwatch.DeduplicateWatchErrors(m), k8scfgwatch.Options{Force: true}); err != nil {
		return err
	}
	return nil
}
