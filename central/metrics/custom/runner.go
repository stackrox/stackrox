package custom

import (
	"context"
	"net/http"

	alertDS "github.com/stackrox/rox/central/alert/datastore"
	configDS "github.com/stackrox/rox/central/config/datastore"
	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/metrics/custom/image_vulnerabilities"
	"github.com/stackrox/rox/central/metrics/custom/policy_violations"
	custom "github.com/stackrox/rox/central/metrics/custom/tracker"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
)

type trackerRunner []struct {
	custom.Tracker
	// getGroupConfig returns the storage configuration associated to the
	// tracker.
	getGroupConfig func(*storage.PrometheusMetrics) *storage.PrometheusMetrics_Group
}

// RunnerConfiguration is a composition of tracker configurations.
// Returned by ValidateConfiguration() and accepted by Reconfigure(). This split
// allows the config service to dry-validate the configuration before applying
// any changes.
type RunnerConfiguration []*custom.Configuration

type runnerDatastores struct {
	deployments deploymentDS.DataStore
	alerts      alertDS.DataStore
}

func makeRunner(ds *runnerDatastores) trackerRunner {
	return trackerRunner{{
		image_vulnerabilities.New(ds.deployments),
		(*storage.PrometheusMetrics).GetImageVulnerabilities,
	}, {
		policy_violations.New(ds.alerts),
		(*storage.PrometheusMetrics).GetPolicyViolations,
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
	if id := authn.IdentityFromContextOrNil(req.Context()); id != nil {
		userID = id.UID()
		// The request context is cancelled when the client's connection closes.
		ctx := authn.CopyContextIdentity(context.Background(), req.Context())
		for _, tracker := range tr {
			go tracker.Gather(ctx)
		}
	}
	registry := metrics.GetCustomRegistry(userID)
	registry.Lock()
	defer registry.Unlock()
	registry.ServeHTTP(w, req)
}
