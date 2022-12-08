package centralclient

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/installation/store"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/telemetry/phonehome"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	apiPathsAnnotation = "rhacs.redhat.com/telemetry-apipaths"
	tenantIDLabel      = "rhacs.redhat.com/tenant"
)

var (
	config = &phonehome.Config{
		ClientID: "11102e5e-ca16-4f2b-8d2e-e9e04e8dc531",
	}
	once sync.Once
	log  = logging.LoggerForModule()
)

func getInstanceConfig() (*phonehome.Config, error) {
	rc, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	clientset, err := kubernetes.NewForConfig(rc)
	if err != nil {
		return nil, err
	}

	deployments := clientset.AppsV1().Deployments(env.Namespace.Setting())
	central, err := deployments.Get(context.Background(), "central", v1.GetOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "cannot get central deployment")
	}

	ii, _, err := store.Singleton().Get(sac.WithAllAccess(context.Background()))
	if err != nil || ii == nil {
		return nil, errors.Wrap(err, "cannot get installation information")
	}
	centralID := ii.Id

	tenantID := central.GetLabels()[tenantIDLabel]
	// Consider on-prem central a tenant of itself:
	if tenantID == "" {
		tenantID = centralID
	}

	return &phonehome.Config{
		ClientID:   centralID,
		ClientName: "Central",
		GroupID:    tenantID,
	}, nil
}

// InstanceConfig collects the central instance telemetry configuration from
// central Deployment annotations and orchestrator properties. The collected
// data is used for instance identification.
func InstanceConfig() *phonehome.Config {
	once.Do(func() {
		if !phonehome.Enabled() {
			log.Info("Phonehome telemetry collection disabled.")
			return
		}
		cfg, err := getInstanceConfig()
		if err != nil {
			log.Errorf("Failed to get telemetry configuration: %v. Using hardcoded values.", err)
			return
		}
		config = cfg
		log.Info("Central ID: ", config.ClientID)
		log.Info("Tenant ID: ", config.GroupID)
	})
	return config
}
