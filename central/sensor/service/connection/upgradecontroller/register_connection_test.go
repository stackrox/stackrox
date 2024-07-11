package upgradecontroller

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/version/testutils"
	"github.com/stretchr/testify/assert"
)

func TestUpgradability(t *testing.T) {
	type testCase struct {
		centralVersion       string
		sensorVersion        string
		autoUpgradeSupported bool
		expectedStatus       storage.ClusterUpgradeStatus_Upgradability
	}

	cases := map[string]testCase{
		"sensor version is empty": {
			centralVersion:       "3.73.0",
			sensorVersion:        "",
			autoUpgradeSupported: true,
			expectedStatus:       storage.ClusterUpgradeStatus_MANUAL_UPGRADE_REQUIRED,
		},
		"sensor and central versions are equal": {
			centralVersion:       "3.73.0",
			sensorVersion:        "3.73.0",
			autoUpgradeSupported: true,
			expectedStatus:       storage.ClusterUpgradeStatus_UP_TO_DATE,
		},
		"sensor version is lower than central": {
			centralVersion:       "3.73.0",
			sensorVersion:        "3.72.3",
			autoUpgradeSupported: true,
			expectedStatus:       storage.ClusterUpgradeStatus_AUTO_UPGRADE_POSSIBLE,
		},
		"sensor version is higher than central": {
			centralVersion:       "3.73.0",
			sensorVersion:        "3.73.1",
			autoUpgradeSupported: true,
			expectedStatus:       storage.ClusterUpgradeStatus_SENSOR_VERSION_HIGHER,
		},
		"sensor version is lower than central when sensor does not support autoupgrade": {
			centralVersion:       "3.73.0",
			sensorVersion:        "3.72.3",
			autoUpgradeSupported: false,
			expectedStatus:       storage.ClusterUpgradeStatus_MANUAL_UPGRADE_REQUIRED,
		},
		"sensor version is higher than central when sensor does not support autoupgrade": {
			centralVersion:       "3.73.0",
			sensorVersion:        "3.73.1",
			autoUpgradeSupported: false,
			expectedStatus:       storage.ClusterUpgradeStatus_MANUAL_UPGRADE_REQUIRED,
		},
	}

	for desc, tc := range cases {
		t.Run(desc, func(t *testing.T) {
			testutils.SetMainVersion(t, tc.centralVersion)
			connection := fakeConnection{
				sensorVersion:        tc.sensorVersion,
				autoUpgradeSupported: tc.autoUpgradeSupported,
			}
			upgradability, _ := determineUpgradabilityFromVersionInfoAndConn(tc.sensorVersion, connection)
			assert.Equal(t, tc.expectedStatus, upgradability)
		})
	}
}

var _ SensorConn = fakeConnection{}

type fakeConnection struct {
	autoUpgradeSupported bool
	sensorVersion        string
}

func (f fakeConnection) InjectMessage(_ concurrency.Waitable, _ *central.MsgToSensor) error {
	panic("not implemented")
}

func (f fakeConnection) InjectMessageIntoQueue(_ *central.MsgFromSensor) {
	panic("not implemented")
}

func (f fakeConnection) CheckAutoUpgradeSupport() error {
	if f.autoUpgradeSupported {
		return nil
	}
	return errors.New("Test error")
}

func (f fakeConnection) SensorVersion() string {
	return f.sensorVersion
}
