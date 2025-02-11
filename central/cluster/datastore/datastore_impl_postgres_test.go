//go:build sql_integration

package datastore

import (
	"context"
	"fmt"
	"testing"
	"time"

	alertDatastore "github.com/stackrox/rox/central/alert/datastore"
	clusterPostgresStore "github.com/stackrox/rox/central/cluster/store/cluster/postgres"
	clusterHealthPostgresStore "github.com/stackrox/rox/central/cluster/store/clusterhealth/postgres"
	compliancePruning "github.com/stackrox/rox/central/complianceoperator/v2/pruner"
	clusterCVEDataStore "github.com/stackrox/rox/central/cve/cluster/datastore"
	deploymentDatastore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/hash/datastore"
	hashManager "github.com/stackrox/rox/central/hash/manager"
	imageIntegrationDataStore "github.com/stackrox/rox/central/imageintegration/datastore"
	namespace "github.com/stackrox/rox/central/namespace/datastore"
	networkBaselineManager "github.com/stackrox/rox/central/networkbaseline/manager"
	netEntityDataStore "github.com/stackrox/rox/central/networkgraph/entity/datastore"
	netFlowsDataStore "github.com/stackrox/rox/central/networkgraph/flow/datastore"
	nodeDataStore "github.com/stackrox/rox/central/node/datastore"
	podDatastore "github.com/stackrox/rox/central/pod/datastore"
	"github.com/stackrox/rox/central/ranking"
	k8sRoleDataStore "github.com/stackrox/rox/central/rbac/k8srole/datastore"
	k8sRoleBindingDataStore "github.com/stackrox/rox/central/rbac/k8srolebinding/datastore"
	secretDataStore "github.com/stackrox/rox/central/secret/datastore"
	"github.com/stackrox/rox/central/sensor/service/connection"
	serviceAccountDataStore "github.com/stackrox/rox/central/serviceaccount/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/alert/convert"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/jsonutil"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/sac"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/scoped"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

const (
	fakeClusterID   = "FAKECLUSTERID"
	mainImage       = "docker.io/stackrox/rox:latest"
	centralEndpoint = "central.stackrox:443"
)

func TestClusterDataStoreWithPostgres(t *testing.T) {
	suite.Run(t, new(ClusterPostgresDataStoreTestSuite))
}

type ClusterPostgresDataStoreTestSuite struct {
	suite.Suite

	ctx                       context.Context
	db                        *pgtest.TestPostgres
	nsDatastore               namespace.DataStore
	alertDatastore            alertDatastore.DataStore
	deploymentDatastore       deploymentDatastore.DataStore
	podDatastore              podDatastore.DataStore
	secretDatastore           secretDataStore.DataStore
	serviceAccountDatastore   serviceAccountDataStore.DataStore
	roleDatastore             k8sRoleDataStore.DataStore
	roleBindingDatastore      k8sRoleBindingDataStore.DataStore
	imageIntegrationDatastore imageIntegrationDataStore.DataStore
	clusterDatastore          DataStore
}

func (s *ClusterPostgresDataStoreTestSuite) SetupTest() {

	s.ctx = sac.WithAllAccess(context.Background())
	s.db = pgtest.ForT(s.T())
	clusterDBStore := clusterPostgresStore.New(s.db)
	clusterHealthDBStore := clusterHealthPostgresStore.New(s.db)
	nodeStore := nodeDataStore.GetTestPostgresDataStore(s.T(), s.db)
	netFlowStore, err := netFlowsDataStore.GetTestPostgresClusterDataStore(s.T(), s.db)
	s.NoError(err)
	netEntityStore, err := netEntityDataStore.GetTestPostgresDataStore(s.T(), s.db)
	s.NoError(err)
	networkBaselineM, err := networkBaselineManager.GetTestPostgresManager(s.T(), s.db)
	s.NoError(err)
	clusterCVEStore, err := clusterCVEDataStore.GetTestPostgresDataStore(s.T(), s.db)
	s.NoError(err)
	hashStore, err := datastore.GetTestPostgresDataStore(s.T(), s.db)
	s.NoError(err)
	sensorCnxMgr := connection.NewManager(hashManager.NewManager(hashStore))
	clusterRanker := ranking.ClusterRanker()
	compliancePruner := compliancePruning.GetTestPruner(s.T(), s.db)
	s.nsDatastore, err = namespace.GetTestPostgresDataStore(s.T(), s.db.DB)
	s.NoError(err)
	s.alertDatastore, err = alertDatastore.GetTestPostgresDataStore(s.T(), s.db.DB)
	s.NoError(err)
	s.deploymentDatastore, err = deploymentDatastore.GetTestPostgresDataStore(s.T(), s.db.DB)
	s.NoError(err)
	s.podDatastore, err = podDatastore.GetTestPostgresDataStore(s.T(), s.db.DB)
	s.NoError(err)
	s.secretDatastore, err = secretDataStore.GetTestPostgresDataStore(s.T(), s.db.DB)
	s.NoError(err)
	s.serviceAccountDatastore, err = serviceAccountDataStore.GetTestPostgresDataStore(s.T(), s.db.DB)
	s.NoError(err)
	s.roleDatastore = k8sRoleDataStore.GetTestPostgresDataStore(s.T(), s.db.DB)
	s.roleBindingDatastore = k8sRoleBindingDataStore.GetTestPostgresDataStore(s.T(), s.db.DB)
	s.imageIntegrationDatastore, err = imageIntegrationDataStore.GetTestPostgresDataStore(s.T(), s.db.DB)
	s.NoError(err)
	s.clusterDatastore, err = New(clusterDBStore, clusterHealthDBStore, clusterCVEStore,
		s.alertDatastore, s.imageIntegrationDatastore, s.nsDatastore, s.deploymentDatastore,
		nodeStore, s.podDatastore, s.secretDatastore, netFlowStore, netEntityStore,
		s.serviceAccountDatastore, s.roleDatastore, s.roleBindingDatastore, sensorCnxMgr, nil,
		clusterRanker, networkBaselineM, compliancePruner)
	s.NoError(err)
}

func (s *ClusterPostgresDataStoreTestSuite) TearDownTest() {
	s.db.Teardown(s.T())
}

// Test that when we try to remove a cluster that does not exist, we return an error.
func (s *ClusterPostgresDataStoreTestSuite) TestHandlesClusterDoesNotExist() {
	ctx := sac.WithAllAccess(context.Background())

	err := s.clusterDatastore.RemoveCluster(ctx, fakeClusterID, nil)
	// Expect that there is an error because our cluster does not exist.
	s.Error(err)
}

func (s *ClusterPostgresDataStoreTestSuite) TestRemoveCluster() {
	ctx := sac.WithAllAccess(context.Background())

	testCluster := &storage.Cluster{Name: fixtureconsts.ClusterName1, MainImage: mainImage, CentralApiEndpoint: centralEndpoint}
	clusterId, clusterAddErr := s.clusterDatastore.AddCluster(ctx, testCluster)
	s.NotEmpty(clusterId)
	s.NoError(clusterAddErr)

	testDeployment := &storage.Deployment{Id: fixtureconsts.Deployment1, ClusterId: clusterId, ClusterName: testCluster.Name}
	deploymentUpsertErr := s.deploymentDatastore.UpsertDeployment(ctx, testDeployment)
	s.NoError(deploymentUpsertErr)

	testPod := &storage.Pod{Id: fixtureconsts.PodUID1, ClusterId: clusterId}
	podUpsertErr := s.podDatastore.UpsertPod(ctx, testPod)
	s.NoError(podUpsertErr)

	testAlert := &storage.Alert{Id: fixtureconsts.Alert1, ClusterId: clusterId, ClusterName: testCluster.Name, Entity: convert.ToAlertDeployment(testDeployment)}
	alertUpsertErr := s.alertDatastore.UpsertAlert(ctx, testAlert)
	s.NoError(alertUpsertErr)

	testSecret := &storage.Secret{Id: fixtureconsts.AlertFake, ClusterId: clusterId, ClusterName: testCluster.Name}
	secretUpsertErr := s.secretDatastore.UpsertSecret(ctx, testSecret)
	s.NoError(secretUpsertErr)

	testServiceAccount := &storage.ServiceAccount{Id: fixtureconsts.ServiceAccount1, ClusterId: clusterId, ClusterName: testCluster.Name}
	serviceAccountUpsertErr := s.serviceAccountDatastore.UpsertServiceAccount(ctx, testServiceAccount)
	s.NoError(serviceAccountUpsertErr)

	testRole := &storage.K8SRole{Id: fixtureconsts.Role1, ClusterId: clusterId, ClusterName: testCluster.Name}
	roleUpsertErr := s.roleDatastore.UpsertRole(ctx, testRole)
	s.NoError(roleUpsertErr)

	testRoleBinding := &storage.K8SRoleBinding{Id: fixtureconsts.RoleBinding1, ClusterId: clusterId, ClusterName: testCluster.Name}
	roleBindingUpsertErr := s.roleBindingDatastore.UpsertRoleBinding(ctx, testRoleBinding)
	s.NoError(roleBindingUpsertErr)

	testImageIntegration := &storage.ImageIntegration{Id: fixtureconsts.AlertFake, ClusterId: clusterId}
	imageIntegrationId, imageIntegrationAddErr := s.imageIntegrationDatastore.AddImageIntegration(ctx, testImageIntegration)
	s.NotEmpty(imageIntegrationId)
	s.NoError(imageIntegrationAddErr)

	// Remove cluster and verify that the removal has been cascaded to all related components
	doneSignal := concurrency.NewSignal()
	clusterRemoveErr := s.clusterDatastore.RemoveCluster(ctx, clusterId, &doneSignal)
	s.NoError(clusterRemoveErr)
	s.True(concurrency.WaitWithTimeout(&doneSignal, 10*time.Second))

	_, deploymentFound, deploymentGetErr := s.deploymentDatastore.GetDeployment(ctx, testDeployment.GetId())
	s.NoError(deploymentGetErr)
	s.False(deploymentFound)

	_, podFound, podGetErr := s.podDatastore.GetPod(ctx, testPod.GetId())
	s.NoError(podGetErr)
	s.False(podFound)

	alert, alertFound, alertFoundErr := s.alertDatastore.GetAlert(ctx, testAlert.GetId())
	s.NoError(alertFoundErr)
	// Verify that the alert was not deleted, but it was marked as resolved as the cluster it was related to has been removed
	s.True(alertFound)
	s.Equal(storage.ViolationState_RESOLVED, alert.GetState())

	_, secretFound, secretGetErr := s.secretDatastore.GetSecret(ctx, testSecret.GetId())
	s.NoError(secretGetErr)
	s.False(secretFound)

	_, serviceAccountFound, serviceAccountGetErr := s.serviceAccountDatastore.GetServiceAccount(ctx, testServiceAccount.GetId())
	s.NoError(serviceAccountGetErr)
	s.False(serviceAccountFound)

	_, roleFound, roleGetErr := s.roleDatastore.GetRole(ctx, testRole.GetId())
	s.NoError(roleGetErr)
	s.False(roleFound)

	_, roleBindingFound, roleBindingGetErr := s.roleBindingDatastore.GetRoleBinding(ctx, testRole.GetId())
	s.NoError(roleBindingGetErr)
	s.False(roleBindingFound)

	_, imageIntegrationFound, imageIntegrationGetErr := s.imageIntegrationDatastore.GetImageIntegration(ctx, testImageIntegration.GetId())
	s.NoError(imageIntegrationGetErr)
	s.False(imageIntegrationFound)
}

func (s *ClusterPostgresDataStoreTestSuite) TestPopulateClusterHealthInfo() {
	ctx := sac.WithAllAccess(context.Background())

	t := time.Now()
	ts := protoconv.ConvertTimeToTimestamp(t)
	healthStatuses := []storage.ClusterHealthStatus_HealthStatusLabel{storage.ClusterHealthStatus_UNHEALTHY, storage.ClusterHealthStatus_DEGRADED, storage.ClusterHealthStatus_HEALTHY}
	expectedHealths := make(map[string]*storage.ClusterHealthStatus)
	for i, status := range healthStatuses {
		clusterName := "Cluster" + string(rune(i+1))
		expectedHealths[clusterName] = &storage.ClusterHealthStatus{
			LastContact:        ts,
			SensorHealthStatus: status,
		}
		cluster := &storage.Cluster{Name: clusterName, MainImage: mainImage, CentralApiEndpoint: centralEndpoint}
		clusterId, err := s.clusterDatastore.AddCluster(ctx, cluster)
		s.NoError(err)
		s.NotEmpty(clusterId)
		cluster.Id = clusterId
		err = s.clusterDatastore.UpdateClusterHealth(ctx, clusterId, expectedHealths[clusterName])
		s.NoError(err)
	}
	results, err := s.clusterDatastore.SearchRawClusters(ctx, pkgSearch.EmptyQuery())
	s.NoError(err)
	for _, result := range results {
		protoassert.Equal(s.T(), expectedHealths[result.Name], result.HealthStatus)
	}
}

func (s *ClusterPostgresDataStoreTestSuite) TestLookupOrCreateClusterFromConfig() {
	const bundleID = "test-bundle-id"
	const policyVersion = "1"
	const someHelmConfigJSON = `{
		"staticConfig": {
		  "type": "KUBERNETES_CLUSTER",
		  "mainImage": "docker.io/stackrox/main",
		  "centralApiEndpoint": "central.stackrox.svc:443",
		  "collectorImage": "docker.io/stackrox/collector"
		},
		"configFingerprint": "1234"
	  }`
	const differentConfigFPHelmConfigJSON = `{
		"staticConfig": {
		  "type": "KUBERNETES_CLUSTER",
		  "mainImage": "docker.io/stackrox/main",
		  "centralApiEndpoint": "central.stackrox.svc:443",
		  "collectorImage": "docker.io/stackrox/collector"
		},
		"configFingerprint": "12345"
	  }`
	var someHelmConfig storage.CompleteClusterConfig
	var differentConfigFPHelmConfig storage.CompleteClusterConfig

	err := jsonutil.JSONToProto(someHelmConfigJSON, &someHelmConfig)
	s.NoError(err)

	err = jsonutil.JSONToProto(differentConfigFPHelmConfigJSON, &differentConfigFPHelmConfig)
	s.NoError(err)

	defaultSensorId := func() *storage.SensorDeploymentIdentification {
		return &storage.SensorDeploymentIdentification{
			AppNamespace:       "stackrox",
			SystemNamespaceId:  "123",
			AppNamespaceId:     "123",
			DefaultNamespaceId: "123",
		}
	}

	someClusterWithManagerType := func(managerType storage.ManagerType, helmConfig *storage.CompleteClusterConfig, capabilities []string) *storage.Cluster {
		return &storage.Cluster{
			Id:                 "",
			Name:               "",
			InitBundleId:       bundleID,
			HelmConfig:         helmConfig,
			MainImage:          mainImage,
			CentralApiEndpoint: centralEndpoint,
			ManagedBy:          managerType,
			MostRecentSensorId: defaultSensorId(),
			SensorCapabilities: capabilities,
			HealthStatus: &storage.ClusterHealthStatus{
				SensorHealthStatus: storage.ClusterHealthStatus_HEALTHY,
				LastContact:        protoconv.ConvertTimeToTimestamp(time.Now()),
			},
		}
	}

	sensorHelloWithHelmManagedConfigInit := func(helmManagedConfigInit *central.HelmManagedConfigInit, identification *storage.SensorDeploymentIdentification, capabilities []string) *central.SensorHello {
		if identification == nil {
			identification = defaultSensorId()
		}
		return &central.SensorHello{
			DeploymentIdentification: identification,
			HelmManagedConfigInit:    helmManagedConfigInit,
			PolicyVersion:            policyVersion,
			Capabilities:             capabilities,
		}
	}

	cases := []struct {
		description                string
		cluster                    *storage.Cluster
		sensorHello                *central.SensorHello
		clusterID                  string
		shouldClusterBeUpserted    bool
		shouldHaveLookupError      bool
		expectedManagerType        storage.ManagerType
		expectedHelmConfig         *storage.CompleteClusterConfig
		expectedSensorCapabilities []string
	}{
		{
			description: "test if cluster with mismatched name to config throws an error",
			cluster:     someClusterWithManagerType(storage.ManagerType_MANAGER_TYPE_HELM_CHART, &someHelmConfig, nil),
			sensorHello: sensorHelloWithHelmManagedConfigInit(&central.HelmManagedConfigInit{
				ClusterName:   "NotTheRightName",
				ManagedBy:     storage.ManagerType_MANAGER_TYPE_HELM_CHART,
				ClusterConfig: &someHelmConfig,
			}, nil, nil),
			shouldClusterBeUpserted: true,
			shouldHaveLookupError:   true,
		},
		{
			description: "test if cluster with id that does not exist throws an error",
			cluster:     someClusterWithManagerType(storage.ManagerType_MANAGER_TYPE_HELM_CHART, &someHelmConfig, nil),
			sensorHello: sensorHelloWithHelmManagedConfigInit(&central.HelmManagedConfigInit{
				ClusterConfig: &someHelmConfig,
				ManagedBy:     storage.ManagerType_MANAGER_TYPE_HELM_CHART,
			}, nil, nil),
			clusterID:               uuid.NewV4().String(),
			shouldClusterBeUpserted: false,
			shouldHaveLookupError:   true,
		},
		{
			description: "try adding a cluster that has already been added with a different namespace for StackRox",
			cluster:     someClusterWithManagerType(storage.ManagerType_MANAGER_TYPE_HELM_CHART, &someHelmConfig, nil),
			sensorHello: sensorHelloWithHelmManagedConfigInit(&central.HelmManagedConfigInit{
				ClusterConfig: &someHelmConfig,
				ManagedBy:     storage.ManagerType_MANAGER_TYPE_HELM_CHART,
			}, &storage.SensorDeploymentIdentification{
				AppNamespace: "NotTheRightNamespace",
			}, nil),
			shouldClusterBeUpserted: true,
			shouldHaveLookupError:   true,
		},
		{
			description: "try adding a cluster that has already been added with a different system namespace ID",
			cluster:     someClusterWithManagerType(storage.ManagerType_MANAGER_TYPE_HELM_CHART, &someHelmConfig, nil),
			sensorHello: sensorHelloWithHelmManagedConfigInit(&central.HelmManagedConfigInit{
				ClusterConfig: &someHelmConfig,
				ManagedBy:     storage.ManagerType_MANAGER_TYPE_HELM_CHART,
			}, &storage.SensorDeploymentIdentification{
				AppNamespace:      "stackrox",
				SystemNamespaceId: "456",
			}, nil),
			shouldClusterBeUpserted: true,
			shouldHaveLookupError:   true,
		},
		{
			description: "try adding a cluster and then seeing if we get the same cluster back",
			cluster:     someClusterWithManagerType(storage.ManagerType_MANAGER_TYPE_HELM_CHART, &someHelmConfig, nil),
			sensorHello: sensorHelloWithHelmManagedConfigInit(&central.HelmManagedConfigInit{
				ClusterConfig: &someHelmConfig,
				ManagedBy:     storage.ManagerType_MANAGER_TYPE_HELM_CHART,
			}, nil, nil),
			shouldClusterBeUpserted: true,
			expectedManagerType:     storage.ManagerType_MANAGER_TYPE_HELM_CHART,
			expectedHelmConfig:      &someHelmConfig,
		},
		{
			description: "try adding a cluster and then changing the manager type",
			cluster:     someClusterWithManagerType(storage.ManagerType_MANAGER_TYPE_HELM_CHART, &someHelmConfig, nil),
			sensorHello: sensorHelloWithHelmManagedConfigInit(&central.HelmManagedConfigInit{
				ManagedBy: storage.ManagerType_MANAGER_TYPE_MANUAL,
			}, nil, nil),
			shouldClusterBeUpserted: true,
			expectedManagerType:     storage.ManagerType_MANAGER_TYPE_MANUAL,
			expectedHelmConfig:      nil,
		},
		{
			description: "try adding a cluster and then changing the helm config",
			cluster:     someClusterWithManagerType(storage.ManagerType_MANAGER_TYPE_HELM_CHART, &someHelmConfig, nil),
			sensorHello: sensorHelloWithHelmManagedConfigInit(&central.HelmManagedConfigInit{
				ManagedBy:     storage.ManagerType_MANAGER_TYPE_HELM_CHART,
				ClusterConfig: &differentConfigFPHelmConfig,
			}, nil, nil),
			shouldClusterBeUpserted: true,
			expectedManagerType:     storage.ManagerType_MANAGER_TYPE_HELM_CHART,
			expectedHelmConfig:      &differentConfigFPHelmConfig,
		},
		{
			description: "test that sensor capabilities match does not trigger an update",
			cluster:     someClusterWithManagerType(storage.ManagerType_MANAGER_TYPE_HELM_CHART, &someHelmConfig, []string{"capability1", "capability2"}),
			sensorHello: sensorHelloWithHelmManagedConfigInit(&central.HelmManagedConfigInit{
				ClusterConfig: &someHelmConfig,
				ManagedBy:     storage.ManagerType_MANAGER_TYPE_HELM_CHART,
			}, nil, []string{"capability2", "capability1"}), // capability ordering should not matter
			shouldClusterBeUpserted:    false,
			expectedSensorCapabilities: []string{"capability1", "capability2"},
		},
		{
			description: "test that sensor capabilities mismatch triggers an update",
			cluster:     someClusterWithManagerType(storage.ManagerType_MANAGER_TYPE_HELM_CHART, &someHelmConfig, []string{"capability1", "capability2"}),
			sensorHello: sensorHelloWithHelmManagedConfigInit(&central.HelmManagedConfigInit{
				ClusterConfig: &someHelmConfig,
				ManagedBy:     storage.ManagerType_MANAGER_TYPE_HELM_CHART,
			}, nil, []string{"capability3", "capability4"}),
			shouldClusterBeUpserted:    true,
			expectedSensorCapabilities: []string{"capability3", "capability4"},
		},
		{
			description: "test that sensor capabilities update correctly with partial overlap",
			cluster:     someClusterWithManagerType(storage.ManagerType_MANAGER_TYPE_HELM_CHART, &someHelmConfig, []string{"capability1", "capability2"}),
			sensorHello: sensorHelloWithHelmManagedConfigInit(&central.HelmManagedConfigInit{
				ClusterConfig: &someHelmConfig,
				ManagedBy:     storage.ManagerType_MANAGER_TYPE_HELM_CHART,
			}, nil, []string{"capability3", "capability2"}),
			shouldClusterBeUpserted:    true,
			expectedSensorCapabilities: []string{"capability2", "capability3"},
		},
		{
			description: "try creating a cluster from a config",
			cluster:     nil,
			sensorHello: sensorHelloWithHelmManagedConfigInit(&central.HelmManagedConfigInit{
				ManagedBy:     storage.ManagerType_MANAGER_TYPE_HELM_CHART,
				ClusterConfig: &someHelmConfig,
			}, nil, []string{"capability2", "capability1", "capability3"}),
			shouldClusterBeUpserted:    false,
			expectedManagerType:        storage.ManagerType_MANAGER_TYPE_HELM_CHART,
			expectedHelmConfig:         &someHelmConfig,
			expectedSensorCapabilities: []string{"capability1", "capability2", "capability3"},
		},
	}

	for i, c := range cases {
		s.T().Run(c.description, func(t *testing.T) {
			var clusterID string
			var cluster *storage.Cluster
			var lookupErr error

			// Make a unique cluster name to simplify testing code
			clusterName := fmt.Sprintf("cluster%d", i)
			if c.cluster != nil {
				c.cluster.Name = clusterName
			}
			if helmCfg := c.sensorHello.GetHelmManagedConfigInit(); helmCfg != nil && helmCfg.ClusterName == "" {
				helmCfg.ClusterName = clusterName
			}

			ctx := sac.WithAllAccess(context.Background())

			if c.shouldClusterBeUpserted {
				clusterID, err = s.clusterDatastore.AddCluster(ctx, c.cluster)
				s.NoError(err)
			}

			if c.clusterID != "" {
				clusterID = c.clusterID
			}

			cluster, lookupErr = s.clusterDatastore.LookupOrCreateClusterFromConfig(ctx, clusterID, bundleID, c.sensorHello)
			if c.shouldHaveLookupError {
				s.Error(lookupErr)
			} else {
				s.NoError(lookupErr)
			}
			if c.expectedManagerType != 0 {
				s.Equal(c.expectedManagerType, cluster.ManagedBy)
			}
			if c.expectedHelmConfig != nil {
				protoassert.Equal(s.T(), c.expectedHelmConfig, cluster.GetHelmConfig())
			}
			if c.expectedSensorCapabilities != nil {
				s.Equal(c.expectedSensorCapabilities, cluster.SensorCapabilities)
			}
		})
	}
}

func (s *ClusterPostgresDataStoreTestSuite) TestUpdateAuditLogFileStates() {
	t1 := time.Now()
	t2 := time.Now()
	ts1 := protoconv.ConvertTimeToTimestamp(t1)
	ts2 := protoconv.ConvertTimeToTimestamp(t2)

	fakeCluster := &storage.Cluster{Name: fixtureconsts.ClusterName1, MainImage: mainImage, CentralApiEndpoint: centralEndpoint}

	states := map[string]*storage.AuditLogFileState{
		"node-1": {CollectLogsSince: ts1, LastAuditId: "abcd"},
		"node-2": {CollectLogsSince: ts2, LastAuditId: "efgh"},
		"node-3": {CollectLogsSince: ts1, LastAuditId: "ijkl"},
	}

	ctx := sac.WithAllAccess(context.Background())

	realClusterID, err := s.clusterDatastore.AddCluster(ctx, fakeCluster)
	s.NoError(err)
	s.NotEmpty(realClusterID)
	fakeCluster.Id = realClusterID
	err = s.clusterDatastore.UpdateAuditLogFileStates(ctx, realClusterID, states)
	s.NoError(err)
}

func (s *ClusterPostgresDataStoreTestSuite) TestUpdateAuditLogFileStatesLeaveUnmodifiedNodesAlone() {
	t1 := time.Now()
	t2 := time.Now().Add(-30 * time.Minute)
	t3 := time.Now().Add(-10 * time.Minute)
	ts1 := protoconv.ConvertTimeToTimestamp(t1)
	ts2 := protoconv.ConvertTimeToTimestamp(t2)
	ts3 := protoconv.ConvertTimeToTimestamp(t3)

	fakeCluster := &storage.Cluster{
		Name:               fixtureconsts.ClusterName1,
		MainImage:          mainImage,
		CentralApiEndpoint: centralEndpoint,
		AuditLogState: map[string]*storage.AuditLogFileState{
			"old-node1": {CollectLogsSince: ts3, LastAuditId: "aaaa"},
		},
	}

	ctx := sac.WithAllAccess(context.Background())

	realClusterID, err := s.clusterDatastore.AddCluster(ctx, fakeCluster)
	s.NoError(err)

	fakeCluster.Id = realClusterID

	newStates := map[string]*storage.AuditLogFileState{
		"node-1": {CollectLogsSince: ts1, LastAuditId: "bbbb"},
		"node-2": {CollectLogsSince: ts2, LastAuditId: "cccc"},
	}

	expectedStates := map[string]*storage.AuditLogFileState{
		"old-node1": {CollectLogsSince: ts3, LastAuditId: "aaaa"},
		"node-1":    {CollectLogsSince: ts1, LastAuditId: "bbbb"},
		"node-2":    {CollectLogsSince: ts2, LastAuditId: "cccc"},
	}

	err = s.clusterDatastore.UpdateAuditLogFileStates(ctx, realClusterID, newStates)
	s.NoError(err)

	realCluster, exists, err := s.clusterDatastore.GetCluster(ctx, realClusterID)
	s.NoError(err)
	s.True(exists)

	protoassert.MapEqual(s.T(), expectedStates, realCluster.AuditLogState)
}

func (s *ClusterPostgresDataStoreTestSuite) TestUpdateAuditLogFileStatesErrorConditions() {
	t1 := time.Now()
	t2 := time.Now().Add(-30 * time.Minute)
	ts1 := protoconv.ConvertTimeToTimestamp(t1)
	ts2 := protoconv.ConvertTimeToTimestamp(t2)

	fakeCluster := &storage.Cluster{Name: fixtureconsts.ClusterName1, MainImage: mainImage, CentralApiEndpoint: centralEndpoint}

	states := map[string]*storage.AuditLogFileState{
		"node-1": {CollectLogsSince: ts1, LastAuditId: "abcd"},
		"node-2": {CollectLogsSince: ts2, LastAuditId: "efgh"},
		"node-3": {CollectLogsSince: ts1, LastAuditId: "zyxw"},
	}

	cases := []struct {
		name             string
		ctx              context.Context
		clusterID        string
		states           map[string]*storage.AuditLogFileState
		clusterIsMissing bool
		realClusterFound bool
	}{
		{
			name:             "Error when no cluster id is provided",
			ctx:              sac.WithAllAccess(context.Background()),
			clusterID:        "",
			states:           states,
			clusterIsMissing: false,
		},
		{
			name:             "Error when no states are provided",
			ctx:              sac.WithAllAccess(context.Background()),
			clusterID:        fakeClusterID,
			states:           nil,
			clusterIsMissing: false,
		},
		{
			name:             "Error when empty states are provided",
			ctx:              sac.WithAllAccess(context.Background()),
			clusterID:        fakeClusterID,
			states:           map[string]*storage.AuditLogFileState{},
			clusterIsMissing: false,
		},
		{
			name:             "Error when context has no perms",
			ctx:              sac.WithNoAccess(context.Background()),
			clusterID:        fakeClusterID,
			states:           states,
			clusterIsMissing: false,
		},
		{
			name:             "Error when is not read only",
			ctx:              sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowFixedScopes(sac.AccessModeScopeKeys(storage.Access_READ_ACCESS))),
			clusterID:        fakeClusterID,
			states:           states,
			clusterIsMissing: false,
			realClusterFound: false,
		},
		{
			name:             "Error when cluster cannot be found",
			ctx:              sac.WithAllAccess(context.Background()),
			clusterID:        fakeClusterID,
			states:           states,
			clusterIsMissing: true,
			realClusterFound: false,
		},
	}

	for _, c := range cases {
		s.T().Run(c.name, func(t *testing.T) {
			var clusterID string
			var cluster *storage.Cluster
			var found bool
			var err error

			writeCtx := sac.WithAllAccess(context.Background())
			if c.clusterIsMissing {
				cluster, found, err = s.clusterDatastore.GetCluster(writeCtx, c.clusterID)
				s.NoError(err)
				s.Nil(cluster)
				s.False(found)
			}
			if c.realClusterFound {
				clusterID, err = s.clusterDatastore.AddCluster(writeCtx, fakeCluster)
				c.clusterID = clusterID
				s.NoError(err)
				cluster, found, err = s.clusterDatastore.GetCluster(writeCtx, c.clusterID)
				s.True(found)
				s.NotNil(cluster)
				s.NoError(err)
			}
			err = s.clusterDatastore.UpdateAuditLogFileStates(c.ctx, c.clusterID, c.states)
			s.Error(err)
		})
	}
}

func (s *ClusterPostgresDataStoreTestSuite) TestNormalizeCluster() {
	cases := []struct {
		name     string
		cluster  *storage.Cluster
		expected string
	}{
		{
			name: "happy path",
			cluster: &storage.Cluster{
				CentralApiEndpoint: centralEndpoint,
				MainImage:          mainImage,
				Name:               "cluster1",
			},
			expected: centralEndpoint,
		},
		{
			name: "http",
			cluster: &storage.Cluster{
				CentralApiEndpoint: "http://" + centralEndpoint,
				MainImage:          mainImage,
				Name:               "cluster2",
			},
			expected: centralEndpoint,
		},
		{
			name: "https",
			cluster: &storage.Cluster{
				CentralApiEndpoint: "https://" + centralEndpoint,
				MainImage:          mainImage,
				Name:               "cluster3",
			},
			expected: centralEndpoint,
		},
	}

	for _, c := range cases {
		s.T().Run(c.name, func(t *testing.T) {
			s.NoError(normalizeCluster(c.cluster))
			s.Equal(c.expected, c.cluster.GetCentralApiEndpoint())
		})
	}
}

func (s *ClusterPostgresDataStoreTestSuite) TestValidateCluster() {
	cases := []struct {
		name          string
		cluster       *storage.Cluster
		expectedError bool
	}{
		{
			name:          "Empty Cluster",
			cluster:       &storage.Cluster{},
			expectedError: true,
		},
		{
			name: "No name",
			cluster: &storage.Cluster{
				MainImage:          "image",
				CentralApiEndpoint: "central:443",
			},
			expectedError: true,
		},
		{
			name: "No Image",
			cluster: &storage.Cluster{
				Name:               "name",
				CentralApiEndpoint: "central:443",
			},
			expectedError: true,
		},
		{
			name: "Image without tag",
			cluster: &storage.Cluster{
				MainImage:          "stackrox/main",
				Name:               "name",
				CentralApiEndpoint: "central:443",
			},
			expectedError: false,
		},
		{
			name: "Non-trivial image",
			cluster: &storage.Cluster{
				MainImage:          "stackrox/main:1.2",
				Name:               "name",
				CentralApiEndpoint: "central:443",
			},
			expectedError: false,
		},
		{
			name: "Moderately complex image",
			cluster: &storage.Cluster{
				MainImage:          "stackrox.io/main:1.2.512-125125",
				Name:               "name",
				CentralApiEndpoint: "central:443",
			},
			expectedError: false,
		},
		{
			name: "Image with SHA",
			cluster: &storage.Cluster{
				MainImage:          "stackrox.io/main@sha256:45b23dee08af5e43a7fea6c4cf9c25ccf269ee113168c19722f87876677c5cb2",
				Name:               "name",
				CentralApiEndpoint: "central:443",
			},
			expectedError: false,
		},
		{
			name: "Invalid image - contains spaces",
			cluster: &storage.Cluster{
				MainImage:          "stackrox.io/main:1.2.3 injectedCommand",
				Name:               "name",
				CentralApiEndpoint: "central:443",
			},
			expectedError: true,
		},
		{
			name: "No Central Endpoint",
			cluster: &storage.Cluster{
				Name:      "name",
				MainImage: "image",
			},
			expectedError: true,
		},
		{
			name: "Central Endpoint w/o port",
			cluster: &storage.Cluster{
				Name:               "name",
				MainImage:          "image",
				CentralApiEndpoint: "central",
			},
			expectedError: true,
		},
		{
			name: "Valid collector registry",
			cluster: &storage.Cluster{
				Name:               "name",
				MainImage:          "image",
				CentralApiEndpoint: "central:443",
				CollectorImage:     "collector.stackrox.io/collector",
			},
			expectedError: false,
		},
		{
			name: "Empty string collector registry",
			cluster: &storage.Cluster{
				Name:               "name",
				MainImage:          "image",
				CentralApiEndpoint: "central:443",
				CollectorImage:     "",
			},
		},
		{
			name: "Invalid collector registry",
			cluster: &storage.Cluster{
				Name:               "name",
				MainImage:          "image",
				CentralApiEndpoint: "central:443",
				CollectorImage:     "collector.stackrox.io/collector injectedCommand",
			},
			expectedError: true,
		},
		{
			name: "Happy path K8s",
			cluster: &storage.Cluster{
				Name:               "name",
				MainImage:          "image",
				CentralApiEndpoint: "central:443",
			},
			expectedError: false,
		},
		{
			name: "Happy path",
			cluster: &storage.Cluster{
				Name:               "name",
				MainImage:          "image",
				CentralApiEndpoint: "central:443",
			},
			expectedError: false,
		},
	}

	for _, c := range cases {
		s.T().Run(c.name, func(t *testing.T) {
			cluster := c.cluster.CloneVT()
			cluster.DynamicConfig = &storage.DynamicClusterConfig{
				DisableAuditLogs: true,
			}
			err := validateInput(cluster)
			if c.expectedError {
				s.Error(err)
			} else {
				s.Nil(err)
			}
		})
	}
}

func (s *ClusterPostgresDataStoreTestSuite) TestAddExistingCluster() {
	ctx := sac.WithAllAccess(context.Background())

	cluster := storage.Cluster{Name: fixtureconsts.ClusterName1, MainImage: mainImage, CentralApiEndpoint: centralEndpoint}
	clusterID, err := s.clusterDatastore.AddCluster(ctx, &cluster)
	s.NoError(err)
	s.NotEmpty(clusterID)
	cluster.Id = ""
	_, err = s.clusterDatastore.AddCluster(ctx, &cluster)
	s.Error(err)
	s.ErrorIs(err, errox.AlreadyExists)
}

func (s *ClusterPostgresDataStoreTestSuite) TestSearchClusterStatus() {
	ctx := sac.WithAllAccess(context.Background())

	// At some point in the postgres migration, the following query did trigger an error
	// because of a missing options map in the cluster health status schema.
	// This test is there to ensure the search does not end in error for technical reasons.
	query := pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.ClusterStatus, storage.ClusterHealthStatus_UNHEALTHY.String()).ProtoQuery()
	res, err := s.clusterDatastore.Search(ctx, query)
	s.NoError(err)
	s.Equal(0, len(res))
}

func (s *ClusterPostgresDataStoreTestSuite) TestAddDefaults() {

	s.Run("Error on nil cluster", func() {
		s.Error(addDefaults(nil))
	})

	s.Run("Some default values are set for uninitialized fields", func() {
		cluster := &storage.Cluster{CentralApiEndpoint: centralEndpoint, MainImage: mainImage}
		s.NoError(addDefaults(cluster))
		s.Empty(cluster.GetCollectorImage()) // must not be set
		s.Equal(centralEndpoint, cluster.GetCentralApiEndpoint())
		s.Equal(storage.CollectionMethod_CORE_BPF, cluster.GetCollectionMethod())
		tc := cluster.GetTolerationsConfig()
		s.Require().NotNil(tc)
		s.False(tc.GetDisabled())
		dc := cluster.GetDynamicConfig()
		s.Require().NotNil(dc)
		s.True(dc.GetDisableAuditLogs())
		acc := dc.GetAdmissionControllerConfig()
		s.Require().NotNil(acc)
		s.False(acc.GetEnabled())
		s.Equal(int32(defaultAdmissionControllerTimeout), acc.GetTimeoutSeconds())
	})

	s.Run("Provided values are either not overridden or properly updated", func() {
		cluster := &storage.Cluster{
			Id:                         fakeClusterID,
			Name:                       "someName",
			Type:                       storage.ClusterType_KUBERNETES_CLUSTER,
			Labels:                     map[string]string{"key": "value"},
			MainImage:                  "somevalue",
			CollectorImage:             "someOtherValue",
			CentralApiEndpoint:         "someEndpoint",
			CollectionMethod:           storage.CollectionMethod_CORE_BPF,
			AdmissionController:        true,
			AdmissionControllerUpdates: true,
			AdmissionControllerEvents:  true,
			DynamicConfig: &storage.DynamicClusterConfig{
				AdmissionControllerConfig: &storage.AdmissionControllerConfig{
					Enabled:        true,
					TimeoutSeconds: 73,
				},
				RegistryOverride: "registryOverride",
				DisableAuditLogs: false,
			},
			TolerationsConfig: &storage.TolerationsConfig{
				Disabled: true,
			},
			Priority:      10,
			SlimCollector: true,
			HelmConfig:    &storage.CompleteClusterConfig{},
			InitBundleId:  "someId",
			ManagedBy:     storage.ManagerType_MANAGER_TYPE_KUBERNETES_OPERATOR,
		}
		s.NoError(addDefaults(cluster))

		s.Equal(fakeClusterID, cluster.GetId())
		s.Equal("someName", cluster.GetName())
		s.Equal(storage.ClusterType_KUBERNETES_CLUSTER, cluster.GetType())
		s.EqualValues(map[string]string{"key": "value"}, cluster.GetLabels())

		s.Equal("somevalue", cluster.GetMainImage())
		s.Equal("someOtherValue", cluster.GetCollectorImage())
		s.Equal("someEndpoint", cluster.GetCentralApiEndpoint())
		s.Equal(storage.CollectionMethod_CORE_BPF, cluster.GetCollectionMethod())
		s.True(cluster.GetAdmissionController())
		s.True(cluster.GetAdmissionControllerUpdates())
		s.True(cluster.GetAdmissionControllerEvents())
		dc := cluster.GetDynamicConfig()
		s.Require().NotNil(dc)
		s.Equal("registryOverride", dc.GetRegistryOverride())
		s.True(dc.GetDisableAuditLogs()) // True for KUBERNETES_CLUSTER
		acc := dc.GetAdmissionControllerConfig()
		s.Require().NotNil(acc)
		s.True(acc.GetEnabled())
		s.Equal(int32(73), acc.GetTimeoutSeconds())
		tc := cluster.GetTolerationsConfig()
		s.Require().NotNil(tc)
		s.True(tc.GetDisabled())
		s.Equal(int64(10), cluster.GetPriority())
		s.True(cluster.SlimCollector)
		s.NotNil(cluster.GetHelmConfig())
		s.Equal("someId", cluster.GetInitBundleId())
		s.Equal(storage.ManagerType_MANAGER_TYPE_KUBERNETES_OPERATOR, cluster.GetManagedBy())
	})

	s.Run("Audit logs", func() {
		for name, testCase := range map[string]struct {
			cluster              *storage.Cluster
			expectedDisabledLogs bool
		}{
			"Kubernetes cluster":  {&storage.Cluster{Type: storage.ClusterType_KUBERNETES_CLUSTER, MainImage: mainImage, CentralApiEndpoint: centralEndpoint}, true},
			"Openshift 3 cluster": {&storage.Cluster{Type: storage.ClusterType_OPENSHIFT_CLUSTER, MainImage: mainImage, CentralApiEndpoint: centralEndpoint}, true},
			"Openshift 4 cluster": {&storage.Cluster{Type: storage.ClusterType_OPENSHIFT4_CLUSTER, MainImage: mainImage, CentralApiEndpoint: centralEndpoint}, false},
			"Openshift 4 cluster with disabled logs": {&storage.Cluster{Type: storage.ClusterType_OPENSHIFT4_CLUSTER, MainImage: mainImage, CentralApiEndpoint: centralEndpoint,
				DynamicConfig: &storage.DynamicClusterConfig{DisableAuditLogs: true}}, true},
		} {
			s.Run(name, func() {
				s.NoError(addDefaults(testCase.cluster))
				dc := testCase.cluster.GetDynamicConfig()
				s.Require().NotNil(dc)
				s.Equal(testCase.expectedDisabledLogs, dc.GetDisableAuditLogs())
			})
		}
	})

	s.Run("Collector image not set when only main image is provided", func() {
		cluster := &storage.Cluster{
			MainImage: "somevalue",
		}
		s.NoError(addDefaults(cluster))
		s.Empty(cluster.GetCollectorImage())
	})

	s.Run("Error for bad timeout", func() {
		cluster := &storage.Cluster{
			DynamicConfig: &storage.DynamicClusterConfig{
				AdmissionControllerConfig: &storage.AdmissionControllerConfig{
					TimeoutSeconds: -1,
				}},
		}
		s.Error(addDefaults(cluster))
	})
}

func (s *ClusterPostgresDataStoreTestSuite) TestSearchWithPostgres() {
	ctx := sac.WithAllAccess(context.Background())

	// Upsert cluster.
	c1ID, err := s.clusterDatastore.AddCluster(ctx, &storage.Cluster{
		Name:               "c1",
		Labels:             map[string]string{"env": "prod", "team": "team"},
		MainImage:          mainImage,
		CentralApiEndpoint: centralEndpoint,
	})
	s.NoError(err)

	// Upsert cluster.
	c2ID, err := s.clusterDatastore.AddCluster(ctx, &storage.Cluster{
		Name:               "c2",
		Labels:             map[string]string{"env": "test", "team": "team"},
		MainImage:          mainImage,
		CentralApiEndpoint: centralEndpoint,
	})
	s.NoError(err)

	ns1C1 := fixtures.GetNamespace(c1ID, "c1", "n1")
	ns2C1 := fixtures.GetNamespace(c1ID, "c1", "n2")
	ns1C2 := fixtures.GetNamespace(c2ID, "c2", "n1")

	// Upsert namespaces.
	s.NoError(s.nsDatastore.AddNamespace(ctx, ns1C1))
	s.NoError(s.nsDatastore.UpdateNamespace(ctx, ns2C1))
	s.NoError(s.nsDatastore.UpdateNamespace(ctx, ns1C2))

	for _, tc := range []struct {
		desc        string
		ctx         context.Context
		query       *v1.Query
		expectedIDs []string
		queryNs     bool
	}{
		{
			desc:  "Search clusters with empty query",
			ctx:   ctx,
			query: pkgSearch.EmptyQuery(),

			expectedIDs: []string{c1ID, c2ID},
		},
		{
			desc:  "Search clusters with cluster query",
			ctx:   ctx,
			query: pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.ClusterID, c1ID).ProtoQuery(),

			expectedIDs: []string{c1ID},
		},
		{
			desc:        "Search clusters with namespace query",
			ctx:         ctx,
			query:       pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.Namespace, "n2").ProtoQuery(),
			expectedIDs: []string{c1ID},
		},
		{
			desc:  "Search clusters with cluster+namespace query",
			ctx:   ctx,
			query: pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.Namespace, "n1").AddMapQuery(pkgSearch.ClusterLabel, "team", "team").ProtoQuery(),

			expectedIDs: []string{c1ID, c2ID},
		},
		{
			desc:  "Search clusters with cluster scope",
			ctx:   scoped.Context(ctx, scoped.Scope{ID: c1ID, Level: v1.SearchCategory_CLUSTERS}),
			query: pkgSearch.EmptyQuery(),

			expectedIDs: []string{c1ID},
		},
		{
			desc:  "Search clusters with cluster scope and in-scope cluster query",
			ctx:   scoped.Context(ctx, scoped.Scope{ID: c1ID, Level: v1.SearchCategory_CLUSTERS}),
			query: pkgSearch.NewQueryBuilder().AddMapQuery(pkgSearch.ClusterLabel, "env", "prod").ProtoQuery(),

			expectedIDs: []string{c1ID},
		},
		{
			desc:  "Search clusters with cluster scope and out-of-scope cluster query",
			ctx:   scoped.Context(ctx, scoped.Scope{ID: c1ID, Level: v1.SearchCategory_CLUSTERS}),
			query: pkgSearch.NewQueryBuilder().AddMapQuery(pkgSearch.ClusterLabel, "env", "test").ProtoQuery(),

			expectedIDs: []string{},
		},
		{
			desc:  "Search clusters with cluster scope and in-scope namespace query",
			ctx:   scoped.Context(ctx, scoped.Scope{ID: c1ID, Level: v1.SearchCategory_CLUSTERS}),
			query: pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.Namespace, "n1").ProtoQuery(),

			expectedIDs: []string{c1ID},
		},
		{
			desc:  "Search clusters with cluster scope and out-of-scope namespace query",
			ctx:   scoped.Context(ctx, scoped.Scope{ID: c2ID, Level: v1.SearchCategory_CLUSTERS}),
			query: pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.Namespace, "n2").ProtoQuery(),

			expectedIDs: []string{},
		},
		{
			desc:  "Search clusters with namespace scope",
			ctx:   scoped.Context(ctx, scoped.Scope{ID: ns1C1.Id, Level: v1.SearchCategory_NAMESPACES}),
			query: pkgSearch.EmptyQuery(),

			expectedIDs: []string{c1ID},
		},
		{
			desc:  "Search clusters with namespace scope and in-scope cluster query",
			ctx:   scoped.Context(ctx, scoped.Scope{ID: ns1C1.Id, Level: v1.SearchCategory_NAMESPACES}),
			query: pkgSearch.NewQueryBuilder().AddMapQuery(pkgSearch.ClusterLabel, "env", "prod").ProtoQuery(),

			expectedIDs: []string{c1ID},
		},
		{
			desc:  "Search clusters with namespace scope and out-of-scope cluster query",
			ctx:   scoped.Context(ctx, scoped.Scope{ID: ns1C1.Id, Level: v1.SearchCategory_NAMESPACES}),
			query: pkgSearch.NewQueryBuilder().AddMapQuery(pkgSearch.ClusterLabel, "env", "test").ProtoQuery(),

			expectedIDs: []string{},
		},

		{
			desc:  "Search namespaces with empty query",
			ctx:   ctx,
			query: pkgSearch.EmptyQuery(),

			expectedIDs: []string{ns1C1.Id, ns2C1.Id, ns1C2.Id},
			queryNs:     true,
		},
		{
			desc:        "Search namespaces with cluster query",
			ctx:         ctx,
			query:       pkgSearch.NewQueryBuilder().AddMapQuery(pkgSearch.ClusterLabel, "env", "test").ProtoQuery(),
			expectedIDs: []string{ns1C2.Id},
			queryNs:     true,
		},
		{
			desc:  "Search namespaces with cluster+namespace query",
			ctx:   ctx,
			query: pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.Namespace, "n1").AddMapQuery(pkgSearch.ClusterLabel, "team", "team").ProtoQuery(),

			expectedIDs: []string{ns1C1.Id, ns1C2.Id},
			queryNs:     true,
		},
		{
			desc:  "Search namespaces with cluster+namespace non-matching search fields",
			ctx:   ctx,
			query: pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.Namespace, "n1").AddMapQuery(pkgSearch.ClusterLabel, "team", "blah").ProtoQuery(),

			expectedIDs: []string{},
			queryNs:     true,
		},
		{
			desc:  "Search namespace with namespace scope",
			ctx:   scoped.Context(ctx, scoped.Scope{ID: ns1C1.Id, Level: v1.SearchCategory_NAMESPACES}),
			query: pkgSearch.EmptyQuery(),

			expectedIDs: []string{ns1C1.Id},
			queryNs:     true,
		},
		{
			desc:  "Search namespace with namespace scope and in-scope cluster query",
			ctx:   scoped.Context(ctx, scoped.Scope{ID: ns1C1.Id, Level: v1.SearchCategory_NAMESPACES}),
			query: pkgSearch.NewQueryBuilder().AddMapQuery(pkgSearch.ClusterLabel, "env", "prod").ProtoQuery(),

			expectedIDs: []string{ns1C1.Id},
			queryNs:     true,
		},
		{
			desc:  "Search namespace with namespace scope and out-of-scope cluster query",
			ctx:   scoped.Context(ctx, scoped.Scope{ID: ns1C1.Id, Level: v1.SearchCategory_NAMESPACES}),
			query: pkgSearch.NewQueryBuilder().AddMapQuery(pkgSearch.ClusterLabel, "env", "test").ProtoQuery(),

			expectedIDs: []string{},
			queryNs:     true,
		},
		{
			desc:  "Search namespace with namespace scope and in-scope namespace query",
			ctx:   scoped.Context(ctx, scoped.Scope{ID: ns1C1.Id, Level: v1.SearchCategory_NAMESPACES}),
			query: pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.Namespace, "n1").ProtoQuery(),

			expectedIDs: []string{ns1C1.Id},
			queryNs:     true,
		},
		{
			desc:  "Search namespace with namespace scope and out-of-scope namespace query",
			ctx:   scoped.Context(ctx, scoped.Scope{ID: ns1C1.Id, Level: v1.SearchCategory_NAMESPACES}),
			query: pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.Namespace, "n2").ProtoQuery(),

			expectedIDs: []string{},
			queryNs:     true,
		},
		{
			desc:  "Search namespaces with cluster scope",
			ctx:   scoped.Context(ctx, scoped.Scope{ID: c1ID, Level: v1.SearchCategory_CLUSTERS}),
			query: pkgSearch.EmptyQuery(),

			expectedIDs: []string{ns1C1.Id, ns2C1.Id},
			queryNs:     true,
		},
		{
			desc:  "Search namespaces with cluster scope and in-scope cluster query",
			ctx:   scoped.Context(ctx, scoped.Scope{ID: c1ID, Level: v1.SearchCategory_CLUSTERS}),
			query: pkgSearch.NewQueryBuilder().AddMapQuery(pkgSearch.ClusterLabel, "env", "prod").ProtoQuery(),

			expectedIDs: []string{ns1C1.Id, ns2C1.Id},
			queryNs:     true,
		},
		{
			desc:  "Search namespaces with cluster scope and out-of-scope cluster query",
			ctx:   scoped.Context(ctx, scoped.Scope{ID: c1ID, Level: v1.SearchCategory_CLUSTERS}),
			query: pkgSearch.NewQueryBuilder().AddMapQuery(pkgSearch.ClusterLabel, "env", "test").ProtoQuery(),

			expectedIDs: []string{},
			queryNs:     true,
		},
		{
			desc: "Search namespaces with cluster+namespace scope",
			ctx: scoped.Context(ctx,
				scoped.Scope{
					ID:    ns1C1.Id,
					Level: v1.SearchCategory_NAMESPACES,
					Parent: &scoped.Scope{
						ID:    c1ID,
						Level: v1.SearchCategory_CLUSTERS,
					},
				},
			),
			query: pkgSearch.NewQueryBuilder().AddMapQuery(pkgSearch.ClusterLabel, "env", "prod").ProtoQuery(),

			expectedIDs: []string{ns1C1.Id},
			queryNs:     true,
		},
		{
			desc: "Search namespaces with cluster+namespace scope and out-of-scope cluster query",
			ctx: scoped.Context(ctx,
				scoped.Scope{
					ID:    ns1C1.Id,
					Level: v1.SearchCategory_NAMESPACES,
					Parent: &scoped.Scope{
						ID:    c1ID,
						Level: v1.SearchCategory_CLUSTERS,
					},
				},
			),
			query: pkgSearch.NewQueryBuilder().AddMapQuery(pkgSearch.ClusterLabel, "env", "test").ProtoQuery(),

			expectedIDs: []string{},
			queryNs:     true,
		},
	} {
		s.T().Run(tc.desc, func(t *testing.T) {
			var actual []pkgSearch.Result
			var err error
			if tc.queryNs {
				actual, err = s.nsDatastore.Search(tc.ctx, tc.query)
			} else {
				actual, err = s.clusterDatastore.Search(tc.ctx, tc.query)
			}
			assert.NoError(t, err)
			assert.Len(t, actual, len(tc.expectedIDs))
			actualIDs := pkgSearch.ResultsToIDs(actual)
			assert.ElementsMatch(t, tc.expectedIDs, actualIDs)
		})
	}
}
