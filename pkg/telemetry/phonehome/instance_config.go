package phonehome

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
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
		Identity: map[string]any{
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
		var err error
		if config, err = getInstanceConfig(); err != nil {
			log.Error("Failed to get telemetry configuration: ", err)
		} else {
			log.Info("Central ID:", config.CentralID)
			log.Info("Tenant ID:", config.TenantID)
			log.Info("API path telemetry enabled for: ", config.APIPaths)
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
