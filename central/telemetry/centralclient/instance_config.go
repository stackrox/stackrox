package centralclient

import (
	"context"

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
	"github.com/stackrox/rox/pkg/version"
	k8sVersion "k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var (
	apiWhiteList   = env.RegisterSetting("ROX_TELEMETRY_API_WHITELIST", env.AllowEmpty())
	userAgentsList = env.RegisterSetting("ROX_TELEMETRY_USERAGENT_LIST", env.AllowEmpty())

	config *centralConfig
	once   sync.Once
	log    = logging.LoggerForModule()
)

type centralConfig struct {
	*phonehome.Config

	campaignMux       sync.RWMutex
	telemetryCampaign phonehome.APICallCampaign
}

func makeCentralConfig(instanceId string) *centralConfig {
	tenantID := env.TenantID.Setting()
	// Consider on-prem central a tenant of itself:
	if tenantID == "" {
		tenantID = instanceId
	}

	cfg := &centralConfig{Config: &phonehome.Config{
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
	}}

	interceptors := map[string][]phonehome.Interceptor{
		"API Call": {cfg.apiCall(), addDefaultProps},
	}

	for event, funcs := range interceptors {
		for _, f := range funcs {
			cfg.AddInterceptorFunc(event, f)
		}
	}

	return cfg
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

	return map[string]any{
		"Image Flavor":       defaults.GetImageFlavorNameFromEnv(),
		"Central version":    version.GetMainVersion(),
		"Chart version":      version.GetChartVersion(),
		"Orchestrator":       orchestrator,
		"Kubernetes version": v.GitVersion,
		"Managed":            env.ManagedCentral.BooleanSetting(),
	}
}

func getInstanceId() (string, error) {
	ii, _, err := installationDS.Singleton().Get(
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
func Singleton() *centralConfig {
	once.Do(func() {
		if env.OfflineModeEnv.BooleanSetting() {
			return
		}

		utils.Must(permanentTelemetryCampaign.Compile())

		iid, err := getInstanceId()
		if err != nil {
			log.Errorf("Failed to get central instance ID for telemetry configuration: %v.", err)
			return
		}

		cfg := makeCentralConfig(iid)

		if !cfg.IsActive() || cfg.Reload() != nil {
			return
		}

		props := getCentralDeploymentProperties()
		cfg.Gatherer().AddGatherer(func(ctx context.Context) (map[string]any, error) {
			return props, nil
		})

		log.Info("Central ID: ", cfg.ClientID)
		log.Info("Tenant ID: ", cfg.GroupID)
		log.Infof("API Telemetry ignored paths: %v", ignoredPaths)
		config = cfg
	})
	return config
}

// RegisterCentralClient adds call interceptors, adds central and admin user
// to the tenant group.
func (cfg *centralConfig) RegisterCentralClient(gc *grpc.Config, basicAuthProviderID string) {
	if !cfg.IsActive() {
		return
	}
	gc.HTTPInterceptors = append(gc.HTTPInterceptors, cfg.GetHTTPInterceptor())
	gc.UnaryInterceptors = append(gc.UnaryInterceptors, cfg.GetGRPCInterceptor())

	// Central adds itself to the tenant group, with no group properties:
	cfg.Telemeter().Group(nil, telemeter.WithGroups(cfg.GroupType, cfg.GroupID))

	// registerAdminUser adds the local admin user to the tenant group.
	// This user is not added to the datastore like other users, so we need to add
	// it to the tenant group specifically.
	adminHash := cfg.HashUserID(basic.DefaultUsername, basicAuthProviderID)
	cfg.Telemeter().Group(nil, telemeter.WithUserID(adminHash), telemeter.WithGroups(cfg.GroupType, cfg.GroupID))
}

// OptOut stops and disables the telemetry collection.
func (cfg *centralConfig) OptOut() {
	if !cfg.IsEnabled() {
		return
	}
	log.Info("Telemetry collection has been disabled.")
	cfg.Telemeter().Track("Telemetry Disabled", nil)
	cfg.Disable()
}

// OptIn enables and starts the telemetry collection.
func (cfg *centralConfig) OptIn() {
	if !cfg.IsActive() || cfg.IsEnabled() {
		return
	}
	cfg.Enable()
	log.Info("Telemetry collection has been enabled.")
	cfg.Telemeter().Track("Telemetry Enabled", nil)
}
