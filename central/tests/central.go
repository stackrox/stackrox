//go:build sql_integration

package tests

import (
	"context"
	"testing"
	"time"

	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	clusterInitStore "github.com/stackrox/rox/central/clusterinit/store"
	checkResultsDS "github.com/stackrox/rox/central/complianceoperator/v2/checkresults/datastore"
	complianceManager "github.com/stackrox/rox/central/complianceoperator/v2/compliancemanager"
	compIntegrationDS "github.com/stackrox/rox/central/complianceoperator/v2/integration/datastore"
	complianceProfilesPipeline "github.com/stackrox/rox/central/complianceoperator/v2/pipelines/complianceoperatorprofilesv2"
	profileDS "github.com/stackrox/rox/central/complianceoperator/v2/profiles/datastore"
	compScanConfigDS "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/datastore"
	compScansDS "github.com/stackrox/rox/central/complianceoperator/v2/scans/datastore"
	compSSBDS "github.com/stackrox/rox/central/complianceoperator/v2/scansettingbindings/datastore"
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
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/grpc/authn"
	mtlsTestutils "github.com/stackrox/rox/pkg/mtls/testutils"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"
)

const testClusterIDKey = "x-test-cluster-id"

// CentralTestServer wraps Central-side components for integration testing.
type CentralTestServer struct {
	t        *testing.T
	listener *bufconn.Listener
	server   *grpc.Server
	hashMgr  hashManager.Manager
	connMgr  connection.Manager

	ClusterDS clusterDataStore.DataStore
	ProfileDS profileDS.DataStore
}

// NewCentral constructs a CentralTestServer.
func NewCentral(t *testing.T) *CentralTestServer {
	tp := pgtest.ForT(t)
	pool := tp.DB

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

	// Policy datastore (real).
	categoryEdgeStore := policyCategoryEdgeDS.New(policyCategoryEdgeStore.New(pool))
	categoryStore := policyCategoryDS.New(policyCategoryStore.New(pool), categoryEdgeStore)
	notifierStore := notifierDS.GetTestPostgresDataStore(t, pool)
	policyStore := policyDS.New(policyStoreWrapper.New(pool), clusterDS, notifierStore, categoryStore)

	// Compliance v2 datastores (real).
	integrationDS := compIntegrationDS.GetTestPostgresDataStore(t, pool)
	scanConfigDS := compScanConfigDS.GetTestPostgresDataStore(t, pool)
	ssbDS := compSSBDS.GetTestPostgresDataStore(t, pool)
	profDS := profileDS.GetTestPostgresDataStore(t, pool)
	scansDS := compScansDS.GetTestPostgresDataStore(t, pool)
	resultsDS := checkResultsDS.GetTestPostgresDataStore(t, pool)

	connMgr := connection.NewManager(hashMgr)

	// Compliance manager (real) — needs connMgr reference but doesn't call it during construction.
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

	// Pipeline factory with compliance v2 fragments.
	fragments := []pipeline.Fragment{
		complianceProfilesPipeline.NewPipeline(profDS),
	}
	factory := &testFactory{hashMgr: hashMgr, fragments: fragments}

	// Sensor service (real).
	installStore := installationPostgres.New(pool)
	clusterInit := clusterInitStore.GetTestPostgresDataStore(t, pool)
	svc := sensorService.New(connMgr, factory, clusterDS, installStore, clusterInit)

	// gRPC server with auth bypass interceptor.
	lis := bufconn.Listen(1024 * 1024)
	srv := grpc.NewServer(grpc.StreamInterceptor(sensorIdentityInterceptor()))
	central.RegisterSensorServiceServer(srv, svc)

	return &CentralTestServer{
		t:         t,
		listener:  lis,
		server:    srv,
		hashMgr:   hashMgr,
		connMgr:   connMgr,
		ClusterDS: clusterDS,
		ProfileDS: profDS,
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
// fake sensor identity into the context. The cluster ID is read from incoming
// metadata (testClusterIDKey), allowing multiple FakeSensors with different
// cluster identities.
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

type wrappedStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedStream) Context() context.Context { return w.ctx }

// testSensorIdentity implements authn.Identity for the auth bypass interceptor.
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

func allAccessCtx() context.Context {
	return sac.WithAllAccess(context.Background())
}
