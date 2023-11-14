package centralclient

import (
	"context"
	"strings"

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
	"github.com/stackrox/rox/pkg/version"
	k8sVersion "k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var (
	apiWhiteList = env.RegisterSetting("ROX_TELEMETRY_API_WHITELIST", env.AllowEmpty())

	config *phonehome.Config
	once   sync.Once
	log    = logging.LoggerForModule()

	startMux sync.RWMutex
	enabled  bool
)

func getInstanceConfig() (*phonehome.Config, map[string]any, error) {
	if env.OfflineModeEnv.BooleanSetting() {
		return nil, nil, nil
	}
	key, err := phonehome.GetKey(env.TelemetryStorageKey.Setting(),
		env.TelemetryConfigURL.Setting())
	if key == "" || err != nil {
		return nil, nil, err
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

	trackedPaths = strings.Split(apiWhiteList.Setting(), ",")

	orchestrator := storage.ClusterType_KUBERNETES_CLUSTER.String()
	if env.Openshift.BooleanSetting() {
		orchestrator = storage.ClusterType_OPENSHIFT_CLUSTER.String()
	}

	ii, _, err := store.Singleton().Get(
		sac.WithGlobalAccessScopeChecker(context.Background(),
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
				sac.ResourceScopeKeys(resources.InstallationInfo))))

	if err != nil || ii == nil {
		return nil, nil, errors.Wrap(err, "cannot get installation information")
	}
	centralID := ii.Id

	tenantID := env.TenantID.Setting()
	// Consider on-prem central a tenant of itself:
	if tenantID == "" {
		tenantID = centralID
	}

	return &phonehome.Config{
			ClientID:     centralID,
			ClientName:   "Central",
			GroupType:    "Tenant",
			GroupID:      tenantID,
			StorageKey:   key,
			Endpoint:     env.TelemetryEndpoint.Setting(),
			PushInterval: env.TelemetryFrequency.DurationSetting(),
		}, map[string]any{
			"Image Flavor":       defaults.GetImageFlavorNameFromEnv(),
			"Central version":    version.GetMainVersion(),
			"Chart version":      version.GetChartVersion(),
			"Orchestrator":       orchestrator,
			"Kubernetes version": v.GitVersion,
			"Managed":            env.ManagedCentral.BooleanSetting(),
		}, nil
}

// InstanceConfig collects the central instance telemetry configuration from
// central Deployment labels and environment variables, installation store and
// orchestrator properties. The collected data is used for configuring the
// telemetry client. Returns nil if data collection is disabled.
func InstanceConfig() *phonehome.Config {
	once.Do(func() {
		var err error
		var props map[string]any
		config, props, err = getInstanceConfig()
		if err != nil {
			log.Errorf("Failed to get telemetry configuration: %v", err)
			return
		}
		if config == nil {
			log.Info("Phonehome telemetry collection disabled")
			return
		}
		log.Info("Central ID: ", config.ClientID)
		log.Info("Tenant ID: ", config.GroupID)
		log.Info("API path telemetry enabled for: ", trackedPaths)

		config.Gatherer().AddGatherer(func(ctx context.Context) (map[string]any, error) {
			return props, nil
		})
	})
	startMux.RLock()
	defer startMux.RUnlock()
	if !enabled {
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
	log.Info("Telemetry collection has been disabled")
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
	cfg.Gatherer().Start(telemeter.WithGroups(cfg.GroupType, cfg.GroupID))
	enabled = true
	log.Info("Telemetry collection has been enabled")
	cfg.Telemeter().Track("Telemetry Enabled", nil)
	return cfg
}
