package pods

import (
	"os"

	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/namespaces"
)

var (
	log = logging.LoggerForModule()
)

const (
	// The corresponding environment variable is configured to contain pod namespace by sensor YAML/helm file using
	// the Kubernetes Downward API, see
	// https://github.com/kubernetes/kubernetes/blob/release-1.0/docs/user-guide/downward-api.md
	nsEnvVar = "POD_NAMESPACE"
)

// GetPodNamespace is a heuristic to determine in what namespace a given Pod runs.
func GetPodNamespace() string {
	sensorNamespace := os.Getenv(nsEnvVar)

	if sensorNamespace == "" {
		sensorNamespace = namespaces.StackRox
		log.Warnf("%s environment variable is unset/empty, using %q as fallback for sensor namespace",
			nsEnvVar, namespaces.StackRox)
	}
	return sensorNamespace
}
