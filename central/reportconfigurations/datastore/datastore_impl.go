package datastore

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/central/reportconfigurations/index"
	"github.com/stackrox/stackrox/central/reportconfigurations/search"
	"github.com/stackrox/stackrox/central/reportconfigurations/store"
	"github.com/stackrox/stackrox/central/role/resources"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/debug"
	"github.com/stackrox/stackrox/pkg/features"
	"github.com/stackrox/stackrox/pkg/logging"
	"github.com/stackrox/stackrox/pkg/sac"
	searchPkg "github.com/stackrox/stackrox/pkg/search"
	"github.com/stackrox/stackrox/pkg/uuid"
)

var (
	reportConfigSAC = sac.ForResource(resources.VulnerabilityReports)

	log = logging.LoggerForModule()
)

type dataStoreImpl struct {
	reportConfigStore store.Store

	searcher search.Searcher
	indexer  index.Indexer
}

func (d *dataStoreImpl) buildIndex(ctx context.Context) error {
	if features.PostgresDatastore.Enabled() {
		return nil
	}
	defer debug.FreeOSMemory()
	log.Info("[STARTUP] Indexing report configurations")

	var reportConfigs []*storage.ReportConfiguration
	err := d.reportConfigStore.Walk(ctx, func(reportConfig *storage.ReportConfiguration) error {
		reportConfigs = append(reportConfigs, reportConfig)
		return nil
	})
	if err != nil {
		return err
	}
	if err := d.indexer.AddReportConfigurations(reportConfigs); err != nil {
		return err
	}
	log.Info("[STARTUP] Successfully indexed report configurations")
	return nil
}

func (d *dataStoreImpl) Search(ctx context.Context, q *v1.Query) ([]searchPkg.Result, error) {
	return d.searcher.Search(ctx, q)
}

func (d *dataStoreImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	return d.searcher.Count(ctx, q)
}

func (d *dataStoreImpl) GetReportConfigurations(ctx context.Context, query *v1.Query) ([]*storage.ReportConfiguration, error) {
	if ok, err := reportConfigSAC.ReadAllowed(ctx); !ok || err != nil {
		return nil, err
	}
	return d.searcher.SearchReportConfigurations(ctx, query)
}

func (d *dataStoreImpl) GetReportConfiguration(ctx context.Context, id string) (*storage.ReportConfiguration, bool, error) {
	if ok, err := reportConfigSAC.ReadAllowed(ctx); !ok || err != nil {
		return nil, false, err
	}
	return d.reportConfigStore.Get(ctx, id)
}

func (d *dataStoreImpl) AddReportConfiguration(ctx context.Context, reportConfig *storage.ReportConfiguration) (string, error) {
	if err := sac.VerifyAuthzOK(reportConfigSAC.WriteAllowed(ctx)); err != nil {
		return "", err
	}
	if reportConfig.Id == "" {
		reportConfig.Id = uuid.NewV4().String()
	}
	if err := d.reportConfigStore.Upsert(ctx, reportConfig); err != nil {
		return "", err
	}
	if err := d.indexer.AddReportConfiguration(reportConfig); err != nil {
		return reportConfig.Id, err
	}
	return reportConfig.Id, nil
}

func (d *dataStoreImpl) UpdateReportConfiguration(ctx context.Context, reportConfig *storage.ReportConfiguration) error {
	if err := sac.VerifyAuthzOK(reportConfigSAC.WriteAllowed(ctx)); err != nil {
		return err
	}

	if reportConfig.GetId() == "" {
		return errors.New("report configuration id field must be set")
	}

	if err := d.reportConfigStore.Upsert(ctx, reportConfig); err != nil {
		return err
	}
	return d.indexer.AddReportConfiguration(reportConfig)
}

func (d *dataStoreImpl) RemoveReportConfiguration(ctx context.Context, id string) error {
	if err := sac.VerifyAuthzOK(reportConfigSAC.WriteAllowed(ctx)); err != nil {
		return err
	}
	if err := d.reportConfigStore.Delete(ctx, id); err != nil {
		return err
	}
	return d.indexer.DeleteReportConfiguration(id)
}
