package sensorupgrader

import (
	"os"

	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/namespaces"
)

var (
	log = logging.LoggerForModule()
)

// GetSensorNamespace is a heuristic to determine in what namespace the Sensor runs.
func GetSensorNamespace() string {
	// The corresponding environment variable is configured to contain pod namespace by sensor YAML/helm file.
	const nsEnvVar = "POD_NAMESPACE"
	sensorNamespace := os.Getenv(nsEnvVar)

	if sensorNamespace == "" {
		sensorNamespace = namespaces.StackRox
		log.Warnf("%s environment variable is unset/empty, using %q as fallback for sensor namespace",
			nsEnvVar, namespaces.StackRox)
	}
	return sensorNamespace
}
