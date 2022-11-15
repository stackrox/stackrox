package marketing

import (
	"context"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/version"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const annotation = "stackrox.com/telemetry-apipaths"

// GetDeviceConfig collects the central instance telemetry configuration.
func GetDeviceConfig() (*Config, error) {
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

	di := clientset.AppsV1().Deployments("stackrox")
	opts := v1.GetOptions{}
	d, err := di.Get(context.Background(), "central", opts)
	if err != nil {
		return nil, errors.Wrap(err, "cannot get central deployment")
	}
	paths := d.GetAnnotations()[annotation]

	return &Config{
		ID:       string(d.GetUID()),
		APIPaths: strings.Split(paths, ","),
		Identity: map[string]any{
			"Central version":    version.GetMainVersion(),
			"Chart version":      version.GetChartVersion(),
			"Orchestrator":       orchestrator,
			"Kubernetes version": v.GitVersion,
		},
	}, nil
}
