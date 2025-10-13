package service

import (
	"context"
	"testing"

	notifierMocks "github.com/stackrox/rox/central/notifier/datastore/mocks"
	"github.com/stackrox/rox/central/reports/config/datastore/mocks"
	managerMocks "github.com/stackrox/rox/central/reports/manager/mocks"
	collectionMocks "github.com/stackrox/rox/central/resourcecollection/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestReportConfigurationServicePostgres(t *testing.T) {
	suite.Run(t, new(ReportConfigurationServicePostgresTestSuite))
}

type ReportConfigurationServicePostgresTestSuite struct {
	suite.Suite
	service               Service
	reportConfigDatastore *mocks.MockDataStore
	notifierDatastore     *notifierMocks.MockDataStore
	collectionDatastore   *collectionMocks.MockDataStore
	manager               *managerMocks.MockManager
	mockCtrl              *gomock.Controller
}

func (s *ReportConfigurationServicePostgresTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.reportConfigDatastore = mocks.NewMockDataStore(s.mockCtrl)
	s.notifierDatastore = notifierMocks.NewMockDataStore(s.mockCtrl)
	s.collectionDatastore = collectionMocks.NewMockDataStore(s.mockCtrl)
	s.manager = managerMocks.NewMockManager(s.mockCtrl)
	s.service = New(s.reportConfigDatastore, s.notifierDatastore, s.collectionDatastore, s.manager)
}

func (s *ReportConfigurationServicePostgresTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *ReportConfigurationServicePostgresTestSuite) TestAddValidReportConfiguration() {
	ctx := context.Background()

	reportConfig := fixtures.GetValidReportConfiguration()
	s.reportConfigDatastore.EXPECT().AddReportConfiguration(ctx, reportConfig).Return(reportConfig.GetId(), nil)
	s.reportConfigDatastore.EXPECT().GetReportConfiguration(ctx, reportConfig.GetId()).Return(reportConfig, true, nil)

	s.notifierDatastore.EXPECT().Exists(ctx, gomock.Any()).Return(true, nil).AnyTimes()
	s.collectionDatastore.EXPECT().Exists(ctx, gomock.Any()).Return(true, nil).AnyTimes()

	s.manager.EXPECT().Upsert(ctx, reportConfig).Return(nil)
	_, err := s.service.PostReportConfiguration(ctx, &v1.PostReportConfigurationRequest{
		ReportConfig: reportConfig,
	})
	s.NoError(err)
}

func (s *ReportConfigurationServicePostgresTestSuite) TestAddInvalidValidReportConfigurations() {
	ctx := context.Background()

	s.notifierDatastore.EXPECT().Exists(ctx, gomock.Any()).Return(true, nil).AnyTimes()
	s.collectionDatastore.EXPECT().Exists(ctx, gomock.Any()).Return(true, nil).AnyTimes()

	noNotifierReportConfig := fixtures.GetInvalidReportConfigurationNoNotifier()
	_, err := s.service.PostReportConfiguration(ctx, &v1.PostReportConfigurationRequest{
		ReportConfig: noNotifierReportConfig,
	})
	s.Error(err)

	incorrectScheduleReportConfig := fixtures.GetInvalidReportConfigurationIncorrectSchedule()
	_, err = s.service.PostReportConfiguration(ctx, &v1.PostReportConfigurationRequest{
		ReportConfig: incorrectScheduleReportConfig,
	})
	s.Error(err)

	missingScheduleReportConfig := fixtures.GetInvalidReportConfigurationMissingSchedule()
	_, err = s.service.PostReportConfiguration(ctx, &v1.PostReportConfigurationRequest{
		ReportConfig: missingScheduleReportConfig,
	})
	s.Error(err)

	missingDaysOfWeekReportConfig := fixtures.GetInvalidReportConfigurationMissingDaysOfWeek()
	_, err = s.service.PostReportConfiguration(ctx, &v1.PostReportConfigurationRequest{
		ReportConfig: missingDaysOfWeekReportConfig,
	})
	s.Error(err)

	missingDaysOfMonthReportConfig := fixtures.GetInvalidReportConfigurationMissingDaysOfMonth()
	_, err = s.service.PostReportConfiguration(ctx, &v1.PostReportConfigurationRequest{
		ReportConfig: missingDaysOfMonthReportConfig,
	})
	s.Error(err)

	dailyScheduleReportConfig := fixtures.GetInvalidReportConfigurationDailySchedule()
	_, err = s.service.PostReportConfiguration(ctx, &v1.PostReportConfigurationRequest{
		ReportConfig: dailyScheduleReportConfig,
	})
	s.Error(err)

	incorrectEmailReportConfig := fixtures.GetInvalidReportConfigurationIncorrectEmailV1()
	_, err = s.service.PostReportConfiguration(ctx, &v1.PostReportConfigurationRequest{
		ReportConfig: incorrectEmailReportConfig,
	})
	s.Error(err)
}

func (s *ReportConfigurationServicePostgresTestSuite) TestUpdateInvalidValidReportConfigurations() {
	ctx := context.Background()

	s.notifierDatastore.EXPECT().Exists(ctx, gomock.Any()).Return(true, nil).AnyTimes()
	s.collectionDatastore.EXPECT().Exists(ctx, gomock.Any()).Return(true, nil).AnyTimes()

	noNotifierReportConfig := fixtures.GetInvalidReportConfigurationNoNotifier()
	_, err := s.service.UpdateReportConfiguration(ctx, &v1.UpdateReportConfigurationRequest{
		ReportConfig: noNotifierReportConfig,
	})
	s.Error(err)

	incorrectScheduleReportConfig := fixtures.GetInvalidReportConfigurationIncorrectSchedule()
	_, err = s.service.UpdateReportConfiguration(ctx, &v1.UpdateReportConfigurationRequest{
		ReportConfig: incorrectScheduleReportConfig,
	})
	s.Error(err)

	missingScheduleReportConfig := fixtures.GetInvalidReportConfigurationMissingSchedule()
	_, err = s.service.UpdateReportConfiguration(ctx, &v1.UpdateReportConfigurationRequest{
		ReportConfig: missingScheduleReportConfig,
	})
	s.Error(err)

	incorrectEmailReportConfig := fixtures.GetInvalidReportConfigurationIncorrectEmailV1()
	_, err = s.service.UpdateReportConfiguration(ctx, &v1.UpdateReportConfigurationRequest{
		ReportConfig: incorrectEmailReportConfig,
	})
	s.Error(err)
}

func (s *ReportConfigurationServicePostgresTestSuite) TestNotifierDoesNotExist() {
	ctx := context.Background()

	s.notifierDatastore.EXPECT().Exists(ctx, gomock.Any()).Return(false, nil)
	s.collectionDatastore.EXPECT().Exists(ctx, gomock.Any()).Return(true, nil).AnyTimes()

	reportConfig := fixtures.GetValidReportConfiguration()
	_, err := s.service.PostReportConfiguration(ctx, &v1.PostReportConfigurationRequest{
		ReportConfig: reportConfig,
	})
	s.Error(err)
}

func (s *ReportConfigurationServicePostgresTestSuite) TestAccessScopeDoesNotExist() {
	ctx := context.Background()

	s.notifierDatastore.EXPECT().Exists(ctx, gomock.Any()).Return(true, nil).AnyTimes()
	s.collectionDatastore.EXPECT().Exists(ctx, gomock.Any()).Return(false, nil)

	reportConfig := fixtures.GetValidReportConfiguration()
	_, err := s.service.PostReportConfiguration(ctx, &v1.PostReportConfigurationRequest{
		ReportConfig: reportConfig,
	})
	s.Error(err)
}

func (s *ReportConfigurationServicePostgresTestSuite) TestNoMailingAddresses() {
	ctx := context.Background()
	reportConfig := fixtures.GetValidReportConfiguration()
	reportConfig.GetEmailConfig().MailingLists = []string{}

	_, err := s.service.PostReportConfiguration(ctx, &v1.PostReportConfigurationRequest{
		ReportConfig: reportConfig,
	})
	s.Error(err)
}
