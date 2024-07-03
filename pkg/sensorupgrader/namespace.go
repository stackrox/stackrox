package sensorupgrader

import (
	"os"

	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/namespaces"
	"github.com/stackrox/rox/pkg/satoken"
)

var (
	log = logging.LoggerForModule()
)

// GetSensorNamespace tries to read the sensor namespace out of the service account token dir, POD_NAMESPACE if that
// fails, and defaults to Stackrox otherwise.
func GetSensorNamespace() string {
	// This attempts to load the namespace from the file serviceaccount/namespace
	sensorNamespace, err := satoken.LoadNamespaceFromFile()
	if err != nil {
		log.Errorf("Failed to determine namespace from service account token file: %s", err)
	}
	if sensorNamespace == "" {
		// This environment variable is configured to contain pod namespace by sensor YAML/helm file.
		sensorNamespace = os.Getenv("POD_NAMESPACE")
	}
	if sensorNamespace == "" {
		sensorNamespace = namespaces.StackRox
		log.Warnf("Unable to determine Sensor namespace, defaulting to %s", sensorNamespace)
	}
	return sensorNamespace
}
