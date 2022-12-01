package phonehome

import (
	"context"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
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
	tenantIDAnnotation = "rhacs.redhat.com/tenant-id"
)

var (
	config = &Config{
		CentralID: "11102e5e-ca16-4f2b-8d2e-e9e04e8dc531",
		APIPaths:  set.NewFrozenSet[string](),
	}
	once sync.Once
	log  = logging.LoggerForModule()
)

func getInstanceConfig(clientset *kubernetes.Clientset) (*Config, error) {
	v, err := clientset.ServerVersion()
	if err != nil {
		return nil, err
	}

	deployments := clientset.AppsV1().Deployments(env.Namespace.Setting())
	central, err := deployments.Get(context.Background(), "central", v1.GetOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "cannot get central deployment")
	}

	centralAnnotations := central.GetAnnotations()
	paths, ok := centralAnnotations[apiPathsAnnotation]
	if !ok {
		paths = "*"
	}

	orchestrator := storage.ClusterType_KUBERNETES_CLUSTER.String()
	if env.OpenshiftAPI.BooleanSetting() {
		orchestrator = storage.ClusterType_OPENSHIFT_CLUSTER.String()
	}

	centralID := string(central.GetUID())
	tenantID := centralAnnotations[tenantIDAnnotation]
	// Consider on-prem central a tenant of itself:
	if tenantID == "" {
		tenantID = centralID
	}

	config = &Config{
		CentralID: centralID,
		TenantID:  tenantID,
		APIPaths:  set.NewFrozenSet(strings.Split(paths, ",")...),
		Properties: map[string]any{
			"Central version":    version.GetMainVersion(),
			"Chart version":      version.GetChartVersion(),
			"Orchestrator":       orchestrator,
			"Kubernetes version": v.GitVersion,
		},
	}

	return config, nil
}

// InstanceConfig collects the central instance telemetry configuration from
// central Deployment annotations and orchestrator properties. The collected
// data is used for instance identification.
func InstanceConfig() *Config {
	once.Do(func() {
		rc, err := rest.InClusterConfig()
		if err != nil {
			log.Errorf("Cannot create k8s config: %v. Using hardcoded values.", err)
			return
		}
		clientset, err := kubernetes.NewForConfig(rc)
		if err != nil {
			log.Errorf("Cannot create k8s clientset: %v. Using hardcoded values.", err)
			return
		}
		if config, err = getInstanceConfig(clientset); err != nil {
			log.Error("Failed to get telemetry configuration: ", err)
		} else {
			log.Info("Central ID:", config.CentralID)
			log.Info("Tenant ID:", config.TenantID)
			log.Info("API path telemetry enabled for: ", config.APIPaths)
		}
	})
	return config
}
