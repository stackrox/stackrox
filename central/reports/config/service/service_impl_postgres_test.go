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
	prcr := &v1.PostReportConfigurationRequest{}
	prcr.SetReportConfig(reportConfig)
	_, err := s.service.PostReportConfiguration(ctx, prcr)
	s.NoError(err)
}

func (s *ReportConfigurationServicePostgresTestSuite) TestAddInvalidValidReportConfigurations() {
	ctx := context.Background()

	s.notifierDatastore.EXPECT().Exists(ctx, gomock.Any()).Return(true, nil).AnyTimes()
	s.collectionDatastore.EXPECT().Exists(ctx, gomock.Any()).Return(true, nil).AnyTimes()

	noNotifierReportConfig := fixtures.GetInvalidReportConfigurationNoNotifier()
	prcr := &v1.PostReportConfigurationRequest{}
	prcr.SetReportConfig(noNotifierReportConfig)
	_, err := s.service.PostReportConfiguration(ctx, prcr)
	s.Error(err)

	incorrectScheduleReportConfig := fixtures.GetInvalidReportConfigurationIncorrectSchedule()
	prcr2 := &v1.PostReportConfigurationRequest{}
	prcr2.SetReportConfig(incorrectScheduleReportConfig)
	_, err = s.service.PostReportConfiguration(ctx, prcr2)
	s.Error(err)

	missingScheduleReportConfig := fixtures.GetInvalidReportConfigurationMissingSchedule()
	prcr3 := &v1.PostReportConfigurationRequest{}
	prcr3.SetReportConfig(missingScheduleReportConfig)
	_, err = s.service.PostReportConfiguration(ctx, prcr3)
	s.Error(err)

	missingDaysOfWeekReportConfig := fixtures.GetInvalidReportConfigurationMissingDaysOfWeek()
	prcr4 := &v1.PostReportConfigurationRequest{}
	prcr4.SetReportConfig(missingDaysOfWeekReportConfig)
	_, err = s.service.PostReportConfiguration(ctx, prcr4)
	s.Error(err)

	missingDaysOfMonthReportConfig := fixtures.GetInvalidReportConfigurationMissingDaysOfMonth()
	prcr5 := &v1.PostReportConfigurationRequest{}
	prcr5.SetReportConfig(missingDaysOfMonthReportConfig)
	_, err = s.service.PostReportConfiguration(ctx, prcr5)
	s.Error(err)

	dailyScheduleReportConfig := fixtures.GetInvalidReportConfigurationDailySchedule()
	prcr6 := &v1.PostReportConfigurationRequest{}
	prcr6.SetReportConfig(dailyScheduleReportConfig)
	_, err = s.service.PostReportConfiguration(ctx, prcr6)
	s.Error(err)

	incorrectEmailReportConfig := fixtures.GetInvalidReportConfigurationIncorrectEmailV1()
	prcr7 := &v1.PostReportConfigurationRequest{}
	prcr7.SetReportConfig(incorrectEmailReportConfig)
	_, err = s.service.PostReportConfiguration(ctx, prcr7)
	s.Error(err)
}

func (s *ReportConfigurationServicePostgresTestSuite) TestUpdateInvalidValidReportConfigurations() {
	ctx := context.Background()

	s.notifierDatastore.EXPECT().Exists(ctx, gomock.Any()).Return(true, nil).AnyTimes()
	s.collectionDatastore.EXPECT().Exists(ctx, gomock.Any()).Return(true, nil).AnyTimes()

	noNotifierReportConfig := fixtures.GetInvalidReportConfigurationNoNotifier()
	urcr := &v1.UpdateReportConfigurationRequest{}
	urcr.SetReportConfig(noNotifierReportConfig)
	_, err := s.service.UpdateReportConfiguration(ctx, urcr)
	s.Error(err)

	incorrectScheduleReportConfig := fixtures.GetInvalidReportConfigurationIncorrectSchedule()
	urcr2 := &v1.UpdateReportConfigurationRequest{}
	urcr2.SetReportConfig(incorrectScheduleReportConfig)
	_, err = s.service.UpdateReportConfiguration(ctx, urcr2)
	s.Error(err)

	missingScheduleReportConfig := fixtures.GetInvalidReportConfigurationMissingSchedule()
	urcr3 := &v1.UpdateReportConfigurationRequest{}
	urcr3.SetReportConfig(missingScheduleReportConfig)
	_, err = s.service.UpdateReportConfiguration(ctx, urcr3)
	s.Error(err)

	incorrectEmailReportConfig := fixtures.GetInvalidReportConfigurationIncorrectEmailV1()
	urcr4 := &v1.UpdateReportConfigurationRequest{}
	urcr4.SetReportConfig(incorrectEmailReportConfig)
	_, err = s.service.UpdateReportConfiguration(ctx, urcr4)
	s.Error(err)
}

func (s *ReportConfigurationServicePostgresTestSuite) TestNotifierDoesNotExist() {
	ctx := context.Background()

	s.notifierDatastore.EXPECT().Exists(ctx, gomock.Any()).Return(false, nil)
	s.collectionDatastore.EXPECT().Exists(ctx, gomock.Any()).Return(true, nil).AnyTimes()

	reportConfig := fixtures.GetValidReportConfiguration()
	prcr := &v1.PostReportConfigurationRequest{}
	prcr.SetReportConfig(reportConfig)
	_, err := s.service.PostReportConfiguration(ctx, prcr)
	s.Error(err)
}

func (s *ReportConfigurationServicePostgresTestSuite) TestAccessScopeDoesNotExist() {
	ctx := context.Background()

	s.notifierDatastore.EXPECT().Exists(ctx, gomock.Any()).Return(true, nil).AnyTimes()
	s.collectionDatastore.EXPECT().Exists(ctx, gomock.Any()).Return(false, nil)

	reportConfig := fixtures.GetValidReportConfiguration()
	prcr := &v1.PostReportConfigurationRequest{}
	prcr.SetReportConfig(reportConfig)
	_, err := s.service.PostReportConfiguration(ctx, prcr)
	s.Error(err)
}

func (s *ReportConfigurationServicePostgresTestSuite) TestNoMailingAddresses() {
	ctx := context.Background()
	reportConfig := fixtures.GetValidReportConfiguration()
	reportConfig.GetEmailConfig().SetMailingLists([]string{})

	prcr := &v1.PostReportConfigurationRequest{}
	prcr.SetReportConfig(reportConfig)
	_, err := s.service.PostReportConfiguration(ctx, prcr)
	s.Error(err)
}
