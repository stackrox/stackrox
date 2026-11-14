package crs

import "github.com/stackrox/rox/pkg/env"

const (
	// LegacySensorServiceCertEnvName contains the environment variable name used by the CRS-flow to check for the
	// existence of a legacy sensor service certificate (e.g. coming from an init-bundle).
	// Needs to be kept in sync with
	// image/templates/helm/stackrox-secured-cluster/templates/sensor.yaml.htpl:spec.template.spec.initContainers[0].env.
	LegacySensorServiceCertEnvName = "ROX_LEGACY_SENSOR_SERVICE_CERT"

	// SensorServiceCertEnvName contains the environment variable name used by the CRS-flow to check for the
	// existence of a new-style sensor service certificate (e.g. coming from a CRS-based registration).
	// Needs to be kept in sync with
	// image/templates/helm/stackrox-secured-cluster/templates/sensor.yaml.htpl:spec.template.spec.initContainers[0].env.
	SensorServiceCertEnvName = "ROX_SENSOR_SERVICE_CERT"

	// crsFilePathEnvName is the env variable name for the CRS file.
	crsFilePathEnvName = "ROX_CRS_FILE"

	// FakeCRSEnvName is the env variable name for enabling fake CRS mode.
	FakeCRSEnvName = "ROX_FAKE_CRS"

	// defaultCRSFilePath is where the Cluster Registration Secret is expected by default.
	defaultCRSFilePath = "/run/secrets/stackrox.io/crs/crs"
)

var (
	// crsFilePathSetting allows configuring the CRS from the environment.
	crsFilePathSetting = env.RegisterSetting(crsFilePathEnvName, env.WithDefault(defaultCRSFilePath))
	// fakeCRSSetting allows enabling fake CRS mode for testing.
	fakeCRSSetting = env.RegisterSetting(FakeCRSEnvName)
)

// crsFilePath returns the path where the CRS certificate is stored.
func crsFilePath() string {
	return crsFilePathSetting.Setting()
}

// IsFakeCRSEnabled returns true if fake CRS mode is enabled.
func IsFakeCRSEnabled() bool {
	return fakeCRSSetting.Setting() != ""
}
