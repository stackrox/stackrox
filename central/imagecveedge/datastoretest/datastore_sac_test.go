package datastoretest

import (
	"context"
	"encoding/base64"
	"strings"
	"testing"

	dackboxTestUtils "github.com/stackrox/rox/central/dackbox/testutils"
	genericCVEEdgeDataStore "github.com/stackrox/rox/central/imagecveedge/datastore"
	imageCVEEdgeDataStore "github.com/stackrox/rox/central/imagecveedge/datastore"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/fixtures"
	sacTestUtils "github.com/stackrox/rox/pkg/sac/testutils"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/suite"
)

func TestCVEDataStoreSAC(t *testing.T) {
	suite.Run(t, new(imageCveEdgeDataStoreSACTestSuite))
}

type imageCveEdgeDataStoreSACTestSuite struct {
	suite.Suite

	dackboxTestStore  dackboxTestUtils.DackboxTestDataStore
	imageCVEEdgeStore imageCVEEdgeDataStore.DataStore

	imageTestContexts map[string]context.Context
}

func (s *imageCveEdgeDataStoreSACTestSuite) SetupSuite() {
	var err error
	s.dackboxTestStore, err = dackboxTestUtils.NewDackboxTestDataStore(s.T())
	s.Require().NoError(err)
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		pool := s.dackboxTestStore.GetPostgresPool()
		s.imageCVEEdgeStore = imageCVEEdgeDataStore.GetTestPostgresDataStore(s.T(), pool)
	} else {
		dacky := s.dackboxTestStore.GetDackbox()
		keyFence := s.dackboxTestStore.GetKeyFence()
		bleveIndex := s.dackboxTestStore.GetBleveIndex()
		s.imageCVEEdgeStore = genericCVEEdgeDataStore.GetTestRocksBleveDataStore(s.T(), bleveIndex, dacky, keyFence)
	}
	s.imageTestContexts = sacTestUtils.GetNamespaceScopedTestContexts(context.Background(), s.T(), resources.Image)
}

func (s *imageCveEdgeDataStoreSACTestSuite) TearDownSuite() {
	s.Require().NoError(s.dackboxTestStore.Cleanup(s.T()))
}

// getImageCVEEdgeID returns base 64 encoded Image:CVE ids
func getImageCVEEdgeID(image, cve string) string {
	e := func(s string) string {
		return base64.RawStdEncoding.EncodeToString([]byte(s))
	}
	return e(image) + ":" + e(cve)
}

// getImageCVEEdgeID returns base 64 encoded Image:CVE ids
func id2pair(s string) imageCvePair {
	split := strings.Split(s, ":")
	if len(split) != 2 {
		return imageCvePair{}
	}
	d := func(s string) string {
		decodeString, err := base64.RawStdEncoding.DecodeString(s)
		if err != nil {
			panic(err)
		}
		return string(decodeString)
	}
	return imageCvePair{
		image: d(split[0]),
		cve:   d(split[1]),
	}
}

func (s *imageCveEdgeDataStoreSACTestSuite) cleanImageToVulnerabilitiesGraph(waitForIndexing bool) {
	s.Require().NoError(s.dackboxTestStore.CleanImageToVulnerabilitiesGraph(waitForIndexing))
}

func (s *imageCveEdgeDataStoreSACTestSuite) cleanNodeToVulnerabilitiesGraph(waitForIndexing bool) {
	s.Require().NoError(s.dackboxTestStore.CleanNodeToVulnerabilitiesGraph(waitForIndexing))
}

type imageCvePair struct {
	image, cve string
}

func (ic imageCvePair) getImageCVEEdgeID() string {
	return getImageCVEEdgeID(ic.image, ic.cve)
}

type cveTestCase struct {
	contextKey       string
	expectedCVEFound map[imageCvePair]bool
}

var (
	img1              = fixtures.GetImageSherlockHolmes1()
	img2              = fixtures.GetImageDoctorJekyll2()
	imageCVETestCases = []cveTestCase{
		{
			contextKey: sacTestUtils.UnrestrictedReadCtx,
			expectedCVEFound: map[imageCvePair]bool{
				imageCvePair{image: img1.GetId(), cve: "CVE-1234-0001"}: true,
				imageCvePair{image: img1.GetId(), cve: "CVE-4567-0002"}: true,
				imageCvePair{image: img1.GetId(), cve: "CVE-1234-0003"}: true,
				imageCvePair{image: img1.GetId(), cve: "CVE-3456-0004"}: true,
				imageCvePair{image: img1.GetId(), cve: "CVE-3456-0005"}: true,
				imageCvePair{image: img2.GetId(), cve: "CVE-4567-0002"}: true,
				imageCvePair{image: img2.GetId(), cve: "CVE-3456-0004"}: true,
				imageCvePair{image: img2.GetId(), cve: "CVE-3456-0005"}: true,
				imageCvePair{image: img2.GetId(), cve: "CVE-2345-0006"}: true,
				imageCvePair{image: img2.GetId(), cve: "CVE-2345-0007"}: true,
			},
		},
		{
			contextKey: sacTestUtils.Cluster1ReadWriteCtx,
			expectedCVEFound: map[imageCvePair]bool{
				imageCvePair{image: img1.GetId(), cve: "CVE-1234-0001"}: true,
				imageCvePair{image: img1.GetId(), cve: "CVE-4567-0002"}: true,
				imageCvePair{image: img1.GetId(), cve: "CVE-1234-0003"}: true,
				imageCvePair{image: img1.GetId(), cve: "CVE-3456-0004"}: true,
				imageCvePair{image: img1.GetId(), cve: "CVE-3456-0005"}: true,
			},
		},
		{
			contextKey: sacTestUtils.Cluster1NamespaceAReadWriteCtx,
			expectedCVEFound: map[imageCvePair]bool{
				imageCvePair{image: img1.GetId(), cve: "CVE-1234-0001"}: true,
				imageCvePair{image: img1.GetId(), cve: "CVE-4567-0002"}: true,
				imageCvePair{image: img1.GetId(), cve: "CVE-1234-0003"}: true,
				imageCvePair{image: img1.GetId(), cve: "CVE-3456-0004"}: true,
				imageCvePair{image: img1.GetId(), cve: "CVE-3456-0005"}: true,
				imageCvePair{image: img1.GetId(), cve: "CVE-2345-0006"}: false,
				imageCvePair{image: img1.GetId(), cve: "CVE-2345-0007"}: false,
			},
		},
		{
			contextKey: sacTestUtils.Cluster1NamespaceBReadWriteCtx,
			expectedCVEFound: map[imageCvePair]bool{
				imageCvePair{image: img1.GetId(), cve: "CVE-1234-0001"}: false,
				imageCvePair{image: img1.GetId(), cve: "CVE-4567-0002"}: false,
				imageCvePair{image: img1.GetId(), cve: "CVE-1234-0003"}: false,
				imageCvePair{image: img1.GetId(), cve: "CVE-3456-0004"}: false,
				imageCvePair{image: img1.GetId(), cve: "CVE-3456-0005"}: false,
				imageCvePair{image: img1.GetId(), cve: "CVE-2345-0006"}: false,
				imageCvePair{image: img1.GetId(), cve: "CVE-2345-0007"}: false,
			},
		},
		{
			contextKey: sacTestUtils.Cluster1NamespacesABReadWriteCtx,
			expectedCVEFound: map[imageCvePair]bool{
				imageCvePair{image: img1.GetId(), cve: "CVE-1234-0001"}: true,
				imageCvePair{image: img1.GetId(), cve: "CVE-4567-0002"}: true,
				imageCvePair{image: img1.GetId(), cve: "CVE-1234-0003"}: true,
				imageCvePair{image: img1.GetId(), cve: "CVE-3456-0004"}: true,
				imageCvePair{image: img1.GetId(), cve: "CVE-3456-0005"}: true,
				imageCvePair{image: img1.GetId(), cve: "CVE-2345-0006"}: false,
				imageCvePair{image: img1.GetId(), cve: "CVE-2345-0007"}: false,
			},
		},
		{
			contextKey: sacTestUtils.Cluster1NamespacesBCReadWriteCtx,
			expectedCVEFound: map[imageCvePair]bool{
				imageCvePair{image: img1.GetId(), cve: "CVE-1234-0001"}: false,
				imageCvePair{image: img1.GetId(), cve: "CVE-4567-0002"}: false,
				imageCvePair{image: img1.GetId(), cve: "CVE-1234-0003"}: false,
				imageCvePair{image: img1.GetId(), cve: "CVE-3456-0004"}: false,
				imageCvePair{image: img1.GetId(), cve: "CVE-3456-0005"}: false,
				imageCvePair{image: img1.GetId(), cve: "CVE-2345-0006"}: false,
				imageCvePair{image: img1.GetId(), cve: "CVE-2345-0007"}: false,
			},
		},
		{
			contextKey: sacTestUtils.Cluster2ReadWriteCtx,
			expectedCVEFound: map[imageCvePair]bool{
				imageCvePair{image: img2.GetId(), cve: "CVE-4567-0002"}: true,
				imageCvePair{image: img2.GetId(), cve: "CVE-3456-0004"}: true,
				imageCvePair{image: img2.GetId(), cve: "CVE-3456-0005"}: true,
				imageCvePair{image: img2.GetId(), cve: "CVE-2345-0006"}: true,
				imageCvePair{image: img2.GetId(), cve: "CVE-2345-0007"}: true,
			},
		},
		{
			contextKey: sacTestUtils.Cluster2NamespaceAReadWriteCtx,
			expectedCVEFound: map[imageCvePair]bool{
				imageCvePair{image: img2.GetId(), cve: "CVE-4567-0002"}: true,
				imageCvePair{image: img2.GetId(), cve: "CVE-3456-0004"}: true,
				imageCvePair{image: img2.GetId(), cve: "CVE-3456-0005"}: true,
				imageCvePair{image: img2.GetId(), cve: "CVE-2345-0006"}: true,
				imageCvePair{image: img2.GetId(), cve: "CVE-2345-0007"}: true,
			},
		},
		{
			contextKey: sacTestUtils.Cluster2NamespaceBReadWriteCtx,
			expectedCVEFound: map[imageCvePair]bool{
				imageCvePair{image: img2.GetId(), cve: "CVE-4567-0002"}: true,
				imageCvePair{image: img2.GetId(), cve: "CVE-3456-0004"}: true,
				imageCvePair{image: img2.GetId(), cve: "CVE-3456-0005"}: true,
				imageCvePair{image: img2.GetId(), cve: "CVE-2345-0006"}: true,
				imageCvePair{image: img2.GetId(), cve: "CVE-2345-0007"}: true,
			},
		},
		{
			contextKey: sacTestUtils.Cluster2NamespacesACReadWriteCtx,
			expectedCVEFound: map[imageCvePair]bool{
				imageCvePair{image: img2.GetId(), cve: "CVE-4567-0002"}: true,
				imageCvePair{image: img2.GetId(), cve: "CVE-3456-0004"}: true,
				imageCvePair{image: img2.GetId(), cve: "CVE-3456-0005"}: true,
				imageCvePair{image: img2.GetId(), cve: "CVE-2345-0006"}: true,
				imageCvePair{image: img2.GetId(), cve: "CVE-2345-0007"}: true,
			},
		},
		{
			contextKey: sacTestUtils.Cluster2NamespacesBCReadWriteCtx,
			expectedCVEFound: map[imageCvePair]bool{
				imageCvePair{image: img2.GetId(), cve: "CVE-4567-0002"}: true,
				imageCvePair{image: img2.GetId(), cve: "CVE-3456-0004"}: true,
				imageCvePair{image: img2.GetId(), cve: "CVE-3456-0005"}: true,
				imageCvePair{image: img2.GetId(), cve: "CVE-2345-0006"}: true,
				imageCvePair{image: img2.GetId(), cve: "CVE-2345-0007"}: true,
			},
		},
		{
			contextKey: sacTestUtils.Cluster3ReadWriteCtx,
			expectedCVEFound: map[imageCvePair]bool{
				imageCvePair{image: img1.GetId(), cve: "CVE-1234-0001"}: false,
				imageCvePair{image: img1.GetId(), cve: "CVE-4567-0002"}: false,
				imageCvePair{image: img1.GetId(), cve: "CVE-1234-0003"}: false,
				imageCvePair{image: img1.GetId(), cve: "CVE-3456-0004"}: false,
				imageCvePair{image: img1.GetId(), cve: "CVE-3456-0005"}: false,
				imageCvePair{image: img1.GetId(), cve: "CVE-2345-0006"}: false,
				imageCvePair{image: img1.GetId(), cve: "CVE-2345-0007"}: false,
			},
		},
		{
			contextKey: sacTestUtils.Cluster3NamespaceAReadWriteCtx,
			expectedCVEFound: map[imageCvePair]bool{
				imageCvePair{image: img1.GetId(), cve: "CVE-1234-0001"}: false,
				imageCvePair{image: img1.GetId(), cve: "CVE-4567-0002"}: false,
				imageCvePair{image: img1.GetId(), cve: "CVE-1234-0003"}: false,
				imageCvePair{image: img1.GetId(), cve: "CVE-3456-0004"}: false,
				imageCvePair{image: img1.GetId(), cve: "CVE-3456-0005"}: false,
				imageCvePair{image: img1.GetId(), cve: "CVE-2345-0006"}: false,
				imageCvePair{image: img1.GetId(), cve: "CVE-2345-0007"}: false,
			},
		},
		{
			contextKey: sacTestUtils.Cluster3NamespaceBReadWriteCtx,
			expectedCVEFound: map[imageCvePair]bool{
				imageCvePair{image: img1.GetId(), cve: "CVE-1234-0001"}: false,
				imageCvePair{image: img1.GetId(), cve: "CVE-4567-0002"}: false,
				imageCvePair{image: img1.GetId(), cve: "CVE-1234-0003"}: false,
				imageCvePair{image: img1.GetId(), cve: "CVE-3456-0004"}: false,
				imageCvePair{image: img1.GetId(), cve: "CVE-3456-0005"}: false,
				imageCvePair{image: img1.GetId(), cve: "CVE-2345-0006"}: false,
				imageCvePair{image: img1.GetId(), cve: "CVE-2345-0007"}: false,
			},
		},
		{
			contextKey: sacTestUtils.Cluster3NamespacesABReadWriteCtx,
			expectedCVEFound: map[imageCvePair]bool{
				imageCvePair{image: img1.GetId(), cve: "CVE-1234-0001"}: false,
				imageCvePair{image: img1.GetId(), cve: "CVE-4567-0002"}: false,
				imageCvePair{image: img1.GetId(), cve: "CVE-1234-0003"}: false,
				imageCvePair{image: img1.GetId(), cve: "CVE-3456-0004"}: false,
				imageCvePair{image: img1.GetId(), cve: "CVE-3456-0005"}: false,
				imageCvePair{image: img1.GetId(), cve: "CVE-2345-0006"}: false,
				imageCvePair{image: img1.GetId(), cve: "CVE-2345-0007"}: false,
			},
		},
		{
			contextKey: sacTestUtils.MixedClusterAndNamespaceReadCtx,
			// The mixed scope context can see cluster1 and namespaceA as well as all cluster2.
			// Therefore it should see all vulnerabilities.
			// (images are in cluster1 namespaceA and cluster2 namespaceB).
			expectedCVEFound: map[imageCvePair]bool{
				imageCvePair{image: img1.GetId(), cve: "CVE-1234-0001"}: true,
				imageCvePair{image: img1.GetId(), cve: "CVE-4567-0002"}: true,
				imageCvePair{image: img1.GetId(), cve: "CVE-1234-0003"}: true,
				imageCvePair{image: img1.GetId(), cve: "CVE-3456-0004"}: true,
				imageCvePair{image: img1.GetId(), cve: "CVE-3456-0005"}: true,
				imageCvePair{image: img1.GetId(), cve: "CVE-2345-0006"}: true,
				imageCvePair{image: img1.GetId(), cve: "CVE-2345-0007"}: true,
			},
		},
	}

	imageCVEByIDMap = map[string]*storage.EmbeddedVulnerability{
		getImageCVEEdgeID("", fixtures.GetEmbeddedImageCVE1234x0001().GetCve()): fixtures.GetEmbeddedImageCVE1234x0001(),
		getImageCVEEdgeID("", fixtures.GetEmbeddedImageCVE4567x0002().GetCve()): fixtures.GetEmbeddedImageCVE4567x0002(),
		getImageCVEEdgeID("", fixtures.GetEmbeddedImageCVE1234x0003().GetCve()): fixtures.GetEmbeddedImageCVE1234x0003(),
		getImageCVEEdgeID("", fixtures.GetEmbeddedImageCVE3456x0004().GetCve()): fixtures.GetEmbeddedImageCVE3456x0004(),
		getImageCVEEdgeID("", fixtures.GetEmbeddedImageCVE3456x0005().GetCve()): fixtures.GetEmbeddedImageCVE3456x0005(),
		getImageCVEEdgeID("", fixtures.GetEmbeddedImageCVE2345x0006().GetCve()): fixtures.GetEmbeddedImageCVE2345x0006(),
		getImageCVEEdgeID("", fixtures.GetEmbeddedImageCVE2345x0007().GetCve()): fixtures.GetEmbeddedImageCVE2345x0007(),
	}
)

const (
	dontWaitForIndexing = false
	waitForIndexing     = true
)

func (s *imageCveEdgeDataStoreSACTestSuite) TestSACImageCVEGetSingleScopeOnly() {
	// Inject the fixture graph, and test retrieval for CVE-1234-0001
	err := s.dackboxTestStore.PushImageToVulnerabilitiesGraph(dontWaitForIndexing)
	defer s.cleanImageToVulnerabilitiesGraph(dontWaitForIndexing)
	s.Require().NoError(err)
	targetCVE := fixtures.GetEmbeddedImageCVE1234x0001()
	cveName := targetCVE.GetCve()
	testImage1 := fixtures.GetImageSherlockHolmes1()
	cveID := imageCvePair{
		image: testImage1.GetId(),
		cve:   cveName,
	}
	for _, c := range imageCVETestCases {
		s.Run(c.contextKey, func() {
			testCtx := s.imageTestContexts[c.contextKey]
			imageCVEEdge, found, err := s.imageCVEEdgeStore.Get(testCtx, cveID.getImageCVEEdgeID())
			s.NoError(err)
			s.Equal(c.expectedCVEFound[cveID], found)
			if c.expectedCVEFound[cveID] {
				s.Require().NotNil(imageCVEEdge)
				s.Equal(cveID.cve, imageCVEEdge.GetImageCveId())
				s.Equal(cveID.image, imageCVEEdge.GetImageId())
			} else {
				s.Nil(imageCVEEdge)
			}
		})
	}
}

func (s *imageCveEdgeDataStoreSACTestSuite) TestSACImageCVEGetSharedAcrossComponents() {
	// Inject the fixture graph, and test retrieval for CVE-4567-0002
	err := s.dackboxTestStore.PushImageToVulnerabilitiesGraph(dontWaitForIndexing)
	defer s.cleanImageToVulnerabilitiesGraph(dontWaitForIndexing)
	s.Require().NoError(err)
	targetCVE := fixtures.GetEmbeddedImageCVE4567x0002()
	cveName := targetCVE.GetCve()
	testImage1 := fixtures.GetImageSherlockHolmes1()
	cveID := imageCvePair{
		image: testImage1.GetId(),
		cve:   cveName,
	}
	for _, c := range imageCVETestCases {
		s.Run(c.contextKey, func() {
			testCtx := s.imageTestContexts[c.contextKey]
			imageCveEdge, found, err := s.imageCVEEdgeStore.Get(testCtx, cveID.getImageCVEEdgeID())
			s.NoError(err)
			s.Equal(c.expectedCVEFound[cveID], found)
			if c.expectedCVEFound[cveID] {
				s.Require().NotNil(imageCveEdge)
				s.Equal(cveID.cve, imageCveEdge.GetImageCveId())
				s.Equal(cveID.image, imageCveEdge.GetImageId())
			} else {
				s.Nil(imageCveEdge)
			}
		})
	}
}

func (s *imageCveEdgeDataStoreSACTestSuite) TestSACImageCVEGetFromSharedComponent() {
	// Inject the fixture graph, and test retrieval for CVE-3456-0004
	err := s.dackboxTestStore.PushImageToVulnerabilitiesGraph(dontWaitForIndexing)
	defer s.cleanImageToVulnerabilitiesGraph(dontWaitForIndexing)
	s.Require().NoError(err)
	targetCVE := fixtures.GetEmbeddedImageCVE3456x0004()
	cveName := targetCVE.GetCve()
	testImage1 := fixtures.GetImageSherlockHolmes1()
	cveID := imageCvePair{
		image: testImage1.GetId(),
		cve:   cveName,
	}
	for _, c := range imageCVETestCases {
		s.Run(c.contextKey, func() {
			testCtx := s.imageTestContexts[c.contextKey]
			imageCveEdge, found, err := s.imageCVEEdgeStore.Get(testCtx, cveID.getImageCVEEdgeID())
			s.NoError(err)
			s.Equal(c.expectedCVEFound[cveID], found)
			if c.expectedCVEFound[cveID] {
				s.Require().NotNil(imageCveEdge)
				s.Equal(cveID.cve, imageCveEdge.GetImageCveId())
			} else {
				s.Nil(imageCveEdge)
			}
		})
	}
}

func (s *imageCveEdgeDataStoreSACTestSuite) TestSACImageCVESearch() {
	err := s.dackboxTestStore.PushImageToVulnerabilitiesGraph(waitForIndexing)
	defer s.cleanImageToVulnerabilitiesGraph(waitForIndexing)
	s.Require().NoError(err)
	for _, c := range imageCVETestCases {
		s.Run(c.contextKey, func() {

			testCtx := s.imageTestContexts[c.contextKey]
			results, err := s.imageCVEEdgeStore.Search(testCtx, nil)
			s.NoError(err)
			expectedCVENames := make([]imageCvePair, 0, len(c.expectedCVEFound))
			for name, visible := range c.expectedCVEFound {
				if visible {
					expectedCVENames = append(expectedCVENames, name)
				}
			}
			fetchedCVEIDs := make(map[string]search.Result, 0)
			for _, result := range results {
				fetchedCVEIDs[result.ID] = result
			}
			fetchedCVENames := make([]imageCvePair, 0, len(fetchedCVEIDs))
			for id := range fetchedCVEIDs {
				fetchedCVENames = append(fetchedCVENames, id2pair(id))
			}
			s.ElementsMatch(fetchedCVENames, expectedCVENames)
		})
	}
}

func (s *imageCveEdgeDataStoreSACTestSuite) TestSACImageCVESearchRawCVEs() {
	err := s.dackboxTestStore.PushImageToVulnerabilitiesGraph(waitForIndexing)
	defer s.cleanImageToVulnerabilitiesGraph(waitForIndexing)
	s.Require().NoError(err)
	for _, c := range imageCVETestCases {
		s.Run(c.contextKey, func() {

			testCtx := s.imageTestContexts[c.contextKey]
			results, err := s.imageCVEEdgeStore.SearchRawEdges(testCtx, nil)
			s.NoError(err)
			expectedCVENames := make([]string, 0, len(c.expectedCVEFound))
			for name, visible := range c.expectedCVEFound {
				if visible {
					expectedCVENames = append(expectedCVENames, name.getImageCVEEdgeID())
				}
			}
			fetchedCVEIDs := make(map[string]*storage.ImageCVEEdge, 0)
			for _, result := range results {
				fetchedCVEIDs[result.GetId()] = result
				s.Equal(imageCVEByIDMap[result.GetId()].GetCve(), result.GetImageCveId())
			}
			fetchedCVENames := make([]string, 0, len(fetchedCVEIDs))
			for id := range fetchedCVEIDs {
				fetchedCVENames = append(fetchedCVENames, id)
			}
			s.ElementsMatch(fetchedCVENames, expectedCVENames)
		})
	}
}
