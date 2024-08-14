package pods

import (
	"os"

	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/namespaces"
	"github.com/stackrox/rox/pkg/satoken"
)

var (
	log = logging.LoggerForModule()
)

const (
	// The corresponding environment variable is configured to contain pod namespace by sensor YAML/helm file using
	// the Kubernetes Downward API.
	nsEnvVar = "POD_NAMESPACE"
)

type HeuristicType int

const (
	// NoSATokenNamespace is the default option. It does *not* read the SAToken
	// file. For more information on why this mechanism is the default, see
	//     https://issues.redhat.com/browse/ROX-12349
	NoSATokenNamespace HeuristicType = iota
	// ConsiderSATokenNamespace switches the legacy mechanism which first reads the
	// SAToken file namespace. While this mechanism works _most_ of the time, on
	// our CI runners we saw the service account namespace set to `ci-op-<HASH`,
	// see this for more information
	//     https://github.com/openshift/ci-operator/blob/master/TEMPLATES.md
	//
	// This options is kept for now to avoid unexpected side effects but we aim
	// to remove it in the future once we ensure that the POD_NAMESPACE env var
	// is set correctly via the Kubernetes Downward API for all our pods, see
	//     https://github.com/kubernetes/kubernetes/blob/release-1.0/docs/user-guide/downward-api.md
	ConsiderSATokenNamespace
)

// GetPodNamespace is a heuristic to determine in which namespace a given Pod runs.
func GetPodNamespace(heuristic HeuristicType) string {
	var sensorNamespace string
	if heuristic == ConsiderSATokenNamespace {
		var err error
		sensorNamespace, err = satoken.LoadNamespaceFromFile()
		if err != nil {
			log.Errorf("Failed to determine namespace from service account token file: %s", err)
		}
	}

	if sensorNamespace == "" {
		sensorNamespace = os.Getenv(nsEnvVar)
	}

	if sensorNamespace == "" {
		sensorNamespace = namespaces.StackRox
		log.Warnf("%s environment variable is unset/empty, using %q as fallback for sensor namespace",
			nsEnvVar, namespaces.StackRox)
	}
	return sensorNamespace
}
