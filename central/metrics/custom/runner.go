package custom

import (
	"context"
	"net/http"

	alertDS "github.com/stackrox/rox/central/alert/datastore"
	configDS "github.com/stackrox/rox/central/config/datastore"
	cveDS "github.com/stackrox/rox/central/cve/node/datastore"
	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/metrics/custom/image_vulnerabilities"
	"github.com/stackrox/rox/central/metrics/custom/node_vulnerabilities"
	"github.com/stackrox/rox/central/metrics/custom/policy_violations"
	custom "github.com/stackrox/rox/central/metrics/custom/tracker"
	nodeDS "github.com/stackrox/rox/central/node/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
)

type aggregatorRunner []struct {
	custom.Tracker
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
	nodes       nodeDS.DataStore
	cves        cveDS.DataStore
}

func makeRunner(registryFactory func(string) metrics.CustomRegistry, ds *runnerDatastores) aggregatorRunner {
	return aggregatorRunner{{
		image_vulnerabilities.New(registryFactory, ds.deployments),
		(*storage.PrometheusMetrics).GetImageVulnerabilities,
	}, {
		policy_violations.New(registryFactory, ds.alerts),
		(*storage.PrometheusMetrics).GetPolicyViolations,
	}, {
		node_vulnerabilities.New(registryFactory, ds.nodes, ds.cves),
		(*storage.PrometheusMetrics).GetNodeVulnerabilities,
	},
	}
}

func (ar aggregatorRunner) initialize(cds configDS.DataStore) {
	ctx := sac.WithAllAccess(context.Background())
	systemPrivateConfig, err := cds.GetPrivateConfig(ctx)
	if err != nil {
		log.Errorw("Failed to read Prometheus metrics configuration from the DB", logging.Err(err))
		return
	}

	cfg, err := ar.ValidateConfiguration(systemPrivateConfig.GetMetrics())
	if err != nil {
		log.Errorw("Failed to configure Prometheus metrics", logging.Err(err))
		return
	}
	ar.Reconfigure(cfg)
}

func (ar aggregatorRunner) ValidateConfiguration(cfg *storage.PrometheusMetrics) (RunnerConfiguration, error) {
	if ar == nil {
		return RunnerConfiguration{}, nil
	}
	var runnerConfig RunnerConfiguration
	for _, c := range ar {
		rcfg, err := c.NewConfiguration(c.getGroupConfig(cfg))
		if err != nil {
			return nil, err
		}
		runnerConfig = append(runnerConfig, rcfg)
	}
	return runnerConfig, nil
}

// Reconfigure applies the provided configuration.
// Non-nil runner will panic on nil cfg. Don't pass nil.
func (ar aggregatorRunner) Reconfigure(cfg RunnerConfiguration) {
	if ar == nil {
		return
	}
	if cfg == nil {
		log.Panic("programmer error: nil configuration passed")
	} else if len(cfg) != len(ar) {
		log.Error("invalid metrics configuration")
	} else {
		for i, r := range ar {
			r.Reconfigure(cfg[i])
		}
	}

}

func (ar aggregatorRunner) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if ar == nil {
		return
	}
	var userID string
	if id := authn.IdentityFromContextOrNil(req.Context()); id != nil {
		userID = id.UID()
		// The request context is cancelled when the client's connection closes.
		ctx := authn.CopyContextIdentity(context.Background(), req.Context())
		for _, r := range ar {
			go r.Gather(ctx)
		}
	}
	registry := metrics.GetCustomRegistry(userID)
	registry.Lock()
	defer registry.Unlock()
	registry.ServeHTTP(w, req)
}
