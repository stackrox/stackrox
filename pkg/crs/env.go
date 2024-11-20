package crs

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
)
