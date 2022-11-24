package marketing

import (
	"context"
	"crypto/sha256"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/version"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const annotation = "rhacs.redhat.com/telemetry-apipaths"
const orgID = "rhacs.redhat.com/organization-id"
const tenantID = "rhacs.redhat.com/cs-tenant-id"

var config *Config

// GetInstanceConfig collects the central instance telemetry configuration from
// central Deployment annotations and orchestrator properties. The collected
// data is used for instance identification.
func GetInstanceConfig() (*Config, error) {
	if config != nil {
		return config, nil
	}
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
	orchestrator := storage.ClusterType_KUBERNETES_CLUSTER.String()
	if env.OpenshiftAPI.BooleanSetting() {
		orchestrator = storage.ClusterType_OPENSHIFT_CLUSTER.String()
	}

	di := clientset.AppsV1().Deployments(env.Namespace.Setting())
	opts := v1.GetOptions{}
	d, err := di.Get(context.Background(), "central", opts)
	if err != nil {
		return nil, errors.Wrap(err, "cannot get central deployment")
	}
	paths, ok := d.GetAnnotations()[annotation]
	if !ok {
		paths = "*"
	}

	config = &Config{
		ID:       string(d.GetUID()),
		OrgID:    d.GetAnnotations()[orgID],
		TenantID: d.GetAnnotations()[tenantID],
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

// hashUserID anonymizes user ID so that it can be sent to the external
// telemetry storage for marketing data analysis.
func hashUserID(id string) string {
	isha := sha256.New()
	isha.Write([]byte(id))
	return string(isha.Sum(nil))
}

// GetUserMetadata returns user identification information map, including
// central instance identificaion, for being used by the frontend when reporting
// marketing telemetry data.
func (config *Config) GetUserMetadata(id authn.Identity) map[string]string {
	metadata := map[string]string{
		"UserId":         "unauthenticated",
		"CentralId":      config.ID,
		"OrganizationId": config.OrgID,
		"StorageKeyV1":   env.TelemetryStorageKey.Setting(),
	}
	if id != nil {
		metadata["UserId"] = hashUserID(id.UID())
	}
	return metadata
}
