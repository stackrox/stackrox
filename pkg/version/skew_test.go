package version

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestComputeVersionSkew(t *testing.T) {
	cases := map[string]struct {
		centralVersion string
		sensorVersion  string
		expectedStatus storage.VersionSkewStatus
		expectedReason storage.VersionSkewReason
		expectedMin    string
		expectedMax    string
	}{
		"matching versions": {
			centralVersion: "4.7.0",
			sensorVersion:  "4.7.0",
			expectedStatus: storage.VersionSkewStatus_VERSION_SKEW_STATUS_MATCHING,
			expectedMin:    "4.5",
			expectedMax:    "4.7",
		},
		"matching X.Y different patch": {
			centralVersion: "4.7.1",
			sensorVersion:  "4.7.0",
			expectedStatus: storage.VersionSkewStatus_VERSION_SKEW_STATUS_MATCHING,
			expectedMin:    "4.5",
			expectedMax:    "4.7",
		},
		"compatible one minor behind": {
			centralVersion: "4.7.0",
			sensorVersion:  "4.6.0",
			expectedStatus: storage.VersionSkewStatus_VERSION_SKEW_STATUS_COMPATIBLE,
			expectedMin:    "4.5",
			expectedMax:    "4.7",
		},
		"compatible two minor behind": {
			centralVersion: "4.7.0",
			sensorVersion:  "4.5.0",
			expectedStatus: storage.VersionSkewStatus_VERSION_SKEW_STATUS_COMPATIBLE,
			expectedMin:    "4.5",
			expectedMax:    "4.7",
		},
		"incompatible three minor behind": {
			centralVersion: "4.7.0",
			sensorVersion:  "4.4.0",
			expectedStatus: storage.VersionSkewStatus_VERSION_SKEW_STATUS_INCOMPATIBLE,
			expectedReason: storage.VersionSkewReason_VERSION_SKEW_REASON_SENSOR_TOO_OLD,
			expectedMin:    "4.5",
			expectedMax:    "4.7",
		},
		"sensor one minor ahead": {
			centralVersion: "4.7.0",
			sensorVersion:  "4.8.0",
			expectedStatus: storage.VersionSkewStatus_VERSION_SKEW_STATUS_INCOMPATIBLE,
			expectedReason: storage.VersionSkewReason_VERSION_SKEW_REASON_SENSOR_AHEAD,
			expectedMin:    "4.5",
			expectedMax:    "4.7",
		},
		"sensor higher major version": {
			centralVersion: "4.7.0",
			sensorVersion:  "5.1.0",
			expectedStatus: storage.VersionSkewStatus_VERSION_SKEW_STATUS_INCOMPATIBLE,
			expectedReason: storage.VersionSkewReason_VERSION_SKEW_REASON_SENSOR_AHEAD,
			expectedMin:    "4.5",
			expectedMax:    "4.7",
		},
		"sensor lower major version": {
			centralVersion: "5.1.0",
			sensorVersion:  "4.10.0",
			expectedStatus: storage.VersionSkewStatus_VERSION_SKEW_STATUS_INCOMPATIBLE,
			expectedReason: storage.VersionSkewReason_VERSION_SKEW_REASON_SENSOR_TOO_OLD,
			expectedMin:    "5.0",
			expectedMax:    "5.1",
		},
		"empty sensor version": {
			centralVersion: "4.7.0",
			sensorVersion:  "",
			expectedStatus: storage.VersionSkewStatus_VERSION_SKEW_STATUS_UNSPECIFIED,
		},
		"empty central version": {
			centralVersion: "",
			sensorVersion:  "4.7.0",
			expectedStatus: storage.VersionSkewStatus_VERSION_SKEW_STATUS_UNSPECIFIED,
		},
		"dev build versions matching": {
			centralVersion: "4.7.x-19-gabcdef1234",
			sensorVersion:  "4.7.0",
			expectedStatus: storage.VersionSkewStatus_VERSION_SKEW_STATUS_MATCHING,
			expectedMin:    "4.5",
			expectedMax:    "4.7",
		},
		"low minor version clamps to zero": {
			centralVersion: "4.1.0",
			sensorVersion:  "4.0.0",
			expectedStatus: storage.VersionSkewStatus_VERSION_SKEW_STATUS_COMPATIBLE,
			expectedMin:    "4.0",
			expectedMax:    "4.1",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			result := ComputeVersionSkew(tc.centralVersion, tc.sensorVersion)
			assert.Equal(t, tc.expectedStatus, result.GetStatus())
			assert.Equal(t, tc.expectedReason, result.GetReason())
			if tc.expectedMin != "" {
				assert.Equal(t, tc.expectedMin, result.GetMinCompatibleSensorVersion())
			}
			if tc.expectedMax != "" {
				assert.Equal(t, tc.expectedMax, result.GetMaxCompatibleSensorVersion())
			}
		})
	}
}
