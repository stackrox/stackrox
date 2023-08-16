//go:build sql_integration

package service

import (
	"context"
	"fmt"
	"testing"

	notifierDS "github.com/stackrox/rox/central/notifier/datastore"
	"github.com/stackrox/rox/central/reports/common"
	reportConfigDS "github.com/stackrox/rox/central/reports/config/datastore"
	managerMocks "github.com/stackrox/rox/central/reports/manager/mocks"
	collectionDS "github.com/stackrox/rox/central/resourcecollection/datastore"
	apiV1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestServiceLevelConfigSeparation(t *testing.T) {
	suite.Run(t, new(ServiceLevelConfigSeparationSuite))
}

type ServiceLevelConfigSeparationSuite struct {
	suite.Suite
	service Service

	testDB                *pgtest.TestPostgres
	ctx                   context.Context
	reportConfigDatastore reportConfigDS.DataStore
	notifierDatastore     notifierDS.DataStore
	collectionDatastore   collectionDS.DataStore
	manager               *managerMocks.MockManager
	mockCtrl              *gomock.Controller

	v1Configs []*storage.ReportConfiguration
	v2Configs []*storage.ReportConfiguration
}

func (s *ServiceLevelConfigSeparationSuite) SetupSuite() {
	s.testDB = pgtest.ForT(s.T())
	s.reportConfigDatastore = reportConfigDS.GetTestPostgresDataStore(s.T(), s.testDB.DB)
	s.notifierDatastore = notifierDS.GetTestPostgresDataStore(s.T(), s.testDB.DB)

	var err error
	s.collectionDatastore, _, err = collectionDS.GetTestPostgresDataStore(s.T(), s.testDB.DB)
	s.Require().NoError(err)

	s.mockCtrl = gomock.NewController(s.T())
	s.manager = managerMocks.NewMockManager(s.mockCtrl)
	s.service = New(s.reportConfigDatastore, s.notifierDatastore, s.collectionDatastore, s.manager)

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

func (s *ServiceLevelConfigSeparationSuite) TearDownSuite() {
	s.mockCtrl.Finish()
	s.testDB.Teardown(s.T())
}

func (s *ServiceLevelConfigSeparationSuite) TestGetReportConfigurations() {
	// Empty Query
	res, err := s.service.GetReportConfigurations(s.ctx, &apiV1.RawQuery{Query: ""})
	s.Require().NoError(err)
	s.Require().ElementsMatch(s.v1Configs, res.ReportConfigs)

	// Non empty query
	res, err = s.service.GetReportConfigurations(s.ctx,
		&apiV1.RawQuery{Query: fmt.Sprintf("Report Name:%s", s.v1Configs[0].Name)})
	s.Require().NoError(err)
	s.Require().Equal(1, len(res.ReportConfigs))
	s.Require().Equal(s.v1Configs[0], res.ReportConfigs[0])
}

func (s *ServiceLevelConfigSeparationSuite) TestGetReportConfiguration() {
	// returns v1 config
	res, err := s.service.GetReportConfiguration(s.ctx, &apiV1.ResourceByID{Id: s.v1Configs[0].Id})
	s.Require().NoError(err)
	s.Require().Equal(s.v1Configs[0], res.ReportConfig)

	// error on requesting v2 config
	_, err = s.service.GetReportConfiguration(s.ctx, &apiV1.ResourceByID{Id: s.v2Configs[0].Id})
	s.Require().Error(err)
}

func (s *ServiceLevelConfigSeparationSuite) TestCountReportConfigurations() {
	// Empty query
	res, err := s.service.CountReportConfigurations(s.ctx, &apiV1.RawQuery{Query: ""})
	s.Require().NoError(err)
	s.Require().Equal(int32(len(s.v1Configs)), res.Count)

	// Non empty query
	res, err = s.service.CountReportConfigurations(s.ctx,
		&apiV1.RawQuery{Query: fmt.Sprintf("Report Name:%s", s.v1Configs[0].Name)})
	s.Require().NoError(err)
	s.Require().Equal(int32(1), res.Count)
}

func (s *ServiceLevelConfigSeparationSuite) TestPostReportConfiguration() {
	// Error on v2 config
	config := s.v2Configs[0].Clone()
	config.Id = ""
	config.NotifierConfig = s.v1Configs[0].NotifierConfig
	_, err := s.service.PostReportConfiguration(s.ctx, &apiV1.PostReportConfigurationRequest{ReportConfig: config})
	s.Require().Error(err)

	// No error on v1 config
	s.manager.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(nil).Times(1)
	config = s.v1Configs[0].Clone()
	config.Id = ""
	res, err := s.service.PostReportConfiguration(s.ctx, &apiV1.PostReportConfigurationRequest{ReportConfig: config})
	s.Require().NoError(err)

	err = s.reportConfigDatastore.RemoveReportConfiguration(s.ctx, res.ReportConfig.Id)
	s.Require().NoError(err)
}

func (s *ServiceLevelConfigSeparationSuite) TestUpdateReportConfiguration() {
	// Error on v2 config
	config := s.v2Configs[0].Clone()
	config.NotifierConfig = s.v1Configs[0].NotifierConfig
	config.GetVulnReportFilters().SinceLastReport = true
	_, err := s.service.UpdateReportConfiguration(s.ctx, &apiV1.UpdateReportConfigurationRequest{ReportConfig: config})
	s.Require().Error(err)

	// No error on v1 config
	s.manager.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(nil).Times(1)
	s.v1Configs[0].GetVulnReportFilters().SinceLastReport = true
	_, err = s.service.UpdateReportConfiguration(s.ctx, &apiV1.UpdateReportConfigurationRequest{ReportConfig: s.v1Configs[0]})
	s.Require().NoError(err)
}

func (s *ServiceLevelConfigSeparationSuite) TestDeleteReportConfiguration() {
	// Error on v2 config ID
	_, err := s.service.DeleteReportConfiguration(s.ctx, &apiV1.ResourceByID{Id: s.v2Configs[0].Id})
	s.Require().Error(err)

	// No error on v1 config ID
	s.manager.EXPECT().Remove(gomock.Any(), gomock.Any()).Return(nil).Times(1)
	config := s.v1Configs[0].Clone()
	config.Id = ""
	config.Name = "Delete report config"
	config.Id, err = s.reportConfigDatastore.AddReportConfiguration(s.ctx, config)
	s.Require().NoError(err)
	_, err = s.service.DeleteReportConfiguration(s.ctx, &apiV1.ResourceByID{Id: config.Id})
	s.Require().NoError(err)
}
