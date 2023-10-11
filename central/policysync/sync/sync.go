package sync

import (
	"context"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	pDataStore "github.com/stackrox/rox/central/policy/datastore"
	psDataStore "github.com/stackrox/rox/central/policysync/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/policies"
	"github.com/stackrox/rox/pkg/registries/types"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/sync"
)

const (
	policySyncInterval = 1 * time.Minute
)

var (
	_ PolicySyncer = (*policySyncerImpl)(nil)

	once sync.Once

	ps PolicySyncer

	log = logging.LoggerForModule()
)

// PolicySyncer handles syncing policies from OCI registries.
type PolicySyncer interface {
	Start()
	Stop()
}

type policySyncerImpl struct {
	psDS       psDataStore.DataStore
	pDS        pDataStore.DataStore
	stopSignal concurrency.Signal
	fetcher    policies.Fetcher
	ctx        context.Context
}

// Singleton provides the singleton.
func Singleton() PolicySyncer {
	once.Do(func() {
		ps = newSyncer(psDataStore.Singleton(), pDataStore.Singleton())
	})
	return ps
}

func newSyncer(psDS psDataStore.DataStore, pDS pDataStore.DataStore) PolicySyncer {
	return &policySyncerImpl{
		psDS:       psDS,
		pDS:        pDS,
		stopSignal: concurrency.NewSignal(),
		fetcher:    policies.NewFetcher(),
		ctx: sac.WithGlobalAccessScopeChecker(context.Background(),
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeys(resources.WorkflowAdministration),
			),
		),
	}
}

func (p *policySyncerImpl) Start() {
	go p.runSync()
}

func (p *policySyncerImpl) Stop() {
	p.stopSignal.Signal()
}

func (p *policySyncerImpl) runSync() {
	ticker := time.NewTicker(policySyncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-p.stopSignal.Done():
			return
		case <-ticker.C:
			if err := p.fetchPolicies(); err != nil {
				log.Errorw("Failed to fetch policies", logging.Err(err))
			}
		}
	}
}

func (p *policySyncerImpl) fetchPolicies() error {
	syncCfg, exists, err := p.psDS.GetPolicySync(p.ctx)
	if err != nil {
		return errors.Wrap(err, "retrieving policy sync config")
	}
	if !exists {
		log.Info("No policy sync config exists, so nothing will be done.")
		return nil
	}

	var fetchedPolicies []*storage.Policy
	var fetchPolicyErrors *multierror.Error
	for _, registry := range syncCfg.GetRegistries() {
		registryConfig := &types.Config{
			RegistryHostname: registry.GetHostname(),
		}
		policies, err := p.fetcher.Fetch(p.ctx, registryConfig, registry.GetRepository())
		if err != nil {
			fetchPolicyErrors = multierror.Append(fetchPolicyErrors, err)
			continue
		}
		fetchedPolicies = append(fetchedPolicies, policies...)
	}

	if err := fetchPolicyErrors.ErrorOrNil(); err != nil {
		log.Warnw("An error occurred while fetching some policies, now policies might be incomplete",
			logging.Err(err))
	}
	if _, _, err := p.pDS.ImportPolicies(p.ctx, fetchedPolicies, true); err != nil {
		log.Warnw("An error occurred while importing some policies, now policies might be stale", logging.Err(err))
	}
	return nil
}
