package centralclient

import (
	"context"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/installation/store"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/grpc/authn/tokenbased"
	"github.com/stackrox/rox/pkg/grpc/client/authn/basic"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/telemetry/phonehome"
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
)

func getInstanceConfig() (*phonehome.Config, map[string]any, error) {
	key := env.TelemetryStorageKey.Setting()
	if key == "" {
		return nil, nil, nil
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
			GroupID:      tenantID,
			StorageKey:   key,
			Endpoint:     env.TelemetryEndpoint.Setting(),
			PushInterval: env.TelemetryFrequency.DurationSetting(),
		}, map[string]any{
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
// telemetry client.
func InstanceConfig() *phonehome.Config {
	once.Do(func() {
		var err error
		var props map[string]any
		config, props, err = getInstanceConfig()
		if err != nil {
			log.Errorf("Failed to get telemetry configuration: %v.", err)
			return
		}
		if config == nil {
			log.Info("Phonehome telemetry collection disabled.")
			return
		}
		log.Info("Central ID: ", config.ClientID)
		log.Info("Tenant ID: ", config.GroupID)
		log.Info("API path telemetry enabled for: ", trackedPaths)

		for event, funcs := range interceptors {
			for _, f := range funcs {
				config.AddInterceptorFunc(event, f)
			}
		}

		config.Gatherer().AddGatherer(func(ctx context.Context) (map[string]any, error) {
			return props, nil
		})
	})
	return config
}

// RegisterCentralClient adds call interceptors, adds central and admin user
// to the tenant group.
func RegisterCentralClient(config grpc.Config, basicAuthProviderID string) {
	cfg := InstanceConfig()
	if !cfg.Enabled() {
		return
	}
	registerInterceptors(config)
	// Central adds itself to the tenant group, with no group properties:
	cfg.Telemeter().With(cfg.ClientID).Group(cfg.GroupID, nil)
	registerAdminUser(basicAuthProviderID)
}

func registerInterceptors(config grpc.Config) {
	cfg := InstanceConfig()
	config.HTTPInterceptors = append(config.HTTPInterceptors, cfg.GetHTTPInterceptor())
	config.UnaryInterceptors = append(config.UnaryInterceptors, cfg.GetGRPCInterceptor())
}

// registerAdminUser adds the local admin user to the tenant group.
// This user is not added to the datastore like other users, so we need to add
// it to the tenant group specifically.
func registerAdminUser(basicAuthProviderID string) {
	cfg := InstanceConfig()

	// Add the basic authorization ID form ('admin'):
	adminHash := cfg.HashUserID(basic.DefaultUsername, basicAuthProviderID)
	cfg.Telemeter().With(adminHash).Group(cfg.GroupID, nil)

	// Add the token based ID form ('sso:<provider id>:admin'):
	adminTokenHash := cfg.HashUserID(
		tokenbased.FormatUserID(basic.DefaultUsername, basicAuthProviderID),
		basicAuthProviderID,
	)
	cfg.Telemeter().With(adminTokenHash).Group(cfg.GroupID, nil)
}
