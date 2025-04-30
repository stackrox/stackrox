package reprocessor

import (
	"context"
	"sync/atomic"

	"github.com/pkg/errors"
	alertDS "github.com/stackrox/rox/central/alert/datastore"
	alertutils "github.com/stackrox/rox/central/alert/utils"
	configDS "github.com/stackrox/rox/central/config/datastore"
	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	platformmatcher "github.com/stackrox/rox/central/platform/matcher"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"golang.org/x/sync/semaphore"
)

const batchSize = 5000

var (
	log = logging.LoggerForModule()

	reprocessorCtx = sac.WithAllAccess(context.Background())

	unsetPlatformComponentQuery = search.NewQueryBuilder().AddNullField(search.PlatformComponent).ProtoQuery()
)

type platformReprocessorImpl struct {
	alertDatastore      alertDS.DataStore
	configDatastore     configDS.DataStore
	deploymentDatastore deploymentDS.DataStore
	platformMatcher     platformmatcher.PlatformMatcher

	semaphore  *semaphore.Weighted
	stopSignal concurrency.Signal
	// isStarted will make sure only one reprocessing routine runs for an instance of reprocessor
	isStarted atomic.Bool

	customized bool
}

func New(alertDatastore alertDS.DataStore,
	configDatastore configDS.DataStore,
	deploymentDatastore deploymentDS.DataStore,
	platformMatcher platformmatcher.PlatformMatcher) PlatformReprocessor {

	return &platformReprocessorImpl{
		alertDatastore:      alertDatastore,
		configDatastore:     configDatastore,
		deploymentDatastore: deploymentDatastore,
		platformMatcher:     platformMatcher,
		semaphore:           semaphore.NewWeighted(1),
		stopSignal:          concurrency.NewSignal(),
		customized:          features.CustomizablePlatformComponents.Enabled(),
	}
}

func (pr *platformReprocessorImpl) Start() {
	swapped := pr.isStarted.CompareAndSwap(false, true)
	if !swapped {
		log.Error("Platform reprocessor was already started")
		return
	}
	go pr.RunReprocessor()
}

func (pr *platformReprocessorImpl) Stop() {
	if !pr.isStarted.Load() {
		log.Error("Platform reprocessor not started")
	}
	pr.stopSignal.Signal()
}

func (pr *platformReprocessorImpl) RunReprocessor() {
	err := pr.semaphore.Acquire(reprocessorCtx, 1)
	if err != nil {
		log.Errorf("Failed to acquire reprocessor semaphore: %v", err)
		return
	}
	flag := true
	if pr.customized {
		config, _, err := pr.configDatastore.GetPlatformComponentConfig(reprocessorCtx)
		if err != nil {
			log.Errorf("Error getting platform component config config: %v", err)
		}
		flag = config.NeedsReevaluation
	}
	log.Infof("Reprocessor started, flag: %v", flag)
	if flag {
		err := pr.reprocessAlerts()
		if err != nil {
			log.Errorf("Error reprocessing alerts with platform rules: %v", err)
		}

		err = pr.reprocessDeployments()
		if err != nil {
			log.Errorf("Error reprocessing deployments with platform rules: %v", err)
		}
		if pr.customized {
			err = pr.configDatastore.MarkPCCReevaluated(reprocessorCtx)
			if err != nil {
				log.Errorf("Error marking platform component config as reevaluated: %v", err)
			}
		}
	}
	pr.semaphore.Release(1)
}

func (pr *platformReprocessorImpl) reprocessAlerts() error {
	var q *v1.Query
	if pr.customized {
		q = search.EmptyQuery()
	} else {
		q = unsetPlatformComponentQuery
	}
	q.Pagination = &v1.QueryPagination{
		Limit: batchSize,
	}

	var alerts []*storage.Alert
	for {
		if pr.stopSignal.IsDone() {
			log.Info("Stop called, stopping platform reprocessor")
			break
		}

		err := pr.alertDatastore.WalkByQuery(reprocessorCtx, q, func(alert *storage.Alert) error {
			alert.EntityType = alertutils.GetEntityType(alert)
			match, err := pr.platformMatcher.MatchAlert(alert)
			if err != nil {
				return errors.Wrap(err, "matching alert")
			}
			alert.PlatformComponent = match
			alerts = append(alerts, alert)
			return nil
		})
		if err != nil {
			return err
		}
		if len(alerts) == 0 {
			break
		}
		err = pr.alertDatastore.UpsertAlerts(reprocessorCtx, alerts)
		if err != nil {
			return err
		}
		alerts = alerts[:0]
	}
	log.Info("Done reprocessing alerts with platform rules")
	return nil
}

func (pr *platformReprocessorImpl) reprocessDeployments() error {
	var q *v1.Query
	if pr.customized {
		q = search.EmptyQuery()
	} else {
		q = unsetPlatformComponentQuery
	}
	q.Pagination = &v1.QueryPagination{
		Limit: batchSize,
	}

	for {
		if pr.stopSignal.IsDone() {
			log.Info("Stop called, stopping platform reprocessor")
			break
		}
		deps, err := pr.deploymentDatastore.SearchRawDeployments(reprocessorCtx, q)
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
	}
	log.Info("Done reprocessing deployments with platform rules")
	return nil
}
