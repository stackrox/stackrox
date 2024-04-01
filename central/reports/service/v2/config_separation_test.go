//go:build sql_integration

package v2

import (
	"context"
	"fmt"
	"testing"

	notifierDS "github.com/stackrox/rox/central/notifier/datastore"
	"github.com/stackrox/rox/central/reports/common"
	reportConfigDS "github.com/stackrox/rox/central/reports/config/datastore"
	schedulerMocks "github.com/stackrox/rox/central/reports/scheduler/v2/mocks"
	snapshotDS "github.com/stackrox/rox/central/reports/snapshot/datastore"
	"github.com/stackrox/rox/central/reports/validation"
	collectionDS "github.com/stackrox/rox/central/resourcecollection/datastore"
	apiV2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestServiceLevelConfigSeparationV2(t *testing.T) {
	suite.Run(t, new(ServiceLevelConfigSeparationSuiteV2))
}

type ServiceLevelConfigSeparationSuiteV2 struct {
	suite.Suite
	service *serviceImpl

	testDB                *pgtest.TestPostgres
	ctx                   context.Context
	reportConfigDatastore reportConfigDS.DataStore
	notifierDatastore     notifierDS.DataStore
	collectionDatastore   collectionDS.DataStore
	scheduler             *schedulerMocks.MockScheduler
	snapshotDS            snapshotDS.DataStore

	mockCtrl *gomock.Controller

	v1Configs []*storage.ReportConfiguration
	v2Configs []*storage.ReportConfiguration
}

func (s *ServiceLevelConfigSeparationSuiteV2) SetupSuite() {
	s.T().Setenv(features.VulnReportingEnhancements.EnvVar(), "true")
	if !features.VulnReportingEnhancements.Enabled() {
		s.T().Skip("Skip test when reporting enhancements are disabled")
		s.T().SkipNow()
	}

	s.testDB = pgtest.ForT(s.T())
	s.reportConfigDatastore = reportConfigDS.GetTestPostgresDataStore(s.T(), s.testDB.DB)
	s.notifierDatastore = notifierDS.GetTestPostgresDataStore(s.T(), s.testDB.DB)
	s.snapshotDS = snapshotDS.GetTestPostgresDataStore(s.T(), s.testDB.DB)

	var err error
	s.collectionDatastore, _, err = collectionDS.GetTestPostgresDataStore(s.T(), s.testDB.DB)
	s.Require().NoError(err)

	s.mockCtrl = gomock.NewController(s.T())
	s.scheduler = schedulerMocks.NewMockScheduler(s.mockCtrl)

	validator := validation.New(s.reportConfigDatastore, nil, s.collectionDatastore, s.notifierDatastore)
	s.service = &serviceImpl{
		reportConfigStore:   s.reportConfigDatastore,
		collectionDatastore: s.collectionDatastore,
		notifierDatastore:   s.notifierDatastore,
		scheduler:           s.scheduler,
		snapshotDatastore:   s.snapshotDS,
		validator:           validator,
	}

	s.ctx = sac.WithAllAccess(context.Background())

	// Add notifier
	notifierID, err := s.notifierDatastore.AddNotifier(s.ctx, &storage.Notifier{Name: "notifier"})
	s.Require().NoError(err)

	// Add collection
	collectionID, err := s.collectionDatastore.AddCollection(s.ctx, &storage.ResourceCollection{Name: "collection"})
	s.Require().NoError(err)

	// Add report configs
	s.v1Configs = common.GetTestReportConfigsV1(s.T(), notifierID, collectionID)
	for _, conf := range s.v1Configs {
		conf.Id, err = s.reportConfigDatastore.AddReportConfiguration(s.ctx, conf)
		s.Require().NoError(err)
	}

	s.v2Configs = common.GetTestReportConfigsV2(s.T(), notifierID, collectionID)
	for _, conf := range s.v2Configs {
		conf.Id, err = s.reportConfigDatastore.AddReportConfiguration(s.ctx, conf)
		s.Require().NoError(err)
	}
}

func (s *ServiceLevelConfigSeparationSuiteV2) TearDownSuite() {
	s.mockCtrl.Finish()
	s.testDB.Teardown(s.T())
}

func (s *ServiceLevelConfigSeparationSuiteV2) TestListReportConfigurations() {
	apiV2Configs := s.convertConfigs(s.v2Configs)

	// Empty Query
	res, err := s.service.ListReportConfigurations(s.ctx, &apiV2.RawQuery{Query: ""})
	s.Require().NoError(err)
	s.Require().ElementsMatch(apiV2Configs, res.ReportConfigs)

	// Non empty query
	res, err = s.service.ListReportConfigurations(s.ctx,
		&apiV2.RawQuery{Query: fmt.Sprintf("Report Name:%s", s.v2Configs[0].Name)})
	s.Require().NoError(err)
	s.Require().Equal(1, len(res.ReportConfigs))
	s.Require().Equal(apiV2Configs[0], res.ReportConfigs[0])
}

func (s *ServiceLevelConfigSeparationSuiteV2) TestGetReportConfiguration() {
	apiV2Configs := s.convertConfigs(s.v2Configs)

	// returns v2 config
	res, err := s.service.GetReportConfiguration(s.ctx, &apiV2.ResourceByID{Id: s.v2Configs[0].Id})
	s.Require().NoError(err)
	s.Require().Equal(apiV2Configs[0], res)

	// error on requesting v1 config
	_, err = s.service.GetReportConfiguration(s.ctx, &apiV2.ResourceByID{Id: s.v1Configs[0].Id})
	s.Require().Error(err)
}

func (s *ServiceLevelConfigSeparationSuiteV2) TestCountReportConfigurations() {
	// Empty query
	res, err := s.service.CountReportConfigurations(s.ctx, &apiV2.RawQuery{Query: ""})
	s.Require().NoError(err)
	s.Require().Equal(int32(len(s.v2Configs)), res.Count)

	// Non empty query
	res, err = s.service.CountReportConfigurations(s.ctx,
		&apiV2.RawQuery{Query: fmt.Sprintf("Report Name:%s", s.v2Configs[0].Name)})
	s.Require().NoError(err)
	s.Require().Equal(int32(1), res.Count)
}

func (s *ServiceLevelConfigSeparationSuiteV2) TestDeleteReportConfiguration() {
	// Error on v1 config ID
	_, err := s.service.DeleteReportConfiguration(s.ctx, &apiV2.ResourceByID{Id: s.v1Configs[0].Id})
	s.Require().Error(err)

	// No error on v2 config ID
	s.scheduler.EXPECT().RemoveReportSchedule(gomock.Any()).Return().Times(1)
	config := s.v2Configs[0].Clone()
	config.Id = ""
	config.Name = "Delete report config"
	config.Id, err = s.reportConfigDatastore.AddReportConfiguration(s.ctx, config)
	s.Require().NoError(err)
	_, err = s.service.DeleteReportConfiguration(s.ctx, &apiV2.ResourceByID{Id: config.Id})
	s.Require().NoError(err)
}

func (s *ServiceLevelConfigSeparationSuiteV2) TestRunReport() {
	// Error on v1 config
	_, err := s.service.RunReport(s.ctx, &apiV2.RunReportRequest{
		ReportConfigId:           s.v1Configs[0].Id,
		ReportNotificationMethod: apiV2.NotificationMethod_EMAIL,
	})
	s.Require().Error(err)
}

func (s *ServiceLevelConfigSeparationSuiteV2) convertConfigs(configs []*storage.ReportConfiguration) []*apiV2.ReportConfiguration {
	apiV2Configs := make([]*apiV2.ReportConfiguration, 0, len(configs))
	for _, conf := range configs {
		c, err := s.service.convertProtoReportConfigurationToV2(conf)
		s.Require().NoError(err)
		apiV2Configs = append(apiV2Configs, c)
	}
	return apiV2Configs
}
