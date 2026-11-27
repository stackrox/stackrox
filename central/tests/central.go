//go:build sql_integration

package tests

import (
	"context"
	"fmt"
	"testing"
	"time"

	blobDS "github.com/stackrox/rox/central/blob/datastore"
	// Register all notifier types (email, slack, etc.) so CreateNotifier works.
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	clusterInitStore "github.com/stackrox/rox/central/clusterinit/store"
	checkResultsDS "github.com/stackrox/rox/central/complianceoperator/v2/checkresults/datastore"
	checkResultsService "github.com/stackrox/rox/central/complianceoperator/v2/checkresults/service"
	complianceManager "github.com/stackrox/rox/central/complianceoperator/v2/compliancemanager"
	compIntegrationDS "github.com/stackrox/rox/central/complianceoperator/v2/integration/datastore"
	complianceInfoPipeline "github.com/stackrox/rox/central/complianceoperator/v2/pipelines/complianceoperatorinfo"
	complianceProfilesPipeline "github.com/stackrox/rox/central/complianceoperator/v2/pipelines/complianceoperatorprofilesv2"
	complianceRemediationsPipeline "github.com/stackrox/rox/central/complianceoperator/v2/pipelines/complianceoperatorremediationsv2"
	complianceResultsPipeline "github.com/stackrox/rox/central/complianceoperator/v2/pipelines/complianceoperatorresultsv2"
	complianceRulesPipeline "github.com/stackrox/rox/central/complianceoperator/v2/pipelines/complianceoperatorrulesv2"
	complianceSSBPipeline "github.com/stackrox/rox/central/complianceoperator/v2/pipelines/complianceoperatorscansettingbindingsv2"
	complianceScansPipeline "github.com/stackrox/rox/central/complianceoperator/v2/pipelines/complianceoperatorscansv2"
	complianceSuitesPipeline "github.com/stackrox/rox/central/complianceoperator/v2/pipelines/complianceoperatorsuitesv2"
	profileDS "github.com/stackrox/rox/central/complianceoperator/v2/profiles/datastore"
	profileService "github.com/stackrox/rox/central/complianceoperator/v2/profiles/service"
	remediationDS "github.com/stackrox/rox/central/complianceoperator/v2/remediations/datastore"
	snapshotDS "github.com/stackrox/rox/central/complianceoperator/v2/report/datastore"
	reportManager "github.com/stackrox/rox/central/complianceoperator/v2/report/manager"
	reportGen "github.com/stackrox/rox/central/complianceoperator/v2/report/manager/generator"
	complianceRuleDS "github.com/stackrox/rox/central/complianceoperator/v2/rules/datastore"
	compScanConfigDS "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/datastore"
	scanConfigService "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/service"
	compScansDS "github.com/stackrox/rox/central/complianceoperator/v2/scans/datastore"
	compSSBDS "github.com/stackrox/rox/central/complianceoperator/v2/scansettingbindings/datastore"
	suiteDS "github.com/stackrox/rox/central/complianceoperator/v2/suites/datastore"
	delegatedRegistryConfigDS "github.com/stackrox/rox/central/delegatedregistryconfig/datastore"
	delegatedRegistryConfigStore "github.com/stackrox/rox/central/delegatedregistryconfig/store/postgres"
	"github.com/stackrox/rox/central/globaldb"
	hashDataStore "github.com/stackrox/rox/central/hash/datastore"
	hashManager "github.com/stackrox/rox/central/hash/manager"
	imageIntegrationDS "github.com/stackrox/rox/central/imageintegration/datastore"
	installationPostgres "github.com/stackrox/rox/central/installation/store/postgres"
	networkBaselineDS "github.com/stackrox/rox/central/networkbaseline/datastore"
	netEntityDS "github.com/stackrox/rox/central/networkgraph/entity/datastore"
	notifierDS "github.com/stackrox/rox/central/notifier/datastore"
	"github.com/stackrox/rox/central/notifier/policycleaner"
	notifierProcessor "github.com/stackrox/rox/central/notifier/processor"
	notifierService "github.com/stackrox/rox/central/notifier/service"
	_ "github.com/stackrox/rox/central/notifiers/all"
	policyDS "github.com/stackrox/rox/central/policy/datastore"
	policyStoreWrapper "github.com/stackrox/rox/central/policy/store"
	policyCategoryDS "github.com/stackrox/rox/central/policycategory/datastore"
	policyCategoryStore "github.com/stackrox/rox/central/policycategory/store/postgres"
	policyCategoryEdgeDS "github.com/stackrox/rox/central/policycategoryedge/datastore"
	policyCategoryEdgeStore "github.com/stackrox/rox/central/policycategoryedge/store/postgres"
	processBaselineDS "github.com/stackrox/rox/central/processbaseline/datastore"
	sensorService "github.com/stackrox/rox/central/sensor/service"
	"github.com/stackrox/rox/central/sensor/service/connection"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	pipelineAll "github.com/stackrox/rox/central/sensor/service/pipeline/all"
	v1 "github.com/stackrox/rox/generated/api/v1"
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/integrationhealth"
	mtlsTestutils "github.com/stackrox/rox/pkg/mtls/testutils"
	"github.com/stackrox/rox/pkg/notifier"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	pgConn "github.com/stackrox/rox/pkg/postgres/pgtest/conn"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"
)

const testClusterIDKey = "x-test-cluster-id"

// noopHealthReporter implements integrationhealth.Reporter with no-op methods.
type noopHealthReporter struct{}

func (n *noopHealthReporter) Register(string, string, storage.IntegrationHealth_Type) error {
	return nil
}
func (n *noopHealthReporter) UpdateIntegrationHealthAsync(*storage.IntegrationHealth) {}
func (n *noopHealthReporter) RetrieveIntegrationHealths(storage.IntegrationHealth_Type) ([]*storage.IntegrationHealth, error) {
	return nil, nil
}
func (n *noopHealthReporter) RemoveIntegrationHealth(string) error { return nil }

var _ integrationhealth.Reporter = (*noopHealthReporter)(nil)

// CentralTestServer wraps Central-side components for integration testing.
type CentralTestServer struct {
	t        *testing.T
	listener *bufconn.Listener
	server   *grpc.Server
	hashMgr  hashManager.Manager
	connMgr  connection.Manager

	ClusterDS      clusterDataStore.DataStore
	ProfileDS      profileDS.DataStore
	CheckResultsDS checkResultsDS.DataStore
	ScansDS        compScansDS.DataStore
	ScanConfigDS   compScanConfigDS.DataStore
	RemediationDS  remediationDS.DataStore
	RuleDS         complianceRuleDS.DataStore
	SnapshotDS     snapshotDS.DataStore
	BlobDS         blobDS.Datastore
	ReportMgr      reportManager.Manager
}

// NewCentral constructs a CentralTestServer.
func NewCentral(t *testing.T) *CentralTestServer {
	tp := pgtest.ForT(t)

	// pgtest.ForT creates a pool with pool_max_conns=1. The report aggregator
	// uses cursor queries that hold a connection while doing nested lookups,
	// which deadlocks with a single connection. Create a larger pool.
	dbName := tp.DB.Config().Config.ConnConfig.Database
	connStr := fmt.Sprintf("%s pool_max_conns=20", pgConn.GetConnectionStringWithDatabaseName(t, dbName))
	pool, err := postgres.Connect(context.Background(), connStr)
	require.NoError(t, err)
	t.Cleanup(func() { pool.Close() })

	globaldb.SetPostgresTest(t, pool)

	hashDS := hashDataStore.GetTestPostgresDataStore(t, pool)
	hashMgr := hashManager.NewManager(hashDS)

	clusterDS, err := clusterDataStore.GetTestPostgresDataStore(t, pool)
	require.NoError(t, err)

	netEntityStore := netEntityDS.GetTestPostgresDataStore(t, pool)
	processBaselineStore := processBaselineDS.GetTestPostgresDataStore(t, pool)
	networkBaselineStore, err := networkBaselineDS.GetTestPostgresDataStore(t, pool)
	require.NoError(t, err)
	imageIntegrationStore := imageIntegrationDS.GetTestPostgresDataStore(t, pool)
	delegatedRegistryStore := delegatedRegistryConfigDS.New(delegatedRegistryConfigStore.New(pool))

	categoryEdgeStore := policyCategoryEdgeDS.New(policyCategoryEdgeStore.New(pool))
	categoryStore := policyCategoryDS.New(policyCategoryStore.New(pool), categoryEdgeStore)
	notifierStore := notifierDS.GetTestPostgresDataStore(t, pool)
	policyStore := policyDS.New(policyStoreWrapper.New(pool), clusterDS, notifierStore, categoryStore)

	// Compliance v2 datastores.
	integrationDS := compIntegrationDS.GetTestPostgresDataStore(t, pool)
	scanConfigDS := compScanConfigDS.GetTestPostgresDataStore(t, pool)
	ssbDS := compSSBDS.GetTestPostgresDataStore(t, pool)
	profDS := profileDS.GetTestPostgresDataStore(t, pool)
	scansDS := compScansDS.GetTestPostgresDataStore(t, pool)
	resultsDS := checkResultsDS.GetTestPostgresDataStore(t, pool)
	ruleDS := complianceRuleDS.GetTestPostgresDataStore(t, pool)
	remDS := remediationDS.GetTestPostgresDataStore(t, pool)
	suitesDS := suiteDS.GetTestPostgresDataStore(t, pool)
	snapDS := snapshotDS.GetTestPostgresDataStore(t, pool)
	blobStore := blobDS.NewTestDatastore(t, pool)

	connMgr := connection.NewManager(hashMgr)

	compMgr := complianceManager.New(connMgr, integrationDS, scanConfigDS, ssbDS, clusterDS, profDS, scansDS, resultsDS)

	require.NoError(t, connMgr.Start(
		clusterDS,
		netEntityStore,
		policyStore,
		processBaselineStore,
		networkBaselineStore,
		delegatedRegistryStore,
		imageIntegrationStore,
		compMgr,
		&concurrency.Flag{},
	))

	require.NoError(t, mtlsTestutils.LoadTestMTLSCerts(t))

	// Notifier processor (real — needed for email delivery).
	healthReporter := &noopHealthReporter{}
	notifierSet := notifier.NewNotifierSet(0)
	notifProc := notifierProcessor.New(notifierSet, healthReporter)

	// Report generator and manager (real).
	repGen := reportGen.New(resultsDS, notifProc, profDS, remDS, scansDS, ruleDS, snapDS, blobStore)
	repMgr := reportManager.New(scanConfigDS, scansDS, profDS, snapDS, integrationDS, suitesDS, ssbDS, resultsDS, repGen)
	repMgr.Start()
	t.Cleanup(func() { repMgr.Stop() })

	// Pipeline factory with all compliance v2 fragments.
	fragments := []pipeline.Fragment{
		complianceProfilesPipeline.NewPipeline(profDS),
		complianceRulesPipeline.NewPipeline(ruleDS),
		complianceResultsPipeline.NewPipeline(resultsDS, clusterDS, repMgr),
		complianceScansPipeline.NewPipeline(scansDS, repMgr),
		complianceSSBPipeline.NewPipeline(ssbDS),
		complianceSuitesPipeline.NewPipeline(suitesDS),
		complianceRemediationsPipeline.NewPipeline(remDS),
		complianceInfoPipeline.NewPipeline(compMgr),
	}
	factory := &testFactory{hashMgr: hashMgr, fragments: fragments}

	// Sensor service.
	installStore := installationPostgres.New(pool)
	clusterInit := clusterInitStore.GetTestPostgresDataStore(t, pool)
	svc := sensorService.New(connMgr, factory, clusterDS, installStore, clusterInit)

	// API services.
	profileSvc := profileService.New(profDS)
	resultsSvc := checkResultsService.New(resultsDS, scanConfigDS, integrationDS, profDS, ruleDS, scansDS)
	notifierSvc := notifierService.New(notifierStore, notifProc, policycleaner.PolicyCleaner{}, healthReporter, "", pool)
	scanConfigSvc := scanConfigService.New(scanConfigDS, ssbDS, suitesDS, compMgr, repMgr, notifierStore, profDS, clusterDS, snapDS, blobStore)

	// gRPC server.
	lis := bufconn.Listen(1024 * 1024)
	srv := grpc.NewServer(
		grpc.StreamInterceptor(sensorIdentityInterceptor()),
		grpc.UnaryInterceptor(allAccessUnaryInterceptor()),
	)
	central.RegisterSensorServiceServer(srv, svc)
	v2.RegisterComplianceProfileServiceServer(srv, profileSvc)
	v2.RegisterComplianceResultsServiceServer(srv, resultsSvc)
	v1.RegisterNotifierServiceServer(srv, notifierSvc)
	v2.RegisterComplianceScanConfigurationServiceServer(srv, scanConfigSvc)

	return &CentralTestServer{
		t:              t,
		listener:       lis,
		server:         srv,
		hashMgr:        hashMgr,
		connMgr:        connMgr,
		ClusterDS:      clusterDS,
		ProfileDS:      profDS,
		CheckResultsDS: resultsDS,
		ScansDS:        scansDS,
		ScanConfigDS:   scanConfigDS,
		RemediationDS:  remDS,
		RuleDS:         ruleDS,
		SnapshotDS:     snapDS,
		BlobDS:         blobStore,
		ReportMgr:      repMgr,
	}
}

// Start begins serving gRPC on the in-memory listener.
func (c *CentralTestServer) Start() {
	go func() {
		if err := c.server.Serve(c.listener); err != nil {
			c.t.Logf("gRPC server stopped: %v", err)
		}
	}()
	c.t.Cleanup(func() {
		c.server.GracefulStop()
	})
}

// APIConn returns a gRPC client connection to the test Central, suitable for
// creating API service clients. The connection is closed on test cleanup.
func (c *CentralTestServer) APIConn() *grpc.ClientConn {
	conn, err := grpc.NewClient("passthrough:///bufconn",
		grpc.WithContextDialer(bufconnDialer(c.listener)),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(c.t, err)
	c.t.Cleanup(func() { conn.Close() })
	return conn
}

// testFactory is a pipeline.Factory that builds a ClusterPipeline with only
// the supplied fragments.
type testFactory struct {
	hashMgr   hashManager.Manager
	fragments []pipeline.Fragment
}

func (f *testFactory) PipelineForCluster(ctx context.Context, clusterID string) (pipeline.ClusterPipeline, error) {
	deduper := f.hashMgr.GetDeduper(ctx, clusterID)
	return pipelineAll.NewClusterPipeline(clusterID, deduper, f.fragments...), nil
}

// sensorIdentityInterceptor returns a gRPC stream interceptor that injects a
// fake sensor identity into the context.
func sensorIdentityInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		clusterID := ""
		if md, ok := metadata.FromIncomingContext(ss.Context()); ok {
			if vals := md.Get(testClusterIDKey); len(vals) > 0 {
				clusterID = vals[0]
			}
		}
		identity := &testSensorIdentity{clusterID: clusterID}
		ctx := sac.WithAllAccess(ss.Context())
		ctx = authn.ContextWithIdentity(ctx, identity, nil)
		return handler(srv, &wrappedStream{ServerStream: ss, ctx: ctx})
	}
}

// allAccessUnaryInterceptor injects SAC all-access and a fake user identity
// into unary RPCs so API service methods pass auth checks.
func allAccessUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		ctx = sac.WithAllAccess(ctx)
		ctx = authn.ContextWithIdentity(ctx, &testUserIdentity{}, nil)
		return handler(ctx, req)
	}
}

type wrappedStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedStream) Context() context.Context { return w.ctx }

// testSensorIdentity implements authn.Identity for sensor streams.
type testSensorIdentity struct {
	clusterID string
}

func (i *testSensorIdentity) UID() string                                  { return "sensor-" + i.clusterID }
func (i *testSensorIdentity) FriendlyName() string                         { return "Sensor " + i.clusterID }
func (i *testSensorIdentity) FullName() string                             { return "" }
func (i *testSensorIdentity) User() *storage.UserInfo                      { return nil }
func (i *testSensorIdentity) Permissions() map[string]storage.Access       { return nil }
func (i *testSensorIdentity) Roles() []permissions.ResolvedRole            { return nil }
func (i *testSensorIdentity) Attributes() map[string][]string              { return nil }
func (i *testSensorIdentity) ExternalAuthProvider() authproviders.Provider { return nil }
func (i *testSensorIdentity) ValidityPeriod() (time.Time, time.Time) {
	return time.Now().Add(-time.Hour), time.Now().Add(24 * time.Hour)
}
func (i *testSensorIdentity) Service() *storage.ServiceIdentity {
	return &storage.ServiceIdentity{
		Type: storage.ServiceType_SENSOR_SERVICE,
		Id:   i.clusterID,
	}
}

// testUserIdentity implements authn.Identity for API calls.
type testUserIdentity struct{}

func (i *testUserIdentity) UID() string                                  { return "test-user" }
func (i *testUserIdentity) FriendlyName() string                         { return "Test User" }
func (i *testUserIdentity) FullName() string                             { return "Test User" }
func (i *testUserIdentity) User() *storage.UserInfo                      { return nil }
func (i *testUserIdentity) Permissions() map[string]storage.Access       { return nil }
func (i *testUserIdentity) Roles() []permissions.ResolvedRole            { return nil }
func (i *testUserIdentity) Attributes() map[string][]string              { return nil }
func (i *testUserIdentity) ExternalAuthProvider() authproviders.Provider { return nil }
func (i *testUserIdentity) ValidityPeriod() (time.Time, time.Time) {
	return time.Now().Add(-time.Hour), time.Now().Add(24 * time.Hour)
}
func (i *testUserIdentity) Service() *storage.ServiceIdentity { return nil }
