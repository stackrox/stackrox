package centralclient

import (
	"context"
	"os"
	"testing"

	"github.com/pkg/errors"
	installationDS "github.com/stackrox/rox/central/installation/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/grpc/client/authn/basic"
	"github.com/stackrox/rox/pkg/images/defaults"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/telemetry/phonehome"
	"github.com/stackrox/rox/pkg/telemetry/phonehome/telemeter"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/pkg/version"
	k8sVersion "k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var (
	apiWhiteList   = env.RegisterSetting("ROX_TELEMETRY_API_WHITELIST", env.AllowEmpty())
	userAgentsList = env.RegisterSetting("ROX_TELEMETRY_USERAGENT_LIST", env.AllowEmpty())

	client *centralClient
	once   sync.Once
	log    = logging.LoggerForModule()
)

type centralClient struct {
	*phonehome.Client

	campaignMux       sync.RWMutex
	telemetryCampaign phonehome.APICallCampaign
}

func newCentralClient(instanceId string) *centralClient {
	// Disable telemetry when running unit tests if no key is configured.
	if env.TelemetryStorageKey.Setting() == "" && testing.Testing() {
		return &centralClient{Client: &phonehome.Client{}}
	}

	if instanceId == "" {
		var err error
		instanceId, err = getInstanceId(installationDS.Singleton())
		if err != nil {
			log.Errorf("Failed to get central instance ID for telemetry configuration: %v.", err)
			return &centralClient{Client: &phonehome.Client{}}
		}
	}

	tenantID := env.TenantID.Setting()
	// Consider on-prem central a tenant of itself:
	if tenantID == "" {
		tenantID = instanceId
	}

	c := &centralClient{
		Client: &phonehome.Client{
			Config: &phonehome.Config{
				// Segment does not respect the processing order of events in a
				// batch. Setting BatchSize to 1, instead of default 250, may reduce
				// the number of (none) values, appearing on Amplitude charts, by
				// introducing a slight delay between consequent events.
				BatchSize:     1,
				ClientID:      instanceId,
				ClientName:    "Central",
				ClientVersion: version.GetMainVersion(),
				GroupType:     "Tenant",
				GroupID:       tenantID,
				StorageKey:    env.TelemetryStorageKey.Setting(),
				Endpoint:      env.TelemetryEndpoint.Setting(),
				PushInterval:  env.TelemetryFrequency.DurationSetting(),
			}}}

	c.AddInterceptorFuncs("API Call", c.apiCall(), addDefaultProps)

	return c
}

func getCentralDeploymentProperties() map[string]any {
	orchestrator := storage.ClusterType_KUBERNETES_CLUSTER.String()
	if env.Openshift.BooleanSetting() {
		orchestrator = storage.ClusterType_OPENSHIFT_CLUSTER.String()
	}
	// k8s apiserver is not accessible in cloud service environment.
	v := &k8sVersion.Info{GitVersion: "unknown"}
	if rc, err := rest.InClusterConfig(); err == nil {
		if clientset, err := kubernetes.NewForConfig(rc); err == nil {
			if serverVersion, err := clientset.ServerVersion(); err == nil {
				v = serverVersion
			}
		}
	}

	var imageFlavor string
	if _, ok := os.LookupEnv(defaults.ImageFlavorEnvName); ok {
		imageFlavor = defaults.GetImageFlavorNameFromEnv()
	}

	return map[string]any{
		"Image Flavor":       imageFlavor,
		"Central version":    version.GetMainVersion(),
		"Chart version":      version.GetChartVersion(),
		"Orchestrator":       orchestrator,
		"Kubernetes version": v.GitVersion,
		"Managed":            env.ManagedCentral.BooleanSetting(),
	}
}

func getInstanceId(ids installationDS.Store) (string, error) {
	if ids == nil {
		// There might be no installation info when running unit tests without
		// a database.
		return uuid.Nil.String(), nil
	}
	ii, _, err := ids.Get(
		sac.WithGlobalAccessScopeChecker(context.Background(),
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
				sac.ResourceScopeKeys(resources.InstallationInfo))))

	if err != nil || ii == nil {
		return "", errors.WithMessagef(err, "failed to get installation information")
	}
	return ii.Id, nil
}

// Singleton instance collects the central instance telemetry configuration from
// central Deployment labels and environment variables, installation store and
// orchestrator properties. The collected data is used for configuring the
// telemetry client. Returns nil if data collection is disabled.
func Singleton() *centralClient {
	once.Do(func() {
		if env.OfflineModeEnv.BooleanSetting() {
			return
		}

		utils.Must(permanentTelemetryCampaign.Compile())

		cfg := newCentralClient("")

		if !cfg.IsActive() || cfg.Reload() != nil {
			client = cfg
			return
		}

		props := getCentralDeploymentProperties()
		cfg.Gatherer().AddGatherer(func(ctx context.Context) (map[string]any, error) {
			return props, nil
		})

		log.Info("Central ID: ", cfg.ClientID)
		log.Info("Tenant ID: ", cfg.GroupID)
		log.Infof("API Telemetry ignored paths: %v", ignoredPaths)
		client = cfg
	})
	return client
}

// RegisterCentralClient adds call interceptors, adds central and admin user
// to the tenant group.
func (c *centralClient) RegisterCentralClient(gc *grpc.Config, basicAuthProviderID string) {
	if !c.IsActive() {
		return
	}
	gc.HTTPInterceptors = append(gc.HTTPInterceptors, c.GetHTTPInterceptor())
	gc.UnaryInterceptors = append(gc.UnaryInterceptors, c.GetGRPCInterceptor())

	// Central adds itself to the tenant group, with no group properties:
	c.Telemeter().Group(nil, telemeter.WithGroups(c.GroupType, c.GroupID))

	// registerAdminUser adds the local admin user to the tenant group.
	// This user is not added to the datastore like other users, so we need to add
	// it to the tenant group specifically.
	adminHash := c.HashUserID(basic.DefaultUsername, basicAuthProviderID)
	c.Telemeter().Group(nil, telemeter.WithUserID(adminHash), telemeter.WithGroups(c.GroupType, c.GroupID))
}

// Disable stops and disables the telemetry collection.
func (c *centralClient) Disable() {
	if !c.IsEnabled() {
		return
	}
	log.Info("Telemetry collection has been disabled.")
	c.Telemeter().Track("Telemetry Disabled", nil)
	c.Client.Disable()
}

// Enable enables and starts the telemetry collection.
func (c *centralClient) Enable() {
	if !c.IsActive() || c.IsEnabled() {
		return
	}
	c.Client.Enable()
	log.Info("Telemetry collection has been enabled.")
	c.Telemeter().Track("Telemetry Enabled", nil)
}
