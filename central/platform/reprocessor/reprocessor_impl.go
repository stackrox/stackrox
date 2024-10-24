package reprocessor

import (
	"context"
	"sync/atomic"

	"github.com/pkg/errors"
	alertDS "github.com/stackrox/rox/central/alert/datastore"
	alertutils "github.com/stackrox/rox/central/alert/utils"
	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	platformmatcher "github.com/stackrox/rox/central/platform/matcher"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
)

const batchSize = 5000

var (
	log = logging.LoggerForModule()

	reprocessorCtx = sac.WithAllAccess(context.Background())

	unsetPlatformComponentQuery = search.NewQueryBuilder().AddNullField(search.PlatformComponent).ProtoQuery()
)

type platformReprocessorImpl struct {
	alertDatastore      alertDS.DataStore
	deploymentDatastore deploymentDS.DataStore
	platformMatcher     platformmatcher.PlatformMatcher

	stopSignal concurrency.Signal
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
		stopSignal:          concurrency.NewSignal(),
	}
}

func (pr *platformReprocessorImpl) Start() {
	swapped := pr.isStarted.CompareAndSwap(false, true)
	if !swapped {
		log.Error("Platform reprocessor was already started")
		return
	}
	go pr.runReprocessing()
}

func (pr *platformReprocessorImpl) Stop() {
	if !pr.isStarted.Load() {
		log.Error("Platform reprocessor not started")
	}
	pr.stopSignal.Signal()
}

func (pr *platformReprocessorImpl) runReprocessing() {
	err := pr.reprocessAlerts()
	if err != nil {
		log.Errorf("Error reprocessing alerts with platform rules: %v", err)
	}

	err = pr.reprocessDeployments()
	if err != nil {
		log.Errorf("Error reprocessing deployments with platform rules: %v", err)
	}
}

func (pr *platformReprocessorImpl) alertsNeedReprocessing() (bool, error) {
	// Check if there is atleast one alert where platform component is unset
	q := unsetPlatformComponentQuery.CloneVT()
	q.Pagination = &v1.QueryPagination{
		Limit: 1,
	}
	alerts, err := pr.alertDatastore.GetByQuery(reprocessorCtx, q)
	if err != nil {
		return false, err
	}
	return len(alerts) > 0, nil
}

func (pr *platformReprocessorImpl) deploymentsNeedReprocessing() (bool, error) {
	// Check if there is atleast one deployment where platform component is unset
	q := unsetPlatformComponentQuery.CloneVT()
	q.Pagination = &v1.QueryPagination{
		Limit: 1,
	}
	deployments, err := pr.deploymentDatastore.SearchRawDeployments(reprocessorCtx, q)
	if err != nil {
		return false, err
	}
	return len(deployments) > 0, nil
}

func (pr *platformReprocessorImpl) reprocessAlerts() error {
	if pr.stopSignal.IsDone() {
		log.Info("Stop called, stopping platform reprocessor")
		return nil
	}
	needReprocessing, err := pr.alertsNeedReprocessing()
	if err != nil {
		return errors.Wrap(err, "Error determining if alerts need reporcessing")
	}
	if !needReprocessing {
		log.Debug("Alerts up to date with platform rules, skipping reprocessing")
		return nil
	}

	q := unsetPlatformComponentQuery.CloneVT()
	q.Pagination = &v1.QueryPagination{
		Limit: batchSize,
	}

	for {
		if pr.stopSignal.IsDone() {
			log.Info("Stop called, stopping platform reprocessor")
			break
		}
		alerts, err := pr.alertDatastore.GetByQuery(reprocessorCtx, q)
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
	}
	log.Info("Done reprocessing alerts with platform rules")
	return nil
}

func (pr *platformReprocessorImpl) reprocessDeployments() error {
	if pr.stopSignal.IsDone() {
		log.Info("Stop called, stopping platform reprocessor")
		return nil
	}
	needReprocessing, err := pr.deploymentsNeedReprocessing()
	if err != nil {
		return errors.Wrap(err, "Error determining if deployments need reporcessing")
	}
	if !needReprocessing {
		log.Debug("Deployments up to date with platform rules, skipping reprocessing")
		return nil
	}

	q := unsetPlatformComponentQuery.CloneVT()
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
