package testutils

import (
	"context"
	"testing"

	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	clusterCVEEdgeDataStore "github.com/stackrox/rox/central/clustercveedge/datastore"
	clusterCVEDataStore "github.com/stackrox/rox/central/cve/cluster/datastore"
	cveConverterV2 "github.com/stackrox/rox/central/cve/converter/v2"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	imageDataStore "github.com/stackrox/rox/central/image/datastore"
	namespaceDataStore "github.com/stackrox/rox/central/namespace/datastore"
	nodeDataStore "github.com/stackrox/rox/central/node/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/testconsts"
	"github.com/stackrox/rox/pkg/uuid"
)

// TestGraphDataStore provides the interface to a utility for connected datastores testing, with accessors to some internals,
// as well as test data injection and cleanup functions.
type TestGraphDataStore interface {
	// Expose internal for the case other datastores would be needed for testing purposes
	GetPostgresPool() postgres.DB
	// Internal accessor for test case generation
	GetStoredClusterIDs() []string
	GetStoredNodeIDs() []string
	// Data injection
	PushClusterToVulnerabilitiesGraph() error
	PushImageToVulnerabilitiesGraph() error
	PushNodeToVulnerabilitiesGraph() error
	// Data cleanup
	CleanClusterToVulnerabilitiesGraph() error
	CleanImageToVulnerabilitiesGraph() error
	CleanNodeToVulnerabilitiesGraph() error
	// Post test cleanup (TearDown)
	Cleanup(t *testing.T)
}

type testGraphDataStoreImpl struct {
	// Pool for postgres mode
	pgtestbase *pgtest.TestPostgres

	// DataStores
	namespaceStore      namespaceDataStore.DataStore
	deploymentStore     deploymentDataStore.DataStore
	imageStore          imageDataStore.DataStore
	nodeStore           nodeDataStore.DataStore
	clusterStore        clusterDataStore.DataStore
	clusterCVEEdgeStore clusterCVEEdgeDataStore.DataStore
	clusterCVEStore     clusterCVEDataStore.DataStore

	storedNodes           []string
	storedImages          []string
	storedDeployments     []string
	storedNamespaces      []string
	storedClusters        []string
	storedClusterCVEEdges []string
}

func embeddedVulnerabilityToClusterCVE(from *storage.EmbeddedVulnerability) *storage.ClusterCVE {
	ret := &storage.ClusterCVE{
		Id: from.GetCve(),
		CveBaseInfo: &storage.CVEInfo{
			Cve:          from.GetCve(),
			Summary:      from.GetSummary(),
			Link:         from.GetLink(),
			PublishedOn:  from.GetPublishedOn(),
			CreatedAt:    from.GetFirstSystemOccurrence(),
			LastModified: from.GetLastModified(),
			CvssV2:       from.GetCvssV2(),
			CvssV3:       from.GetCvssV3(),
		},
		Cvss:         from.GetCvss(),
		Severity:     from.GetSeverity(),
		Snoozed:      from.GetSuppressed(),
		SnoozeStart:  from.GetSuppressActivation(),
		SnoozeExpiry: from.GetSuppressExpiry(),
	}
	if ret.GetCveBaseInfo().GetCvssV3() != nil {
		ret.CveBaseInfo.ScoreVersion = storage.CVEInfo_V3
		ret.ImpactScore = from.GetCvssV3().GetImpactScore()
	} else if ret.GetCveBaseInfo().GetCvssV2() != nil {
		ret.CveBaseInfo.ScoreVersion = storage.CVEInfo_V2
		ret.ImpactScore = from.GetCvssV2().GetImpactScore()
	}
	return ret
}

func (s *testGraphDataStoreImpl) GetPostgresPool() postgres.DB {
	return s.pgtestbase.DB
}

func (s *testGraphDataStoreImpl) GetStoredClusterIDs() []string {
	return s.storedClusters
}

func (s *testGraphDataStoreImpl) GetStoredNodeIDs() []string {
	return s.storedNodes
}

// PushClusterToVulnerabilitiesGraph inserts the cluster -> CVE graph defined
// in the graph fixture (see the comment at the top of the cluster section for more details).
// The actual edges are declared in the function
func (s *testGraphDataStoreImpl) PushClusterToVulnerabilitiesGraph() (err error) {
	ctx := sac.WithAllAccess(context.Background())
	cluster1 := fixtures.GetCluster(testconsts.Cluster1)
	cluster2 := fixtures.GetCluster(testconsts.Cluster2)
	cluster1ID, err := s.clusterStore.AddCluster(ctx, cluster1)
	if err != nil {
		return err
	}
	s.storedClusters = append(s.storedClusters, cluster1ID)
	cluster1.Id = cluster1ID
	cluster2ID, err := s.clusterStore.AddCluster(ctx, cluster2)
	if err != nil {
		return err
	}
	s.storedClusters = append(s.storedClusters, cluster2ID)
	cluster2.Id = cluster2ID
	clusters1Only := []*storage.Cluster{cluster1}
	clusters2Only := []*storage.Cluster{cluster2}
	embeddedClusterCVE1 := fixtures.GetEmbeddedClusterCVE1234x0001()
	cve1FixVersion := embeddedClusterCVE1.GetFixedBy()
	embeddedClusterCVE2 := fixtures.GetEmbeddedClusterCVE4567x0002()
	cve2FixVersion := embeddedClusterCVE2.GetFixedBy()
	embeddedClusterCVE3 := fixtures.GetEmbeddedClusterCVE2345x0003()
	cve3FixVersion := embeddedClusterCVE3.GetFixedBy()

	clusterCVE1 := embeddedVulnerabilityToClusterCVE(embeddedClusterCVE1)
	clusterCVE2 := embeddedVulnerabilityToClusterCVE(embeddedClusterCVE2)
	clusterCVE3 := embeddedVulnerabilityToClusterCVE(embeddedClusterCVE3)
	clusterCVEParts1x1 := cveConverterV2.NewClusterCVEParts(clusterCVE1, clusters1Only, cve1FixVersion)
	clusterCVEParts1x2 := cveConverterV2.NewClusterCVEParts(clusterCVE2, clusters1Only, cve2FixVersion)
	clusterCVEParts2x2 := cveConverterV2.NewClusterCVEParts(clusterCVE2, clusters2Only, cve2FixVersion)
	clusterCVEParts2x3 := cveConverterV2.NewClusterCVEParts(clusterCVE3, clusters2Only, cve3FixVersion)
	err = s.clusterCVEStore.UpsertClusterCVEsInternal(ctx, storage.CVE_OPENSHIFT_CVE, clusterCVEParts1x1)
	if err != nil {
		return err
	}
	err = s.clusterCVEStore.UpsertClusterCVEsInternal(ctx, storage.CVE_OPENSHIFT_CVE, clusterCVEParts1x2)
	if err != nil {
		return err
	}
	err = s.clusterCVEStore.UpsertClusterCVEsInternal(ctx, storage.CVE_OPENSHIFT_CVE, clusterCVEParts2x2)
	if err != nil {
		return err
	}
	err = s.clusterCVEStore.UpsertClusterCVEsInternal(ctx, storage.CVE_OPENSHIFT_CVE, clusterCVEParts2x3)
	if err != nil {
		return err
	}
	return nil
}

// PushImageToVulnerabilitiesGraph inserts the namespace -> deployment -> image -> CVE graph defined
// in the graph fixture (see the comment at the top of the image section for more details).
// This function creates NamespaceA in Cluster1 and NamespaceB in Cluster2, then injects the SherlockHolmes
// and DoctorJekyll images, to finally bind them to their respective namespaces through the identically
// names deployments.
// Sherlock holmes is the deployment / image part from Cluster1 and NamespaceA.
// Dr Jekyll is the deployment / image part from Cluster2 and NamespaceB.
func (s *testGraphDataStoreImpl) PushImageToVulnerabilitiesGraph() (err error) {
	ctx := sac.WithAllAccess(context.Background())
	testNamespace1 := fixtures.GetNamespace(testconsts.Cluster1, testconsts.Cluster1, testconsts.NamespaceA)
	testNamespace2 := fixtures.GetNamespace(testconsts.Cluster2, testconsts.Cluster2, testconsts.NamespaceB)
	testImage1 := fixtures.GetImageSherlockHolmes1()
	testImage2 := fixtures.GetImageDoctorJekyll2()
	testDeployment1 := fixtures.GetDeploymentSherlockHolmes1(uuid.NewV4().String(), testNamespace1)
	testDeployment2 := fixtures.GetDeploymentDoctorJekyll2(uuid.NewV4().String(), testNamespace2)
	err = s.namespaceStore.AddNamespace(ctx, testNamespace1)
	if err != nil {
		return err
	}
	s.storedNamespaces = append(s.storedNamespaces, testNamespace1.GetId())
	err = s.namespaceStore.AddNamespace(ctx, testNamespace2)
	if err != nil {
		return err
	}
	s.storedNamespaces = append(s.storedNamespaces, testNamespace2.GetId())
	err = s.imageStore.UpsertImage(ctx, testImage1)
	if err != nil {
		return err
	}
	s.storedImages = append(s.storedImages, testImage1.GetId())
	err = s.imageStore.UpsertImage(ctx, testImage2)
	if err != nil {
		return err
	}
	s.storedImages = append(s.storedImages, testImage2.GetId())
	err = s.deploymentStore.UpsertDeployment(ctx, testDeployment1)
	if err != nil {
		return err
	}
	s.storedDeployments = append(s.storedDeployments, testDeployment1.GetId())
	err = s.deploymentStore.UpsertDeployment(ctx, testDeployment2)
	if err != nil {
		return err
	}
	s.storedDeployments = append(s.storedDeployments, testDeployment2.GetId())
	return nil
}

// PushNodeToVulnerabilitiesGraph inserts the node -> CVE graph defined
// in the graph fixture (see the comment at the top of the image section for more details).
// Sherlock holmes is the node part from Cluster1.
// Dr Jekyll is the node part from Cluster2.
func (s *testGraphDataStoreImpl) PushNodeToVulnerabilitiesGraph() (err error) {
	ctx := sac.WithAllAccess(context.Background())
	testNode1 := fixtures.GetScopedNode1(uuid.NewV4().String(), testconsts.Cluster1)
	testNode2 := fixtures.GetScopedNode2(uuid.NewV4().String(), testconsts.Cluster2)
	err = s.nodeStore.UpsertNode(ctx, testNode1)
	if err != nil {
		return err
	}
	s.storedNodes = append(s.storedNodes, testNode1.GetId())
	err = s.nodeStore.UpsertNode(ctx, testNode2)
	if err != nil {
		return err
	}
	s.storedNodes = append(s.storedNodes, testNode2.GetId())
	return nil
}

// CleanClusterToVulnerabilitiesGraph removes from database the data injected by PushClusterToVulnerabilitiesGraph.
func (s *testGraphDataStoreImpl) CleanClusterToVulnerabilitiesGraph() (err error) {
	ctx := sac.WithAllAccess(context.Background())
	storedClusters := s.storedClusters
	for _, clusterID := range storedClusters {
		err = s.clusterCVEStore.DeleteClusterCVEsInternal(ctx, clusterID)
		if err != nil {
			return err
		}
	}
	for _, clusterID := range storedClusters {
		deletionDoneSignal := concurrency.NewSignal()
		err = s.clusterStore.RemoveCluster(ctx, clusterID, &deletionDoneSignal)
		if err != nil {
			return err
		}
		<-deletionDoneSignal.Done()
	}
	s.storedClusters = s.storedClusters[:0]
	return nil
}

// CleanImageToVulnerabilitiesGraph removes from database the data injected by PushImageToVulnerabilitiesGraph.
func (s *testGraphDataStoreImpl) CleanImageToVulnerabilitiesGraph() (err error) {
	ctx := sac.WithAllAccess(context.Background())
	storedDeployments := s.storedDeployments
	for _, deploymentID := range storedDeployments {
		deployment, found, err := s.deploymentStore.GetDeployment(ctx, deploymentID)
		if err != nil {
			return err
		}
		if !found {
			continue
		}
		err = s.deploymentStore.RemoveDeployment(ctx, deployment.GetClusterId(), deploymentID)
		if err != nil {
			return err
		}
	}
	s.storedDeployments = s.storedDeployments[:0]
	storedImages := s.storedImages
	for _, imageID := range storedImages {
		err = s.imageStore.DeleteImages(ctx, imageID)
		if err != nil {
			return err
		}
	}
	s.storedImages = s.storedImages[:0]
	storedNamespaces := s.storedNamespaces
	for _, namespaceID := range storedNamespaces {
		err := s.namespaceStore.RemoveNamespace(ctx, namespaceID)
		if err != nil {
			return err
		}
	}
	s.storedNamespaces = s.storedNamespaces[:0]
	return nil
}

// CleanNodeToVulnerabilitiesGraph removes from database the data injected by PushNodeToVulnerabilitiesGraph.
func (s *testGraphDataStoreImpl) CleanNodeToVulnerabilitiesGraph() (err error) {
	ctx := sac.WithAllAccess(context.Background())
	storedNodes := s.storedNodes
	for _, nodeID := range storedNodes {
		err = s.nodeStore.DeleteNodes(ctx, nodeID)
		if err != nil {
			return err
		}
	}
	s.storedNodes = s.storedNodes[:0]
	return nil
}

func (s *testGraphDataStoreImpl) Cleanup(t *testing.T) {
	s.pgtestbase.Teardown(t)
}

// NewTestGraphDataStore provides a utility for storage testing, which contains a set of connected
// datastores, as well as a set of functions to inject and cleanup data.
func NewTestGraphDataStore(t *testing.T) (TestGraphDataStore, error) {
	var err error
	s := &testGraphDataStoreImpl{}

	s.pgtestbase = pgtest.ForT(t)
	s.nodeStore = nodeDataStore.GetTestPostgresDataStore(t, s.GetPostgresPool())
	s.imageStore = imageDataStore.GetTestPostgresDataStore(t, s.GetPostgresPool())
	s.deploymentStore, err = deploymentDataStore.GetTestPostgresDataStore(t, s.GetPostgresPool())
	if err != nil {
		return nil, err
	}
	s.namespaceStore, err = namespaceDataStore.GetTestPostgresDataStore(t, s.GetPostgresPool())
	if err != nil {
		return nil, err
	}
	s.clusterStore, err = clusterDataStore.GetTestPostgresDataStore(t, s.GetPostgresPool())
	if err != nil {
		return nil, err
	}
	s.clusterCVEStore, err = clusterCVEDataStore.GetTestPostgresDataStore(t, s.GetPostgresPool())
	if err != nil {
		return nil, err
	}
	s.storedDeployments = make([]string, 0)
	s.storedNamespaces = make([]string, 0)
	s.storedImages = make([]string, 0)
	s.storedNodes = make([]string, 0)
	s.storedClusters = make([]string, 0)
	s.storedClusterCVEEdges = make([]string, 0)

	return s, nil
}
