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
func GetSensorNamespace(params ...bool) string {
	// Operator installs and some CI runs set a NAMESPACE env which needs to be adhered to so tests don't fail
	sensorNamespace := os.Getenv("TEST_NAMESPACE")

	verbose := false
	if len(params) > 0 && params[0] == true {
		verbose = true
	}
	if verbose {
		log.Infof("sensorNamespace is %s after running 'os.Getenv(\"TEST_NAMESPACE\")'", sensorNamespace)
	}
	// This attempts to load the namespace from the file serviceaccount/namespace
	if sensorNamespace == "" {
		var err error
		sensorNamespace, err = satoken.LoadNamespaceFromFile()
		if err != nil {
			log.Errorf("Failed to determine namespace from service account token file: %s", err)
		}
		if verbose {
			log.Infof("sensorNamespace is %s after running 'satoken.LoadNamespaceFromFile()'", sensorNamespace)
		}
	}
	if sensorNamespace == "" {
		// This environment variable is configured to contain pod namespace by sensor YAML/helm file.
		sensorNamespace = os.Getenv("POD_NAMESPACE")
		if verbose {
			log.Infof("sensorNamespace is %s after running 'os.Getenv(\"POD_NAMESPACE\")'", sensorNamespace)
		}
	}
	if sensorNamespace == "" {
		sensorNamespace = namespaces.StackRox
		log.Warnf("Unable to determine Sensor namespace, defaulting to %s", sensorNamespace)
	}
	return sensorNamespace
}
