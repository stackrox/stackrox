package reprocessor

import (
	"context"

	"github.com/pkg/errors"
	alertDS "github.com/stackrox/rox/central/alert/datastore"
	alertutils "github.com/stackrox/rox/central/alert/utils"
	configDS "github.com/stackrox/rox/central/config/datastore"
	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	platformmatcher "github.com/stackrox/rox/central/platform/matcher"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/backgroundworker"
	"github.com/stackrox/rox/pkg/features"
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
	configDatastore     configDS.DataStore
	deploymentDatastore deploymentDS.DataStore
	platformMatcher     platformmatcher.PlatformMatcher

	worker *backgroundworker.RunOnceWorker

	customized bool
}

func New(alertDatastore alertDS.DataStore,
	configDatastore configDS.DataStore,
	deploymentDatastore deploymentDS.DataStore,
	platformMatcher platformmatcher.PlatformMatcher) PlatformReprocessor {

	pr := &platformReprocessorImpl{
		alertDatastore:      alertDatastore,
		configDatastore:     configDatastore,
		deploymentDatastore: deploymentDatastore,
		platformMatcher:     platformMatcher,
		customized:          features.CustomizablePlatformComponents.Enabled(),
	}

	pr.worker = &backgroundworker.RunOnceWorker{
		Name: "platform-component-reprocessor",
		Run: func(ctx context.Context) error {
			pr.RunReprocessor(ctx)
			return nil
		},
		MaxAttempts: 1,
	}
	backgroundworker.Global.Register(pr.worker)

	return pr
}

func (pr *platformReprocessorImpl) Start() {
	pr.worker.Start(context.Background())
}

func (pr *platformReprocessorImpl) Stop() {
	pr.worker.Stop()
}

func (pr *platformReprocessorImpl) RunReprocessor(ctx context.Context) {
	flag := true
	if pr.customized {
		config, _, err := pr.configDatastore.GetPlatformComponentConfig(reprocessorCtx)
		if err != nil {
			log.Errorf("Error getting platform component config: %v", err)
		}
		flag = config.GetNeedsReevaluation()
	}
	if flag {
		err := pr.reprocessAlerts(ctx)
		if err != nil {
			log.Errorf("Error reprocessing alerts with platform rules: %v", err)
		}

		err = pr.reprocessDeployments(ctx)
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
}

func (pr *platformReprocessorImpl) reprocessAlerts(ctx context.Context) error {
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
		if ctx.Err() != nil {
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
		q.Pagination.Offset += batchSize
	}
	log.Info("Done reprocessing alerts with platform rules")
	return nil
}

func (pr *platformReprocessorImpl) reprocessDeployments(ctx context.Context) error {
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
		if ctx.Err() != nil {
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

		q.Pagination.Offset += batchSize
	}
	log.Info("Done reprocessing deployments with platform rules")
	return nil
}
