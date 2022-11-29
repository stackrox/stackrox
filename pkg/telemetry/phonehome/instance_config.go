package phonehome

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/version"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	apiPathsAnnotation = "rhacs.redhat.com/telemetry-apipaths"
	csIDAnnotation     = "rhacs.redhat.com/cs-identity"
)

var (
	config *Config
	once   sync.Once
	log    = logging.LoggerForModule()
)

func getInstanceConfig() (*Config, error) {
	rc, err := rest.InClusterConfig()
	if err != nil {
		return nil, errors.Wrap(err, "cannot create k8s config")
	}
	clientset, err := kubernetes.NewForConfig(rc)
	if err != nil {
		return nil, errors.Wrap(err, "cannot create k8s clientset")
	}
	v, err := clientset.ServerVersion()
	if err != nil {
		return nil, err
	}

	deployments := clientset.AppsV1().Deployments(env.Namespace.Setting())
	central, err := deployments.Get(context.Background(), "central", v1.GetOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "cannot get central deployment")
	}

	paths, ok := central.GetAnnotations()[apiPathsAnnotation]
	if !ok {
		paths = "*"
	}

	orchestrator := storage.ClusterType_KUBERNETES_CLUSTER.String()
	if env.OpenshiftAPI.BooleanSetting() {
		orchestrator = storage.ClusterType_OPENSHIFT_CLUSTER.String()
	}

	config = &Config{
		CentralID: string(central.GetUID()),
		APIPaths:  set.NewFrozenSet(strings.Split(paths, ",")...),
		Identity: map[string]any{
			"Central version":    version.GetMainVersion(),
			"Chart version":      version.GetChartVersion(),
			"Orchestrator":       orchestrator,
			"Kubernetes version": v.GitVersion,
		},
	}

	// Add Cloud Services identity properties to the central identity.
	if props := central.GetAnnotations()[csIDAnnotation]; props != "" {
		if err = json.Unmarshal(([]byte)(props), &config.CSProperties); err != nil {
			log.Errorf("Failed to unmarshal %s annotation: %v", csIDAnnotation, err)
		} else if config.CSProperties != nil {
			for k, v := range config.CSProperties {
				config.Identity[k] = v
			}
		}
	}

	return config, nil
}

// InstanceConfig collects the central instance telemetry configuration from
// central Deployment annotations and orchestrator properties. The collected
// data is used for instance identification.
func InstanceConfig() *Config {
	once.Do(func() {
		var err error
		if config, err = getInstanceConfig(); err != nil {
			log.Error("Failed to get telemetry configuration: ", err)
		} else {
			log.Info("Telemetry device ID:", config.CentralID)
			log.Info("API path telemetry enabled for: ", config.APIPaths)
			if config.CSProperties != nil {
				log.Info("Cloud Services identity: ", config.CSProperties)
			}
		}
	})
	return config
}

// HashUserID anonymizes user ID so that it can be sent to the external
// telemetry storage for product data analysis.
func HashUserID(id string) string {
	sha := sha256.New()
	_, _ = sha.Write([]byte(id))
	return base64.StdEncoding.EncodeToString(sha.Sum(nil))
}

// GetUserMetadata returns user identification information map, including
// central instance ID, for being used by the frontend when reporting
// product telemetry data, as well as Cloud Services identity properties.
func (config *Config) GetUserMetadata(id authn.Identity) map[string]string {
	metadata := map[string]string{
		"UserId":    "unauthenticated",
		"CentralId": config.CentralID,
	}
	if id != nil {
		metadata["UserId"] = HashUserID(id.UID())
	}
	for k, v := range config.CSProperties {
		metadata[k] = v
	}
	return metadata
}
