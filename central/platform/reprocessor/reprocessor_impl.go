package reprocessor

import (
	"context"
	"sync/atomic"

	alertDS "github.com/stackrox/rox/central/alert/datastore"
	configDS "github.com/stackrox/rox/central/config/datastore"
	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	platformmatcher "github.com/stackrox/rox/central/platform/matcher"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
)

const batchSize = 5000

var (
	log = logging.LoggerForModule()

	reprocessorCtx = sac.WithAllAccess(context.Background())
)

type platformReprocessorImpl struct {
	configDatastore     configDS.DataStore
	alertDatastore      alertDS.DataStore
	deploymentDatastore deploymentDS.DataStore
	platformMatcher     platformmatcher.PlatformMatcher

	// isStarted will make sure only one reprocessing routine runs for an instance of reprocessor
	isStarted atomic.Bool
}

func New(configDatastore configDS.DataStore,
	alertDatastore alertDS.DataStore,
	deploymentDatastore deploymentDS.DataStore,
	platformMatcher platformmatcher.PlatformMatcher) PlatformReprocessor {

	return &platformReprocessorImpl{
		configDatastore:     configDatastore,
		alertDatastore:      alertDatastore,
		deploymentDatastore: deploymentDatastore,
		platformMatcher:     platformMatcher,
	}
}

func (pr *platformReprocessorImpl) Start() {
	swapped := pr.isStarted.CompareAndSwap(false, true)
	if !swapped {
		log.Error("Platform reprocessor already running")
		return
	}
	go pr.runReprocessing()
}

func (pr *platformReprocessorImpl) Stop() {
	if !pr.isStarted.Load() {
		log.Error("Platform reprocessor not started")
		return
	}
}

func (pr *platformReprocessorImpl) runReprocessing() {
	conf, err := pr.configDatastore.GetInternalConfig(reprocessorCtx)
	if err != nil {
		log.Errorf("Error getting platform component config: %s", err)
		return
	}
	if conf.GetPlatformComponentConfig() == nil {
		log.Errorf("Platform component config is not set")
		return
	}

	if !conf.GetPlatformComponentConfig().GetNeedsReprocessing() {
		log.Info("Platform components up to date, skipping reprocessing")
		return
	}

	err = pr.reprocessAlerts()
	if err != nil {
		log.Errorf("Error reprocessing alerts with platform rules: %s", err)
		return
	}

	err = pr.reprocessDeployments()
	if err != nil {
		log.Errorf("Error reprocessing deployments with platform rules: %s", err)
		return
	}

	conf.GetPlatformComponentConfig().NeedsReprocessing = false
	err = pr.configDatastore.UpsertInternalConfig(reprocessorCtx, conf)
	if err != nil {
		log.Errorf("Error upserting platform config after reprocessing: %s", err)
	}
}

func (pr *platformReprocessorImpl) reprocessAlerts() error {
	query := search.EmptyQuery()
	query.Pagination = &v1.QueryPagination{
		Limit:  batchSize,
		Offset: 0,
		SortOptions: []*v1.QuerySortOption{
			{
				Field: search.AlertID.String(),
			},
		},
	}

	for {
		alerts, err := pr.alertDatastore.SearchRawAlerts(reprocessorCtx, query)
		if err != nil {
			return err
		}
		if len(alerts) == 0 {
			break
		}
		for _, alert := range alerts {
			match, err := pr.platformMatcher.MatchAlert(alert)
			if err != nil {
				return err
			}
			alert.PlatformComponent = match
		}
		err = pr.alertDatastore.UpsertAlerts(reprocessorCtx, alerts)
		if err != nil {
			return err
		}
		query.GetPagination().Offset += int32(len(alerts))
	}
	return nil
}

func (pr *platformReprocessorImpl) reprocessDeployments() error {
	query := search.EmptyQuery()
	query.Pagination = &v1.QueryPagination{
		Limit:  batchSize,
		Offset: 0,
		SortOptions: []*v1.QuerySortOption{
			{
				Field: search.DeploymentID.String(),
			},
		},
	}

	for {
		deps, err := pr.deploymentDatastore.SearchRawDeployments(reprocessorCtx, query)
		if err != nil {
			return err
		}
		if len(deps) == 0 {
			break
		}
		for _, dep := range deps {
			match, err := pr.platformMatcher.MatchDeployment(dep)
			if err != nil {
				return err
			}
			dep.PlatformComponent = match
		}
		err = pr.deploymentDatastore.UpsertDeployments(reprocessorCtx, deps)
		if err != nil {
			return err
		}
		query.GetPagination().Offset += int32(len(deps))
	}
	return nil
}
