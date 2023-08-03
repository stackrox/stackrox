package service

import (
	"context"

	notifierDataStore "github.com/stackrox/rox/central/notifier/datastore"
	"github.com/stackrox/rox/central/reportconfigurations/datastore"
	"github.com/stackrox/rox/central/reports/manager"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	as Service
)

func initialize() {
	mgr := initializeManager()
	as = New(datastore.Singleton(), notifierDataStore.Singleton(), mgr)
}

func initializeManager() manager.Manager {
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.WorkflowAdministration)))

	query := search.NewQueryBuilder().AddExactMatches(search.ReportType, storage.ReportConfiguration_VULNERABILITY.String()).ProtoQuery()
	reportConfigs, err := datastore.Singleton().GetReportConfigurations(ctx, query)
	mgr := manager.Singleton()
	if err != nil {
		log.Errorf("Error finding scheduled reports: %s", err)
		return mgr
	}

	for _, rc := range reportConfigs {
		if err := mgr.Upsert(ctx, rc); err != nil {
			log.Errorf("error upserting report config: %v", err)
		}
	}
	mgr.Start()
	return mgr
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
