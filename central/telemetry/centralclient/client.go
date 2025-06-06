package centralclient

import (
	"context"
	"encoding/json"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/globaldb"
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

	client        *centralClient
	onceSingleton sync.Once
	log           = logging.LoggerForModule()
)

type centralClient struct {
	*phonehome.Client

	campaignMux       sync.RWMutex
	telemetryCampaign phonehome.APICallCampaign
}

func noopClient(instanceId string) *centralClient {
	return &centralClient{Client: phonehome.NewClient(instanceId, "Central", version.GetMainVersion())}
}

func newCentralClient(instanceId string) *centralClient {
	if env.OfflineModeEnv.BooleanSetting() {
		return noopClient(instanceId)
	}

	// The internal client configuration is copied from cfg, so pointer access
	// doesn't modify the internal configuration.
	if instanceId == "" {
		if globaldb.GetPostgres() == nil {
			log.Warnf("No database. Telemetry disabled.")
			return noopClient(instanceId)
		}
		var err error
		instanceId, err = getInstanceId(installationDS.Singleton())
		if err != nil {
			log.Warnf("Failed to get central instance ID for telemetry configuration: %v.", err)
			return noopClient(instanceId)
		}
	}
	utils.Must(permanentTelemetryCampaign.Compile())

	c := &centralClient{}

	groupID := env.TenantID.Setting()
	// Consider a self-managed central a tenant of itself:
	if groupID == "" {
		groupID = instanceId
	}

	// Installation store might be not available when running unit tests, so
	// let's first check if the client is active and then update the instanceID.
	c.Client = phonehome.NewClient(instanceId, "Central", version.GetMainVersion(),
		phonehome.WithEndpoint(env.TelemetryEndpoint.Setting()),
		phonehome.WithStorageKey(env.TelemetryStorageKey.Setting()),
		phonehome.WithConfigURL(env.TelemetryConfigURL.Setting()),
		phonehome.WithGroup("Tenant", groupID),
		phonehome.WithAwaitInitialIdentity(),
		// If no key is provided via environment, the framework will eventually
		// download configuration with a key from the ConfigURL, and will
		// reconfigure the client. This applies only to release versions.
		phonehome.WithConfigureCallback(c.onReconfigure),
		// Segment does not respect the processing order of events in a
		// batch. Setting BatchSize to 1, instead of default 250, may reduce
		// the number of (none) values, appearing on Amplitude charts, by
		// introducing a slight delay between consequent events.
		phonehome.WithBatchSize(1),
		phonehome.WithPushInterval(env.TelemetryFrequency.DurationSetting()),
	)
	if !c.IsEnabled() {
		return c
	}
	c.AddInterceptorFuncs("API Call", c.apiCall(), addDefaultProps)

	if c.IsEnabled() {
		props := getCentralDeploymentProperties()
		c.Gatherer().AddGatherer(func(ctx context.Context) (map[string]any, error) {
			return props, nil
		})
	}

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
		"Chart version":      version.GetChartVersionOrEmpty(),
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
	onceSingleton.Do(func() {
		client = newCentralClient("")
		log.Infof("API Telemetry ignored paths: %v", ignoredPaths)
	})
	return client
}

// RegisterCentralClient adds call interceptors, adds central and admin user
// to the tenant group.
func (c *centralClient) RegisterCentralClient(gc *grpc.Config, basicAuthProviderID string) {
	gc.HTTPInterceptors = append(gc.HTTPInterceptors, c.GetHTTPInterceptor())
	gc.UnaryInterceptors = append(gc.UnaryInterceptors, c.GetGRPCInterceptor())

	groups := c.WithGroups()
	// Central adds itself to the tenant group, with no group properties:
	c.Group(groups...)

	// registerAdminUser adds the local admin user to the tenant group.
	// This user is not added to the datastore like other users, so we need to add
	// it to the tenant group specifically.
	adminHash := c.HashUserID(basic.DefaultUsername, basicAuthProviderID)
	c.Group(append(groups, telemeter.WithUserID(adminHash))...)
}

// Disable stops and disables the telemetry collection.
func (c *centralClient) Disable() {
	if c.Client.IsActive() {
		log.Info("Telemetry collection has been disabled on demand.")
		c.Track("Telemetry Disabled", nil)
		c.Gatherer().Stop()
	}
	c.Client.WithdrawConsent()
}

// Enable the client and start the telemetry collection.
func (c *centralClient) Enable() {
	if !c.IsEnabled() {
		return
	}
	c.Client.GrantConsent()

	c.Gatherer().Start(
		// Wrap WithNoDuplicates() with dynamic timestamp: don't capture the
		// time, but call time.Now() on every gathering iteration, so that
		// the message prefix is updated.
		func(co *telemeter.CallOptions) {
			// Issue a possible duplicate only once a day as a heartbeat.
			telemeter.WithNoDuplicates(time.Now().Format(time.DateOnly))(co)
		},
	)

	// This unblocks potentially waiting Track events.
	c.InitialIdentitySent()

	log.Info("Telemetry collection has been enabled.")
	go c.Track("Telemetry Enabled", nil)
}

func (c *centralClient) appendRuntimeCampaign(campaign phonehome.APICallCampaign) {
	c.campaignMux.Lock()
	defer c.campaignMux.Unlock()
	c.telemetryCampaign = append(permanentTelemetryCampaign, campaign...)
	jc, err := json.Marshal(c.telemetryCampaign)
	if err != nil {
		log.Warnw("Failed to marshal the API Telemetry campaign to JSON", logging.Err(err))
	} else {
		log.Info("API Telemetry campaign: ", string(jc))
	}
}

func (c *centralClient) onReconfigure(rc *phonehome.RuntimeConfig) {
	c.appendRuntimeCampaign(rc.APICallCampaign)
}
