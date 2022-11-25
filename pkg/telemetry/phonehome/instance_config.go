package phonehome

import (
	"context"
	"crypto/sha256"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/version"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	annotation = "rhacs.redhat.com/telemetry-apipaths"
	orgID      = "rhacs.redhat.com/organization-id"
	tenantID   = "rhacs.redhat.com/cs-tenant-id"
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

	paths, ok := central.GetAnnotations()[annotation]
	if !ok {
		paths = "*"
	}

	orchestrator := storage.ClusterType_KUBERNETES_CLUSTER.String()
	if env.OpenshiftAPI.BooleanSetting() {
		orchestrator = storage.ClusterType_OPENSHIFT_CLUSTER.String()
	}

	config = &Config{
		ID:       string(central.GetUID()),
		OrgID:    central.GetAnnotations()[orgID],
		TenantID: central.GetAnnotations()[tenantID],
		APIPaths: strings.Split(paths, ","),
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
		}
	})
	return config
}

// hashUserID anonymizes user ID so that it can be sent to the external
// telemetry storage for product data analysis.
func hashUserID(id string) string {
	sha := sha256.New()
	_, _ = sha.Write([]byte(id))
	return string(sha.Sum(nil))
}

// GetUserMetadata returns user identification information map, including
// central instance ID, for being used by the frontend when reporting
// product telemetry data.
func (config *Config) GetUserMetadata(id authn.Identity) map[string]string {
	metadata := map[string]string{
		"UserId":         "unauthenticated",
		"CentralId":      config.ID,
		"OrganizationId": config.OrgID,
		"TenantId":       config.TenantID,
	}
	if id != nil {
		metadata["UserId"] = hashUserID(id.UID())
	}
	return metadata
}
