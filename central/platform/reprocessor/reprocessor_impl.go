package reprocessor

import (
	"context"
	"sync/atomic"

	alertDS "github.com/stackrox/rox/central/alert/datastore"
	alertutils "github.com/stackrox/rox/central/alert/utils"
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
	alertDatastore      alertDS.DataStore
	deploymentDatastore deploymentDS.DataStore
	platformMatcher     platformmatcher.PlatformMatcher

	// isStarted will make sure only one reprocessing routine runs for an instance of reprocessor
	isStarted atomic.Bool
}

func New(alertDatastore alertDS.DataStore,
	deploymentDatastore deploymentDS.DataStore,
	platformMatcher platformmatcher.PlatformMatcher) PlatformReprocessor {

	return &platformReprocessorImpl{
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
	needsReprocessing, err := pr.needsReprocessing()
	if err != nil {
		log.Errorf("Error determining if platform components need reporcessing: %v", err)
		return
	}

	if !needsReprocessing {
		log.Info("Platform components up to date, skipping reprocessing")
		return
	}

	err = pr.reprocessAlerts()
	if err != nil {
		log.Errorf("Error reprocessing alerts with platform rules: %v", err)
		return
	}

	err = pr.reprocessDeployments()
	if err != nil {
		log.Errorf("Error reprocessing deployments with platform rules: %v", err)
		return
	}
}

func (pr *platformReprocessorImpl) needsReprocessing() (bool, error) {
	query := search.NewQueryBuilder().AddNullField(search.PlatformComponent).ProtoQuery()
	numAlerts, err := pr.alertDatastore.Count(reprocessorCtx, query)
	if err != nil {
		return false, err
	}
	numDeployments, err := pr.deploymentDatastore.Count(reprocessorCtx, query)
	if err != nil {
		return false, err
	}
	return numAlerts > 0 || numDeployments > 0, nil
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
			alert.EntityType = alertutils.GetEntityType(alert)
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
			err = pr.deploymentDatastore.UpsertDeployment(reprocessorCtx, dep)
			if err != nil {
				return err
			}
		}
		query.GetPagination().Offset += int32(len(deps))
	}
	return nil
}
