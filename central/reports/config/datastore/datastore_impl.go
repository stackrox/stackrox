package datastore

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/reports/config/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	searchPkg "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
)

var (
	reportConfigSAC = sac.ForResource(resources.WorkflowAdministration)
)

type dataStoreImpl struct {
	reportConfigStore store.Store
}

func (d *dataStoreImpl) Search(ctx context.Context, q *v1.Query) ([]searchPkg.Result, error) {
	return d.reportConfigStore.Search(ctx, q)
}

func (d *dataStoreImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	return d.reportConfigStore.Count(ctx, q)
}

func (d *dataStoreImpl) GetReportConfigurations(ctx context.Context, query *v1.Query) ([]*storage.ReportConfiguration, error) {
	if ok, err := reportConfigSAC.ReadAllowed(ctx); !ok || err != nil {
		return nil, err
	}

	var configs []*storage.ReportConfiguration
	// Using WalkByQuery as risk could potentially return a large amount of data
	err := d.reportConfigStore.GetByQueryFn(ctx, query, func(config *storage.ReportConfiguration) error {
		configs = append(configs, config)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return configs, nil
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
	if reportConfig.GetId() == "" {
		reportConfig.Id = uuid.NewV4().String()
	}
	if err := d.reportConfigStore.Upsert(ctx, reportConfig); err != nil {
		return "", err
	}
	return reportConfig.GetId(), nil
}

func (d *dataStoreImpl) UpdateReportConfiguration(ctx context.Context, reportConfig *storage.ReportConfiguration) error {
	if err := sac.VerifyAuthzOK(reportConfigSAC.WriteAllowed(ctx)); err != nil {
		return err
	}

	if reportConfig.GetId() == "" {
		return errors.New("report configuration id field must be set")
	}

	return d.reportConfigStore.Upsert(ctx, reportConfig)
}

func (d *dataStoreImpl) RemoveReportConfiguration(ctx context.Context, id string) error {
	if err := sac.VerifyAuthzOK(reportConfigSAC.WriteAllowed(ctx)); err != nil {
		return err
	}
	if err := d.reportConfigStore.Delete(ctx, id); err != nil {
		return err
	}
	return nil
}

func (d *dataStoreImpl) Walk(ctx context.Context, fn func(reportConfig *storage.ReportConfiguration) error) error {
	if ok, err := reportConfigSAC.ReadAllowed(ctx); !ok || err != nil {
		return err
	}
	return d.reportConfigStore.Walk(ctx, fn, true)
}
