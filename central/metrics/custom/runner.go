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

type aggregatorRunner struct {
	registry              metrics.CustomRegistry
	image_vulnerabilities custom.Tracker
	policy_violations     custom.Tracker
}

// RunnerConfiguration is a composition of tracker configurations.
// Returned by ValidateConfiguration() and accepted by Reconfigure(). This split
// allows the config service to dry-validate the configuration before applying
// any changes.
type RunnerConfiguration struct {
	image_vulnerabilities *custom.Configuration
	policy_violations     *custom.Configuration
}

func makeRunner(registry metrics.CustomRegistry, dds deploymentDS.DataStore, ads alertDS.DataStore) *aggregatorRunner {
	return &aggregatorRunner{
		registry:              registry,
		image_vulnerabilities: image_vulnerabilities.New(registry, dds),
		policy_violations:     policy_violations.New(registry, ads),
	}
}

func (ar *aggregatorRunner) initialize(cds configDS.DataStore) {
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

func (ar *aggregatorRunner) ValidateConfiguration(cfg *storage.PrometheusMetrics) (*RunnerConfiguration, error) {
	if ar == nil {
		return &RunnerConfiguration{}, nil
	}
	var err error
	runnerConfig := &RunnerConfiguration{}
	runnerConfig.image_vulnerabilities, err = ar.image_vulnerabilities.NewConfiguration(cfg.GetImageVulnerabilities())
	if err != nil {
		return nil, err
	}
	runnerConfig.policy_violations, err = ar.policy_violations.NewConfiguration(cfg.GetPolicyViolations())
	if err != nil {
		return nil, err
	}
	return runnerConfig, nil
}

// Reconfigure applies the provided configuration.
// Non-nil runner will panic on nil cfg. Don't pass nil.
func (ar *aggregatorRunner) Reconfigure(cfg *RunnerConfiguration) {
	if ar == nil {
		return
	}
	if cfg == nil {
		log.Panic("programmer error: nil configuration passed")
	} else {
		ar.image_vulnerabilities.Reconfigure(cfg.image_vulnerabilities)
		ar.policy_violations.Reconfigure(cfg.policy_violations)
	}
}

func (ar *aggregatorRunner) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if ar == nil {
		return
	}
	if id := authn.IdentityFromContextOrNil(req.Context()); id != nil {
		// The request context is cancelled when the client's connection closes.
		ctx := authn.CopyContextIdentity(context.Background(), req.Context())
		go ar.image_vulnerabilities.Gather(ctx)
		go ar.policy_violations.Gather(ctx)
	}
	ar.registry.Lock()
	defer ar.registry.Unlock()
	ar.registry.ServeHTTP(w, req)
}
