package compliancemanager

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac/testconsts"
	"github.com/stackrox/rox/pkg/sac/testutils"
	"go.uber.org/mock/gomock"
)

func (suite *complianceManagerTestSuite) expectPassesPreSSBChecks() {
	ctx := suite.testContexts[testutils.UnrestrictedReadWriteCtx]
	suite.scanConfigDS.EXPECT().ScanConfigurationProfileExists(ctx, gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)
	suite.scanConfigDS.EXPECT().GetScanConfigurationByName(ctx, mockScanName).Return(nil, nil).Times(1)
}

func (suite *complianceManagerTestSuite) expectSingleClusterSuccess() {
	ctx := suite.testContexts[testutils.UnrestrictedReadWriteCtx]
	suite.profileDS.EXPECT().SearchProfiles(ctx, gomock.Any()).Return([]*storage.ComplianceOperatorProfileV2{
		getTestProfile("ocp4-cis", "1.0.0", "platform", "ocp4", testconsts.Cluster1, 1),
	}, nil).Times(1)
	suite.scanConfigDS.EXPECT().UpsertScanConfiguration(ctx, gomock.Any()).Return(nil).Times(1)
	suite.connectionMgr.EXPECT().SendMessage(testconsts.Cluster1, gomock.Any()).Return(nil).Times(1)
	suite.clusterDatastore.EXPECT().GetClusterName(gomock.Any(), gomock.Any()).Return("test_cluster", true, nil).Times(1)
	suite.scanConfigDS.EXPECT().UpdateClusterStatus(ctx, gomock.Any(), testconsts.Cluster1, "", "test_cluster")
}

func (suite *complianceManagerTestSuite) TestProcessScanRequestSSBConflicts() {
	ctx := suite.testContexts[testutils.UnrestrictedReadWriteCtx]

	suite.Run("Profile conflict with external ScanSettingBinding", func() {
		suite.expectPassesPreSSBChecks()
		suite.ssbDS.EXPECT().GetScanSettingBindingsByCluster(gomock.Any(), testconsts.Cluster1).Return([]*storage.ComplianceOperatorScanSettingBindingV2{
			{Name: "external-argocd-ssb", ClusterId: testconsts.Cluster1, ProfileNames: []string{"ocp4-cis"}, Labels: map[string]string{"managed-by": "argocd"}},
		}, nil).Times(1)

		config, err := suite.manager.ProcessScanRequest(ctx, getTestRecNoID(), []string{testconsts.Cluster1})
		suite.Require().ErrorContains(err, "conflict with external ScanSettingBinding")
		suite.Require().Nil(config)
	})

	suite.Run("No conflict when external SSB uses different profiles", func() {
		suite.expectPassesPreSSBChecks()
		suite.ssbDS.EXPECT().GetScanSettingBindingsByCluster(gomock.Any(), testconsts.Cluster1).Return([]*storage.ComplianceOperatorScanSettingBindingV2{
			{Name: "external-argocd-ssb", ClusterId: testconsts.Cluster1, ProfileNames: []string{"ocp4-moderate"}, Labels: map[string]string{"managed-by": "argocd"}},
		}, nil).Times(1)
		suite.expectSingleClusterSuccess()

		config, err := suite.manager.ProcessScanRequest(ctx, getTestRecNoID(), []string{testconsts.Cluster1})
		suite.Require().NoError(err)
		suite.Require().NotNil(config)
	})

	suite.Run("Stackrox-managed SSB is ignored during conflict check", func() {
		suite.expectPassesPreSSBChecks()
		suite.ssbDS.EXPECT().GetScanSettingBindingsByCluster(gomock.Any(), testconsts.Cluster1).Return([]*storage.ComplianceOperatorScanSettingBindingV2{
			{Name: "stackrox-managed-ssb", ClusterId: testconsts.Cluster1, ProfileNames: []string{"ocp4-cis"}, Labels: map[string]string{"app.kubernetes.io/name": "stackrox"}},
		}, nil).Times(1)
		suite.expectSingleClusterSuccess()

		config, err := suite.manager.ProcessScanRequest(ctx, getTestRecNoID(), []string{testconsts.Cluster1})
		suite.Require().NoError(err)
		suite.Require().NotNil(config)
	})

	suite.Run("SSB datastore error propagates", func() {
		suite.expectPassesPreSSBChecks()
		suite.ssbDS.EXPECT().GetScanSettingBindingsByCluster(gomock.Any(), testconsts.Cluster1).Return(nil, errors.New("db connection failed")).Times(1)

		config, err := suite.manager.ProcessScanRequest(ctx, getTestRecNoID(), []string{testconsts.Cluster1})
		suite.Require().ErrorContains(err, "checking for external SSB conflicts")
		suite.Require().Nil(config)
	})

	suite.Run("Conflicts across multiple clusters reported together", func() {
		suite.expectPassesPreSSBChecks()
		clusters := []string{testconsts.Cluster1, testconsts.Cluster2}
		suite.ssbDS.EXPECT().GetScanSettingBindingsByCluster(gomock.Any(), testconsts.Cluster1).Return([]*storage.ComplianceOperatorScanSettingBindingV2{
			{Name: "external-ssb-cluster1", ClusterId: testconsts.Cluster1, ProfileNames: []string{"ocp4-cis"}, Labels: map[string]string{"managed-by": "argocd"}},
		}, nil).Times(1)
		suite.ssbDS.EXPECT().GetScanSettingBindingsByCluster(gomock.Any(), testconsts.Cluster2).Return([]*storage.ComplianceOperatorScanSettingBindingV2{
			{Name: "external-ssb-cluster2", ClusterId: testconsts.Cluster2, ProfileNames: []string{"ocp4-cis"}, Labels: map[string]string{"managed-by": "argocd"}},
		}, nil).Times(1)

		config, err := suite.manager.ProcessScanRequest(ctx, getTestRecNoID(), clusters)
		suite.Require().ErrorContains(err, "external-ssb-cluster1")
		suite.Require().ErrorContains(err, "external-ssb-cluster2")
		suite.Require().ErrorContains(err, "remove the external ScanSettingBindings")
		suite.Require().Nil(config)
	})

	suite.Run("SSB with nil labels is treated as external", func() {
		suite.expectPassesPreSSBChecks()
		suite.ssbDS.EXPECT().GetScanSettingBindingsByCluster(gomock.Any(), testconsts.Cluster1).Return([]*storage.ComplianceOperatorScanSettingBindingV2{
			{Name: "unlabeled-ssb", ClusterId: testconsts.Cluster1, ProfileNames: []string{"ocp4-cis"}},
		}, nil).Times(1)

		config, err := suite.manager.ProcessScanRequest(ctx, getTestRecNoID(), []string{testconsts.Cluster1})
		suite.Require().ErrorContains(err, "conflict with external ScanSettingBinding")
		suite.Require().Nil(config)
	})
}
