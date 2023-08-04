package manager

import (
	"context"
	"errors"

	"github.com/stackrox/rox/central/graphql/resolvers/loaders"
	"github.com/stackrox/rox/central/reports/scheduler"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/contextutil"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
)

var (
	reportsSAC = sac.ForResource(resources.WorkflowAdministration)
)

type managerImpl struct {
	scheduler  scheduler.Scheduler
	inProgress concurrency.Flag
}

func (m *managerImpl) Remove(ctx context.Context, id string) error {
	if env.VulnReportingEnhancements.BooleanSetting() {
		return nil
	}
	if ok, err := reportsSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}
	m.scheduler.RemoveReportSchedule(id)
	return nil
}

func (m *managerImpl) Upsert(ctx context.Context, reportConfig *storage.ReportConfiguration) error {
	if env.VulnReportingEnhancements.BooleanSetting() {
		return nil
	}
	if ok, err := reportsSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}
	if err := m.scheduler.UpsertReportSchedule(reportConfig); err != nil {
		return err
	}
	return nil
}

func (m *managerImpl) RunReport(ctx context.Context, reportConfig *storage.ReportConfiguration) error {
	if env.VulnReportingEnhancements.BooleanSetting() {
		return nil
	}
	/*
	 * Multiple on demand reports cannot be executed concurrently.
	 * An on demand report may be submitted while other scheduled reports are being run and will be
	 * executed in FIFO order (in case multiple scheduled reports are already queued up).
	 */
	if m.inProgress.TestAndSet(true) {
		return errors.New("report generation already in progress, please try again later")
	}
	defer m.inProgress.Set(false)

	m.scheduler.SubmitReport(&scheduler.ReportRequest{
		ReportConfig: reportConfig,
		OnDemand:     true,
		Ctx:          loaders.WithLoaderContext(contextutil.WithValuesFrom(context.Background(), ctx)),
	})
	return nil
}

func (m *managerImpl) Start() {
	if env.VulnReportingEnhancements.BooleanSetting() {
		return
	}
	m.scheduler.Start()
}

func (m *managerImpl) Stop() {
	if env.VulnReportingEnhancements.BooleanSetting() {
		return
	}
	m.scheduler.Stop()
}
