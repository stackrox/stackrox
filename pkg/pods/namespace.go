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
	// the Kubernetes Downward API
	nsEnvVar = "POD_NAMESPACE"
)

type HeuristicType int

const (
	// NoSATokenNamespace NO_SA_TOKEN if this is set the SAToken namespace will not be queried
	// This should be the default option unless we can be sure that the service account namespace contained within the
	// SAToken namespace file indeed matches the namespace the pod is in. This is the case _most_ of the time but not
	// always, eg. in our CI runners the service account namespace is set to <ci-op-HASH> instead, see
	// https://github.com/openshift/ci-operator/blob/master/TEMPLATES.md
	NoSATokenNamespace HeuristicType = iota
	// UseSATokenNamespace USE_SA_TOKEN if this is set the SAToken will be queried first
	// This option is here for legacy reasons since it has been how the namespace has been determined in CreateSensor
	// since 2019, and it's unclear if removing this method would have side effects. In principle discarding the
	// SAToken namespace method seems best as long as we can ensure that the POD_NAMESPACE env var is set to the
	// namespace of its pod via the Kubernetes Downward API, see
	// https://github.com/kubernetes/kubernetes/blob/release-1.0/docs/user-guide/downward-api.md
	UseSATokenNamespace
)

// GetPodNamespace is a heuristic to determine in what namespace a given Pod runs.
func GetPodNamespace(heuristic HeuristicType) string {
	sensorNamespace := ""
	if heuristic == UseSATokenNamespace {
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
