package centralclient

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/installation/store"
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

	config *phonehome.Config
	once   sync.Once
	log    = logging.LoggerForModule()

	startMux   sync.RWMutex
	enabled    bool
	instanceId string
)

func getInstanceConfig(key string) (*phonehome.Config, map[string]any) {
	// k8s apiserver is not accessible in cloud service environment.
	v := &k8sVersion.Info{GitVersion: "unknown"}
	if rc, err := rest.InClusterConfig(); err == nil {
		if clientset, err := kubernetes.NewForConfig(rc); err == nil {
			if serverVersion, err := clientset.ServerVersion(); err == nil {
				v = serverVersion
			}
		}
	}

	orchestrator := storage.ClusterType_KUBERNETES_CLUSTER.String()
	if env.Openshift.BooleanSetting() {
		orchestrator = storage.ClusterType_OPENSHIFT_CLUSTER.String()
	}

	tenantID := env.TenantID.Setting()
	// Consider on-prem central a tenant of itself:
	if tenantID == "" {
		tenantID = instanceId
	}

	return &phonehome.Config{
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
			StorageKey:    key,
			Endpoint:      env.TelemetryEndpoint.Setting(),
			PushInterval:  env.TelemetryFrequency.DurationSetting(),
		}, map[string]any{
			"Image Flavor":       defaults.GetImageFlavorNameFromEnv(),
			"Central version":    version.GetMainVersion(),
			"Chart version":      version.GetChartVersion(),
			"Orchestrator":       orchestrator,
			"Kubernetes version": v.GitVersion,
			"Managed":            env.ManagedCentral.BooleanSetting(),
		}
}

func getInstanceId() error {
	startMux.Lock()
	defer startMux.Unlock()
	if instanceId != "" {
		return nil
	}

	ii, _, err := store.Singleton().Get(
		sac.WithGlobalAccessScopeChecker(context.Background(),
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
				sac.ResourceScopeKeys(resources.InstallationInfo))))

	if err != nil || ii == nil {
		return errors.WithMessagef(err, "failed to get installation information")
	}
	instanceId = ii.Id
	return nil
}

// InstanceConfig collects the central instance telemetry configuration from
// central Deployment labels and environment variables, installation store and
// orchestrator properties. The collected data is used for configuring the
// telemetry client. Returns nil if data collection is disabled.
func InstanceConfig() *phonehome.Config {
	once.Do(func() {
		utils.Must(permanentTelemetryCampaign.Compile())
		if _, err := applyConfig(); err != nil {
			log.Errorf("Failed to apply telemetry configuration: %v.", err)
			return
		}
		startMux.RLock()
		defer startMux.RUnlock()
		if config != nil {
			log.Info("Central ID: ", config.ClientID)
			log.Info("Tenant ID: ", config.GroupID)
			log.Infof("API Telemetry ignored paths: %v", ignoredPaths)
		}
	})
	startMux.RLock()
	defer startMux.RUnlock()
	if !enabled {
		log.Info("Telemetry collection is disabled")
		// This will make InstanceConfig().Enabled() to return false, while
		// keeping the config configured for eventual Start().
		return nil
	}
	return config
}

// GetConfig returns the client configuration, whether the collection is enabled
// or not. Returns nil if the client is not configured and therefore the data
// collection cannot be enabled.
func GetConfig() *phonehome.Config {
	InstanceConfig()
	return config
}

// RegisterCentralClient adds call interceptors, adds central and admin user
// to the tenant group.
func RegisterCentralClient(gc *grpc.Config, basicAuthProviderID string) {
	cfg := config
	if !cfg.Enabled() {
		return
	}
	registerInterceptors(gc)
	// Central adds itself to the tenant group, with no group properties:
	cfg.Telemeter().Group(nil, telemeter.WithGroups(cfg.GroupType, cfg.GroupID))
	registerAdminUser(basicAuthProviderID)
}

func registerInterceptors(gc *grpc.Config) {
	cfg := config
	gc.HTTPInterceptors = append(gc.HTTPInterceptors, cfg.GetHTTPInterceptor())
	gc.UnaryInterceptors = append(gc.UnaryInterceptors, cfg.GetGRPCInterceptor())
}

// registerAdminUser adds the local admin user to the tenant group.
// This user is not added to the datastore like other users, so we need to add
// it to the tenant group specifically.
func registerAdminUser(basicAuthProviderID string) {
	cfg := config
	adminHash := cfg.HashUserID(basic.DefaultUsername, basicAuthProviderID)
	cfg.Telemeter().Group(nil, telemeter.WithUserID(adminHash), telemeter.WithGroups(cfg.GroupType, cfg.GroupID))
}

// Disable stops and disables the telemetry collection.
func Disable() {
	startMux.Lock()
	defer startMux.Unlock()
	cfg := config
	if !enabled || !cfg.Enabled() {
		return
	}
	cfg.Gatherer().Stop()
	cfg.RemoveInterceptors()
	enabled = false
	log.Info("Telemetry collection has been disabled.")
	cfg.Telemeter().Track("Telemetry Disabled", nil)
}

// Enable enables and starts the telemetry collection.
func Enable() *phonehome.Config {
	// Prepare the configuration.
	InstanceConfig()

	startMux.Lock()
	defer startMux.Unlock()
	// Use config as InstanceConfig may return nil for not yet enabled instance.
	cfg := config
	if !cfg.Enabled() {
		// Cannot enable without proper configuration.
		return nil
	}
	if enabled {
		return cfg
	}
	cfg.RemoveInterceptors()
	for event, funcs := range interceptors {
		for _, f := range funcs {
			cfg.AddInterceptorFunc(event, f)
		}
	}
	cfg.Gatherer().Start(
		telemeter.WithGroups(cfg.GroupType, cfg.GroupID),
		// Don't capture the time, but call WithNoDuplicates on every gathering
		// iteration, so that the time is updated.
		func(co *telemeter.CallOptions) {
			// Issue a possible duplicate only once a day as a heartbeat.
			telemeter.WithNoDuplicates(time.Now().Format(time.DateOnly))(co)
		},
	)
	enabled = true
	log.Info("Telemetry collection has been enabled.")
	cfg.Telemeter().Track("Telemetry Enabled", nil)
	return cfg
}
