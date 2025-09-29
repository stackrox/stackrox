package custom

import (
	"context"
	"net/http"

	alertDS "github.com/stackrox/rox/central/alert/datastore"
	clusterDS "github.com/stackrox/rox/central/cluster/datastore"
	configDS "github.com/stackrox/rox/central/config/datastore"
	expiryS "github.com/stackrox/rox/central/credentialexpiry/service"
	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/metrics/custom/clusters"
	"github.com/stackrox/rox/central/metrics/custom/expiry"
	"github.com/stackrox/rox/central/metrics/custom/image_vulnerabilities"
	"github.com/stackrox/rox/central/metrics/custom/node_vulnerabilities"
	"github.com/stackrox/rox/central/metrics/custom/policies"
	"github.com/stackrox/rox/central/metrics/custom/policy_violations"
	"github.com/stackrox/rox/central/metrics/custom/tracker"
	nodeDS "github.com/stackrox/rox/central/node/datastore"
	policyDS "github.com/stackrox/rox/central/policy/datastore"
	"github.com/stackrox/rox/central/telemetry/centralclient"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/telemetry/phonehome/telemeter"
)

type trackerRunner []struct {
	tracker.Tracker
	// getGroupConfig returns the storage configuration associated to the
	// tracker.
	getGroupConfig func(*storage.PrometheusMetrics) *storage.PrometheusMetrics_Group
}

// RunnerConfiguration is a composition of tracker configurations.
// Returned by ValidateConfiguration() and accepted by Reconfigure(). This split
// allows the config service to dry-validate the configuration before applying
// any changes.
type RunnerConfiguration []*tracker.Configuration

type runnerDatastores struct {
	deployments deploymentDS.DataStore
	alerts      alertDS.DataStore
	nodes       nodeDS.DataStore
	clusters    clusterDS.DataStore
	policies    policyDS.DataStore
	expiry      expiryS.Service
}

func withHardcodedConfiguration(period uint32, descriptors map[string][]string) func(*storage.PrometheusMetrics) *storage.PrometheusMetrics_Group {
	group := &storage.PrometheusMetrics_Group{
		GatheringPeriodMinutes: period,
		Descriptors:            map[string]*storage.PrometheusMetrics_Group_Labels{},
	}

	for metric, labels := range descriptors {
		group.Descriptors[metric] = &storage.PrometheusMetrics_Group_Labels{
			Labels: labels,
		}
	}

	return func(*storage.PrometheusMetrics) *storage.PrometheusMetrics_Group {
		return group
	}
}

func makeRunner(ds *runnerDatastores) trackerRunner {
	return trackerRunner{{
		image_vulnerabilities.New(ds.deployments),
		(*storage.PrometheusMetrics).GetImageVulnerabilities,
	}, {
		policy_violations.New(ds.alerts),
		(*storage.PrometheusMetrics).GetPolicyViolations,
	}, {
		node_vulnerabilities.New(ds.nodes),
		(*storage.PrometheusMetrics).GetNodeVulnerabilities,
	}, {
		clusters.New(ds.clusters),
		withHardcodedConfiguration(60, map[string][]string{
			// rox_central_health_cluster_info
			"cluster_info": tracker.GetLabels(clusters.LazyLabels),
		}),
	}, {
		policies.New(ds.policies),
		withHardcodedConfiguration(60, map[string][]string{
			// rox_central_cfg_total_policies
			"total_policies": tracker.GetLabels(policies.LazyLabels),
		}),
	}, {
		expiry.New(ds.expiry),
		withHardcodedConfiguration(60, map[string][]string{
			// rox_central_cert_exp_hours
			"hours": tracker.GetLabels(expiry.LazyLabels),
		}),
	},
	}
}

func (tr trackerRunner) initialize(cds configDS.DataStore) {
	ctx := sac.WithAllAccess(context.Background())
	systemPrivateConfig, err := cds.GetPrivateConfig(ctx)
	if err != nil {
		log.Errorw("Failed to read Prometheus metrics configuration from the DB", logging.Err(err))
		return
	}

	cfg, err := tr.ValidateConfiguration(systemPrivateConfig.GetMetrics())
	if err != nil {
		log.Errorw("Failed to configure Prometheus metrics", logging.Err(err))
		return
	}
	tr.Reconfigure(cfg)
}

func (tr trackerRunner) ValidateConfiguration(cfg *storage.PrometheusMetrics) (RunnerConfiguration, error) {
	if tr == nil {
		return RunnerConfiguration{}, nil
	}
	var runnerConfig RunnerConfiguration
	for _, tracker := range tr {
		trackerConfig, err := tracker.NewConfiguration(tracker.getGroupConfig(cfg))
		if err != nil {
			return nil, err
		}
		runnerConfig = append(runnerConfig, trackerConfig)
	}
	return runnerConfig, nil
}

// Reconfigure applies the provided configuration.
// Non-nil runner will panic on nil cfg. Don't pass nil.
func (tr trackerRunner) Reconfigure(cfg RunnerConfiguration) {
	if tr == nil {
		return
	}
	if cfg == nil {
		log.Panic("programmer error: nil configuration passed")
	} else if len(cfg) != len(tr) {
		log.Error("invalid metrics configuration")
	} else {
		for i, tracker := range tr {
			tracker.Reconfigure(cfg[i])
		}
	}
}

func (tr trackerRunner) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if tr == nil {
		return
	}
	var userID string
	reqCtx := req.Context()
	if id := authn.IdentityFromContextOrNil(reqCtx); id != nil {
		userID = id.UID()
		// The request context is cancelled when the client's connection closes.
		newCtx := authn.CopyContextIdentity(context.Background(), reqCtx)
		newCtx = sac.CopyAccessScopeCheckerCore(newCtx, reqCtx)
		for _, tracker := range tr {
			go tracker.Gather(newCtx)
		}
	}
	registry, err := metrics.GetCustomRegistry(userID)
	if err != nil {
		httputil.WriteError(w, err)
		return
	}
	registry.Lock()
	defer registry.Unlock()
	registry.ServeHTTP(w, req)
	go phonehome()
}

func phonehome() {
	props := map[string]any{
		"Total custom Prometheus registries": metrics.GetCustomRegistriesCount(),
	}

	centralclient.Singleton().Telemeter().Track(
		"Served custom Prometheus metrics", nil,
		telemeter.WithTraits(props),
		telemeter.WithNoDuplicates("prom_registries"))
}
