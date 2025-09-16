//go:build sql_integration

package datastore

import (
	"context"
	"testing"

	graphDBTestUtils "github.com/stackrox/rox/central/graphdb/testutils"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/cve"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	sacTestUtils "github.com/stackrox/rox/pkg/sac/testutils"
	"github.com/stackrox/rox/pkg/scancomponent"
	"github.com/stretchr/testify/suite"
)

var (
	log = logging.LoggerForModule()
)

func TestCVEV2DataStoreSAC(t *testing.T) {
	suite.Run(t, new(cveV2DataStoreSACTestSuite))
}

type cveV2DataStoreSACTestSuite struct {
	suite.Suite

	testGraphDatastore graphDBTestUtils.TestGraphDataStore
	imageCVEStore      DataStore

	imageTestContexts map[string]context.Context
}

func (s *cveV2DataStoreSACTestSuite) SetupSuite() {
	if !features.FlattenCVEData.Enabled() {
		s.T().Setenv(features.FlattenCVEData.EnvVar(), "true")
	}

	var err error
	s.testGraphDatastore, err = graphDBTestUtils.NewTestGraphDataStore(s.T())
	s.Require().NoError(err)
	pool := s.testGraphDatastore.GetPostgresPool()
	s.imageCVEStore = GetTestPostgresDataStore(s.T(), pool)
	s.Require().NoError(err)
	s.imageTestContexts = sacTestUtils.GetNamespaceScopedTestContexts(context.Background(), s.T(), resources.Image)
}

// Vulnerability identifiers have been modified in the migration to Postgres to hold
// operating system information as well. This information is propagated from the image
// scan data.
// This helper is here to ease testing against the various datastore flavours.
func getImageCVEID(vuln *storage.EmbeddedVulnerability, component *storage.EmbeddedImageScanComponent, imageID string) string {
	componentID, _ := scancomponent.ComponentIDV2(component, imageID)
	cveID, _ := cve.IDV2(vuln, componentID)
	return cveID
}

func (s *cveV2DataStoreSACTestSuite) cleanImageToVulnerabilitiesGraph() {
	s.Require().NoError(s.testGraphDatastore.CleanImageToVulnerabilitiesGraph())
}

type cveTestCase struct {
	contextKey       string
	expectedCVEFound map[string]bool
}

var (
	imageCVETestCases = []cveTestCase{
		{
			contextKey: sacTestUtils.UnrestrictedReadCtx,
			expectedCVEFound: map[string]bool{
				getImageCVEID(fixtures.GetEmbeddedImageCVE1234x0001(), fixtures.GetEmbeddedImageComponent1x1(), fixtures.GetImageSherlockHolmes1().GetId()):   true,
				getImageCVEID(fixtures.GetEmbeddedImageCVE4567x0002(), fixtures.GetEmbeddedImageComponent1x1(), fixtures.GetImageSherlockHolmes1().GetId()):   true,
				getImageCVEID(fixtures.GetEmbeddedImageCVE1234x0003(), fixtures.GetEmbeddedImageComponent1x2(), fixtures.GetImageSherlockHolmes1().GetId()):   true,
				getImageCVEID(fixtures.GetEmbeddedImageCVE3456x0004(), fixtures.GetEmbeddedImageComponent1s2x3(), fixtures.GetImageSherlockHolmes1().GetId()): true,
				getImageCVEID(fixtures.GetEmbeddedImageCVE3456x0005(), fixtures.GetEmbeddedImageComponent1s2x3(), fixtures.GetImageSherlockHolmes1().GetId()): true,
				getImageCVEID(fixtures.GetEmbeddedImageCVE3456x0004(), fixtures.GetEmbeddedImageComponent1s2x3(), fixtures.GetImageDoctorJekyll2().GetId()):   true,
				getImageCVEID(fixtures.GetEmbeddedImageCVE3456x0005(), fixtures.GetEmbeddedImageComponent1s2x3(), fixtures.GetImageDoctorJekyll2().GetId()):   true,
				getImageCVEID(fixtures.GetEmbeddedImageCVE4567x0002(), fixtures.GetEmbeddedImageComponent2x5(), fixtures.GetImageDoctorJekyll2().GetId()):     true,
				getImageCVEID(fixtures.GetEmbeddedImageCVE2345x0006(), fixtures.GetEmbeddedImageComponent2x5(), fixtures.GetImageDoctorJekyll2().GetId()):     true,
				getImageCVEID(fixtures.GetEmbeddedImageCVE2345x0007(), fixtures.GetEmbeddedImageComponent2x5(), fixtures.GetImageDoctorJekyll2().GetId()):     true,
			},
		},
		{
			contextKey: sacTestUtils.UnrestrictedReadWriteCtx,
			expectedCVEFound: map[string]bool{
				getImageCVEID(fixtures.GetEmbeddedImageCVE1234x0001(), fixtures.GetEmbeddedImageComponent1x1(), fixtures.GetImageSherlockHolmes1().GetId()):   true,
				getImageCVEID(fixtures.GetEmbeddedImageCVE4567x0002(), fixtures.GetEmbeddedImageComponent1x1(), fixtures.GetImageSherlockHolmes1().GetId()):   true,
				getImageCVEID(fixtures.GetEmbeddedImageCVE1234x0003(), fixtures.GetEmbeddedImageComponent1x2(), fixtures.GetImageSherlockHolmes1().GetId()):   true,
				getImageCVEID(fixtures.GetEmbeddedImageCVE3456x0004(), fixtures.GetEmbeddedImageComponent1s2x3(), fixtures.GetImageSherlockHolmes1().GetId()): true,
				getImageCVEID(fixtures.GetEmbeddedImageCVE3456x0005(), fixtures.GetEmbeddedImageComponent1s2x3(), fixtures.GetImageSherlockHolmes1().GetId()): true,
				getImageCVEID(fixtures.GetEmbeddedImageCVE3456x0004(), fixtures.GetEmbeddedImageComponent1s2x3(), fixtures.GetImageDoctorJekyll2().GetId()):   true,
				getImageCVEID(fixtures.GetEmbeddedImageCVE3456x0005(), fixtures.GetEmbeddedImageComponent1s2x3(), fixtures.GetImageDoctorJekyll2().GetId()):   true,
				getImageCVEID(fixtures.GetEmbeddedImageCVE4567x0002(), fixtures.GetEmbeddedImageComponent2x5(), fixtures.GetImageDoctorJekyll2().GetId()):     true,
				getImageCVEID(fixtures.GetEmbeddedImageCVE2345x0006(), fixtures.GetEmbeddedImageComponent2x5(), fixtures.GetImageDoctorJekyll2().GetId()):     true,
				getImageCVEID(fixtures.GetEmbeddedImageCVE2345x0007(), fixtures.GetEmbeddedImageComponent2x5(), fixtures.GetImageDoctorJekyll2().GetId()):     true,
			},
		},
		{
			contextKey: sacTestUtils.Cluster1ReadWriteCtx,
			expectedCVEFound: map[string]bool{
				getImageCVEID(fixtures.GetEmbeddedImageCVE1234x0001(), fixtures.GetEmbeddedImageComponent1x1(), fixtures.GetImageSherlockHolmes1().GetId()):   true,
				getImageCVEID(fixtures.GetEmbeddedImageCVE4567x0002(), fixtures.GetEmbeddedImageComponent1x1(), fixtures.GetImageSherlockHolmes1().GetId()):   true,
				getImageCVEID(fixtures.GetEmbeddedImageCVE1234x0003(), fixtures.GetEmbeddedImageComponent1x2(), fixtures.GetImageSherlockHolmes1().GetId()):   true,
				getImageCVEID(fixtures.GetEmbeddedImageCVE3456x0004(), fixtures.GetEmbeddedImageComponent1s2x3(), fixtures.GetImageSherlockHolmes1().GetId()): true,
				getImageCVEID(fixtures.GetEmbeddedImageCVE3456x0005(), fixtures.GetEmbeddedImageComponent1s2x3(), fixtures.GetImageSherlockHolmes1().GetId()): true,
				getImageCVEID(fixtures.GetEmbeddedImageCVE3456x0004(), fixtures.GetEmbeddedImageComponent1s2x3(), fixtures.GetImageDoctorJekyll2().GetId()):   false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE3456x0005(), fixtures.GetEmbeddedImageComponent1s2x3(), fixtures.GetImageDoctorJekyll2().GetId()):   false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE4567x0002(), fixtures.GetEmbeddedImageComponent2x5(), fixtures.GetImageDoctorJekyll2().GetId()):     false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE2345x0006(), fixtures.GetEmbeddedImageComponent2x5(), fixtures.GetImageDoctorJekyll2().GetId()):     false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE2345x0007(), fixtures.GetEmbeddedImageComponent2x5(), fixtures.GetImageDoctorJekyll2().GetId()):     false,
			},
		},
		{
			contextKey: sacTestUtils.Cluster1NamespaceAReadWriteCtx,
			expectedCVEFound: map[string]bool{
				getImageCVEID(fixtures.GetEmbeddedImageCVE1234x0001(), fixtures.GetEmbeddedImageComponent1x1(), fixtures.GetImageSherlockHolmes1().GetId()):   true,
				getImageCVEID(fixtures.GetEmbeddedImageCVE4567x0002(), fixtures.GetEmbeddedImageComponent1x1(), fixtures.GetImageSherlockHolmes1().GetId()):   true,
				getImageCVEID(fixtures.GetEmbeddedImageCVE1234x0003(), fixtures.GetEmbeddedImageComponent1x2(), fixtures.GetImageSherlockHolmes1().GetId()):   true,
				getImageCVEID(fixtures.GetEmbeddedImageCVE3456x0004(), fixtures.GetEmbeddedImageComponent1s2x3(), fixtures.GetImageSherlockHolmes1().GetId()): true,
				getImageCVEID(fixtures.GetEmbeddedImageCVE3456x0005(), fixtures.GetEmbeddedImageComponent1s2x3(), fixtures.GetImageSherlockHolmes1().GetId()): true,
				getImageCVEID(fixtures.GetEmbeddedImageCVE3456x0004(), fixtures.GetEmbeddedImageComponent1s2x3(), fixtures.GetImageDoctorJekyll2().GetId()):   false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE3456x0005(), fixtures.GetEmbeddedImageComponent1s2x3(), fixtures.GetImageDoctorJekyll2().GetId()):   false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE4567x0002(), fixtures.GetEmbeddedImageComponent2x5(), fixtures.GetImageDoctorJekyll2().GetId()):     false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE2345x0006(), fixtures.GetEmbeddedImageComponent2x5(), fixtures.GetImageDoctorJekyll2().GetId()):     false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE2345x0007(), fixtures.GetEmbeddedImageComponent2x5(), fixtures.GetImageDoctorJekyll2().GetId()):     false,
			},
		},
		{
			contextKey: sacTestUtils.Cluster1NamespaceBReadWriteCtx,
			expectedCVEFound: map[string]bool{
				getImageCVEID(fixtures.GetEmbeddedImageCVE1234x0001(), fixtures.GetEmbeddedImageComponent1x1(), fixtures.GetImageSherlockHolmes1().GetId()):   false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE4567x0002(), fixtures.GetEmbeddedImageComponent1x1(), fixtures.GetImageSherlockHolmes1().GetId()):   false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE1234x0003(), fixtures.GetEmbeddedImageComponent1x2(), fixtures.GetImageSherlockHolmes1().GetId()):   false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE3456x0004(), fixtures.GetEmbeddedImageComponent1s2x3(), fixtures.GetImageSherlockHolmes1().GetId()): false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE3456x0005(), fixtures.GetEmbeddedImageComponent1s2x3(), fixtures.GetImageSherlockHolmes1().GetId()): false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE3456x0004(), fixtures.GetEmbeddedImageComponent1s2x3(), fixtures.GetImageDoctorJekyll2().GetId()):   false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE3456x0005(), fixtures.GetEmbeddedImageComponent1s2x3(), fixtures.GetImageDoctorJekyll2().GetId()):   false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE4567x0002(), fixtures.GetEmbeddedImageComponent2x5(), fixtures.GetImageDoctorJekyll2().GetId()):     false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE2345x0006(), fixtures.GetEmbeddedImageComponent2x5(), fixtures.GetImageDoctorJekyll2().GetId()):     false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE2345x0007(), fixtures.GetEmbeddedImageComponent2x5(), fixtures.GetImageDoctorJekyll2().GetId()):     false,
			},
		},
		{
			contextKey: sacTestUtils.Cluster1NamespacesABReadWriteCtx,
			expectedCVEFound: map[string]bool{
				getImageCVEID(fixtures.GetEmbeddedImageCVE1234x0001(), fixtures.GetEmbeddedImageComponent1x1(), fixtures.GetImageSherlockHolmes1().GetId()):   true,
				getImageCVEID(fixtures.GetEmbeddedImageCVE4567x0002(), fixtures.GetEmbeddedImageComponent1x1(), fixtures.GetImageSherlockHolmes1().GetId()):   true,
				getImageCVEID(fixtures.GetEmbeddedImageCVE1234x0003(), fixtures.GetEmbeddedImageComponent1x2(), fixtures.GetImageSherlockHolmes1().GetId()):   true,
				getImageCVEID(fixtures.GetEmbeddedImageCVE3456x0004(), fixtures.GetEmbeddedImageComponent1s2x3(), fixtures.GetImageSherlockHolmes1().GetId()): true,
				getImageCVEID(fixtures.GetEmbeddedImageCVE3456x0005(), fixtures.GetEmbeddedImageComponent1s2x3(), fixtures.GetImageSherlockHolmes1().GetId()): true,
				getImageCVEID(fixtures.GetEmbeddedImageCVE3456x0004(), fixtures.GetEmbeddedImageComponent1s2x3(), fixtures.GetImageDoctorJekyll2().GetId()):   false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE3456x0005(), fixtures.GetEmbeddedImageComponent1s2x3(), fixtures.GetImageDoctorJekyll2().GetId()):   false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE4567x0002(), fixtures.GetEmbeddedImageComponent2x5(), fixtures.GetImageDoctorJekyll2().GetId()):     false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE2345x0006(), fixtures.GetEmbeddedImageComponent2x5(), fixtures.GetImageDoctorJekyll2().GetId()):     false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE2345x0007(), fixtures.GetEmbeddedImageComponent2x5(), fixtures.GetImageDoctorJekyll2().GetId()):     false,
			},
		},
		{
			contextKey: sacTestUtils.Cluster1NamespacesBCReadWriteCtx,
			expectedCVEFound: map[string]bool{
				getImageCVEID(fixtures.GetEmbeddedImageCVE1234x0001(), fixtures.GetEmbeddedImageComponent1x1(), fixtures.GetImageSherlockHolmes1().GetId()):   false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE4567x0002(), fixtures.GetEmbeddedImageComponent1x1(), fixtures.GetImageSherlockHolmes1().GetId()):   false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE1234x0003(), fixtures.GetEmbeddedImageComponent1x2(), fixtures.GetImageSherlockHolmes1().GetId()):   false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE3456x0004(), fixtures.GetEmbeddedImageComponent1s2x3(), fixtures.GetImageSherlockHolmes1().GetId()): false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE3456x0005(), fixtures.GetEmbeddedImageComponent1s2x3(), fixtures.GetImageSherlockHolmes1().GetId()): false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE3456x0004(), fixtures.GetEmbeddedImageComponent1s2x3(), fixtures.GetImageDoctorJekyll2().GetId()):   false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE3456x0005(), fixtures.GetEmbeddedImageComponent1s2x3(), fixtures.GetImageDoctorJekyll2().GetId()):   false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE4567x0002(), fixtures.GetEmbeddedImageComponent2x5(), fixtures.GetImageDoctorJekyll2().GetId()):     false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE2345x0006(), fixtures.GetEmbeddedImageComponent2x5(), fixtures.GetImageDoctorJekyll2().GetId()):     false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE2345x0007(), fixtures.GetEmbeddedImageComponent2x5(), fixtures.GetImageDoctorJekyll2().GetId()):     false,
			},
		},
		{
			contextKey: sacTestUtils.Cluster2ReadWriteCtx,
			expectedCVEFound: map[string]bool{
				getImageCVEID(fixtures.GetEmbeddedImageCVE1234x0001(), fixtures.GetEmbeddedImageComponent1x1(), fixtures.GetImageSherlockHolmes1().GetId()):   false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE4567x0002(), fixtures.GetEmbeddedImageComponent1x1(), fixtures.GetImageSherlockHolmes1().GetId()):   false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE1234x0003(), fixtures.GetEmbeddedImageComponent1x2(), fixtures.GetImageSherlockHolmes1().GetId()):   false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE3456x0004(), fixtures.GetEmbeddedImageComponent1s2x3(), fixtures.GetImageSherlockHolmes1().GetId()): false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE3456x0005(), fixtures.GetEmbeddedImageComponent1s2x3(), fixtures.GetImageSherlockHolmes1().GetId()): false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE3456x0004(), fixtures.GetEmbeddedImageComponent1s2x3(), fixtures.GetImageDoctorJekyll2().GetId()):   true,
				getImageCVEID(fixtures.GetEmbeddedImageCVE3456x0005(), fixtures.GetEmbeddedImageComponent1s2x3(), fixtures.GetImageDoctorJekyll2().GetId()):   true,
				getImageCVEID(fixtures.GetEmbeddedImageCVE4567x0002(), fixtures.GetEmbeddedImageComponent2x5(), fixtures.GetImageDoctorJekyll2().GetId()):     true,
				getImageCVEID(fixtures.GetEmbeddedImageCVE2345x0006(), fixtures.GetEmbeddedImageComponent2x5(), fixtures.GetImageDoctorJekyll2().GetId()):     true,
				getImageCVEID(fixtures.GetEmbeddedImageCVE2345x0007(), fixtures.GetEmbeddedImageComponent2x5(), fixtures.GetImageDoctorJekyll2().GetId()):     true,
			},
		},
		{
			contextKey: sacTestUtils.Cluster2NamespaceAReadWriteCtx,
			expectedCVEFound: map[string]bool{
				getImageCVEID(fixtures.GetEmbeddedImageCVE1234x0001(), fixtures.GetEmbeddedImageComponent1x1(), fixtures.GetImageSherlockHolmes1().GetId()):   false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE4567x0002(), fixtures.GetEmbeddedImageComponent1x1(), fixtures.GetImageSherlockHolmes1().GetId()):   false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE1234x0003(), fixtures.GetEmbeddedImageComponent1x2(), fixtures.GetImageSherlockHolmes1().GetId()):   false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE3456x0004(), fixtures.GetEmbeddedImageComponent1s2x3(), fixtures.GetImageSherlockHolmes1().GetId()): false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE3456x0005(), fixtures.GetEmbeddedImageComponent1s2x3(), fixtures.GetImageSherlockHolmes1().GetId()): false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE3456x0004(), fixtures.GetEmbeddedImageComponent1s2x3(), fixtures.GetImageDoctorJekyll2().GetId()):   false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE3456x0005(), fixtures.GetEmbeddedImageComponent1s2x3(), fixtures.GetImageDoctorJekyll2().GetId()):   false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE4567x0002(), fixtures.GetEmbeddedImageComponent2x5(), fixtures.GetImageDoctorJekyll2().GetId()):     false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE2345x0006(), fixtures.GetEmbeddedImageComponent2x5(), fixtures.GetImageDoctorJekyll2().GetId()):     false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE2345x0007(), fixtures.GetEmbeddedImageComponent2x5(), fixtures.GetImageDoctorJekyll2().GetId()):     false,
			},
		},
		{
			contextKey: sacTestUtils.Cluster2NamespaceBReadWriteCtx,
			expectedCVEFound: map[string]bool{
				getImageCVEID(fixtures.GetEmbeddedImageCVE1234x0001(), fixtures.GetEmbeddedImageComponent1x1(), fixtures.GetImageSherlockHolmes1().GetId()):   false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE4567x0002(), fixtures.GetEmbeddedImageComponent1x1(), fixtures.GetImageSherlockHolmes1().GetId()):   false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE1234x0003(), fixtures.GetEmbeddedImageComponent1x2(), fixtures.GetImageSherlockHolmes1().GetId()):   false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE3456x0004(), fixtures.GetEmbeddedImageComponent1s2x3(), fixtures.GetImageSherlockHolmes1().GetId()): false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE3456x0005(), fixtures.GetEmbeddedImageComponent1s2x3(), fixtures.GetImageSherlockHolmes1().GetId()): false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE3456x0004(), fixtures.GetEmbeddedImageComponent1s2x3(), fixtures.GetImageDoctorJekyll2().GetId()):   true,
				getImageCVEID(fixtures.GetEmbeddedImageCVE3456x0005(), fixtures.GetEmbeddedImageComponent1s2x3(), fixtures.GetImageDoctorJekyll2().GetId()):   true,
				getImageCVEID(fixtures.GetEmbeddedImageCVE4567x0002(), fixtures.GetEmbeddedImageComponent2x5(), fixtures.GetImageDoctorJekyll2().GetId()):     true,
				getImageCVEID(fixtures.GetEmbeddedImageCVE2345x0006(), fixtures.GetEmbeddedImageComponent2x5(), fixtures.GetImageDoctorJekyll2().GetId()):     true,
				getImageCVEID(fixtures.GetEmbeddedImageCVE2345x0007(), fixtures.GetEmbeddedImageComponent2x5(), fixtures.GetImageDoctorJekyll2().GetId()):     true,
			},
		},
		{
			contextKey: sacTestUtils.Cluster2NamespacesACReadWriteCtx,
			expectedCVEFound: map[string]bool{
				getImageCVEID(fixtures.GetEmbeddedImageCVE1234x0001(), fixtures.GetEmbeddedImageComponent1x1(), fixtures.GetImageSherlockHolmes1().GetId()):   false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE4567x0002(), fixtures.GetEmbeddedImageComponent1x1(), fixtures.GetImageSherlockHolmes1().GetId()):   false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE1234x0003(), fixtures.GetEmbeddedImageComponent1x2(), fixtures.GetImageSherlockHolmes1().GetId()):   false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE3456x0004(), fixtures.GetEmbeddedImageComponent1s2x3(), fixtures.GetImageSherlockHolmes1().GetId()): false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE3456x0005(), fixtures.GetEmbeddedImageComponent1s2x3(), fixtures.GetImageSherlockHolmes1().GetId()): false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE3456x0004(), fixtures.GetEmbeddedImageComponent1s2x3(), fixtures.GetImageDoctorJekyll2().GetId()):   false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE3456x0005(), fixtures.GetEmbeddedImageComponent1s2x3(), fixtures.GetImageDoctorJekyll2().GetId()):   false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE4567x0002(), fixtures.GetEmbeddedImageComponent2x5(), fixtures.GetImageDoctorJekyll2().GetId()):     false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE2345x0006(), fixtures.GetEmbeddedImageComponent2x5(), fixtures.GetImageDoctorJekyll2().GetId()):     false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE2345x0007(), fixtures.GetEmbeddedImageComponent2x5(), fixtures.GetImageDoctorJekyll2().GetId()):     false,
			},
		},
		{
			contextKey: sacTestUtils.Cluster2NamespacesBCReadWriteCtx,
			expectedCVEFound: map[string]bool{
				getImageCVEID(fixtures.GetEmbeddedImageCVE1234x0001(), fixtures.GetEmbeddedImageComponent1x1(), fixtures.GetImageSherlockHolmes1().GetId()):   false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE4567x0002(), fixtures.GetEmbeddedImageComponent1x1(), fixtures.GetImageSherlockHolmes1().GetId()):   false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE1234x0003(), fixtures.GetEmbeddedImageComponent1x2(), fixtures.GetImageSherlockHolmes1().GetId()):   false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE3456x0004(), fixtures.GetEmbeddedImageComponent1s2x3(), fixtures.GetImageSherlockHolmes1().GetId()): false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE3456x0005(), fixtures.GetEmbeddedImageComponent1s2x3(), fixtures.GetImageSherlockHolmes1().GetId()): false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE3456x0004(), fixtures.GetEmbeddedImageComponent1s2x3(), fixtures.GetImageDoctorJekyll2().GetId()):   true,
				getImageCVEID(fixtures.GetEmbeddedImageCVE3456x0005(), fixtures.GetEmbeddedImageComponent1s2x3(), fixtures.GetImageDoctorJekyll2().GetId()):   true,
				getImageCVEID(fixtures.GetEmbeddedImageCVE4567x0002(), fixtures.GetEmbeddedImageComponent2x5(), fixtures.GetImageDoctorJekyll2().GetId()):     true,
				getImageCVEID(fixtures.GetEmbeddedImageCVE2345x0006(), fixtures.GetEmbeddedImageComponent2x5(), fixtures.GetImageDoctorJekyll2().GetId()):     true,
				getImageCVEID(fixtures.GetEmbeddedImageCVE2345x0007(), fixtures.GetEmbeddedImageComponent2x5(), fixtures.GetImageDoctorJekyll2().GetId()):     true,
			},
		},
		{
			contextKey: sacTestUtils.Cluster3ReadWriteCtx,
			expectedCVEFound: map[string]bool{
				getImageCVEID(fixtures.GetEmbeddedImageCVE1234x0001(), fixtures.GetEmbeddedImageComponent1x1(), fixtures.GetImageSherlockHolmes1().GetId()):   false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE4567x0002(), fixtures.GetEmbeddedImageComponent1x1(), fixtures.GetImageSherlockHolmes1().GetId()):   false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE1234x0003(), fixtures.GetEmbeddedImageComponent1x2(), fixtures.GetImageSherlockHolmes1().GetId()):   false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE3456x0004(), fixtures.GetEmbeddedImageComponent1s2x3(), fixtures.GetImageSherlockHolmes1().GetId()): false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE3456x0005(), fixtures.GetEmbeddedImageComponent1s2x3(), fixtures.GetImageSherlockHolmes1().GetId()): false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE3456x0004(), fixtures.GetEmbeddedImageComponent1s2x3(), fixtures.GetImageDoctorJekyll2().GetId()):   false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE3456x0005(), fixtures.GetEmbeddedImageComponent1s2x3(), fixtures.GetImageDoctorJekyll2().GetId()):   false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE4567x0002(), fixtures.GetEmbeddedImageComponent2x5(), fixtures.GetImageDoctorJekyll2().GetId()):     false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE2345x0006(), fixtures.GetEmbeddedImageComponent2x5(), fixtures.GetImageDoctorJekyll2().GetId()):     false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE2345x0007(), fixtures.GetEmbeddedImageComponent2x5(), fixtures.GetImageDoctorJekyll2().GetId()):     false,
			},
		},
		{
			contextKey: sacTestUtils.Cluster3NamespaceAReadWriteCtx,
			expectedCVEFound: map[string]bool{
				getImageCVEID(fixtures.GetEmbeddedImageCVE1234x0001(), fixtures.GetEmbeddedImageComponent1x1(), fixtures.GetImageSherlockHolmes1().GetId()):   false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE4567x0002(), fixtures.GetEmbeddedImageComponent1x1(), fixtures.GetImageSherlockHolmes1().GetId()):   false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE1234x0003(), fixtures.GetEmbeddedImageComponent1x2(), fixtures.GetImageSherlockHolmes1().GetId()):   false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE3456x0004(), fixtures.GetEmbeddedImageComponent1s2x3(), fixtures.GetImageSherlockHolmes1().GetId()): false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE3456x0005(), fixtures.GetEmbeddedImageComponent1s2x3(), fixtures.GetImageSherlockHolmes1().GetId()): false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE3456x0004(), fixtures.GetEmbeddedImageComponent1s2x3(), fixtures.GetImageDoctorJekyll2().GetId()):   false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE3456x0005(), fixtures.GetEmbeddedImageComponent1s2x3(), fixtures.GetImageDoctorJekyll2().GetId()):   false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE4567x0002(), fixtures.GetEmbeddedImageComponent2x5(), fixtures.GetImageDoctorJekyll2().GetId()):     false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE2345x0006(), fixtures.GetEmbeddedImageComponent2x5(), fixtures.GetImageDoctorJekyll2().GetId()):     false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE2345x0007(), fixtures.GetEmbeddedImageComponent2x5(), fixtures.GetImageDoctorJekyll2().GetId()):     false,
			},
		},
		{
			contextKey: sacTestUtils.Cluster3NamespaceBReadWriteCtx,
			expectedCVEFound: map[string]bool{
				getImageCVEID(fixtures.GetEmbeddedImageCVE1234x0001(), fixtures.GetEmbeddedImageComponent1x1(), fixtures.GetImageSherlockHolmes1().GetId()):   false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE4567x0002(), fixtures.GetEmbeddedImageComponent1x1(), fixtures.GetImageSherlockHolmes1().GetId()):   false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE1234x0003(), fixtures.GetEmbeddedImageComponent1x2(), fixtures.GetImageSherlockHolmes1().GetId()):   false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE3456x0004(), fixtures.GetEmbeddedImageComponent1s2x3(), fixtures.GetImageSherlockHolmes1().GetId()): false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE3456x0005(), fixtures.GetEmbeddedImageComponent1s2x3(), fixtures.GetImageSherlockHolmes1().GetId()): false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE3456x0004(), fixtures.GetEmbeddedImageComponent1s2x3(), fixtures.GetImageDoctorJekyll2().GetId()):   false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE3456x0005(), fixtures.GetEmbeddedImageComponent1s2x3(), fixtures.GetImageDoctorJekyll2().GetId()):   false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE4567x0002(), fixtures.GetEmbeddedImageComponent2x5(), fixtures.GetImageDoctorJekyll2().GetId()):     false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE2345x0006(), fixtures.GetEmbeddedImageComponent2x5(), fixtures.GetImageDoctorJekyll2().GetId()):     false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE2345x0007(), fixtures.GetEmbeddedImageComponent2x5(), fixtures.GetImageDoctorJekyll2().GetId()):     false,
			},
		},
		{
			contextKey: sacTestUtils.Cluster3NamespacesABReadWriteCtx,
			expectedCVEFound: map[string]bool{
				getImageCVEID(fixtures.GetEmbeddedImageCVE1234x0001(), fixtures.GetEmbeddedImageComponent1x1(), fixtures.GetImageSherlockHolmes1().GetId()):   false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE4567x0002(), fixtures.GetEmbeddedImageComponent1x1(), fixtures.GetImageSherlockHolmes1().GetId()):   false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE1234x0003(), fixtures.GetEmbeddedImageComponent1x2(), fixtures.GetImageSherlockHolmes1().GetId()):   false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE3456x0004(), fixtures.GetEmbeddedImageComponent1s2x3(), fixtures.GetImageSherlockHolmes1().GetId()): false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE3456x0005(), fixtures.GetEmbeddedImageComponent1s2x3(), fixtures.GetImageSherlockHolmes1().GetId()): false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE3456x0004(), fixtures.GetEmbeddedImageComponent1s2x3(), fixtures.GetImageDoctorJekyll2().GetId()):   false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE3456x0005(), fixtures.GetEmbeddedImageComponent1s2x3(), fixtures.GetImageDoctorJekyll2().GetId()):   false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE4567x0002(), fixtures.GetEmbeddedImageComponent2x5(), fixtures.GetImageDoctorJekyll2().GetId()):     false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE2345x0006(), fixtures.GetEmbeddedImageComponent2x5(), fixtures.GetImageDoctorJekyll2().GetId()):     false,
				getImageCVEID(fixtures.GetEmbeddedImageCVE2345x0007(), fixtures.GetEmbeddedImageComponent2x5(), fixtures.GetImageDoctorJekyll2().GetId()):     false,
			},
		},
		{
			contextKey: sacTestUtils.MixedClusterAndNamespaceReadCtx,
			// The mixed scope context can see cluster1 and namespaceA as well as all cluster2.
			// Therefore, it should see all vulnerabilities.
			// (images are in cluster1 namespaceA and cluster2 namespaceB).
			expectedCVEFound: map[string]bool{
				getImageCVEID(fixtures.GetEmbeddedImageCVE1234x0001(), fixtures.GetEmbeddedImageComponent1x1(), fixtures.GetImageSherlockHolmes1().GetId()):   true,
				getImageCVEID(fixtures.GetEmbeddedImageCVE4567x0002(), fixtures.GetEmbeddedImageComponent1x1(), fixtures.GetImageSherlockHolmes1().GetId()):   true,
				getImageCVEID(fixtures.GetEmbeddedImageCVE1234x0003(), fixtures.GetEmbeddedImageComponent1x2(), fixtures.GetImageSherlockHolmes1().GetId()):   true,
				getImageCVEID(fixtures.GetEmbeddedImageCVE3456x0004(), fixtures.GetEmbeddedImageComponent1s2x3(), fixtures.GetImageSherlockHolmes1().GetId()): true,
				getImageCVEID(fixtures.GetEmbeddedImageCVE3456x0005(), fixtures.GetEmbeddedImageComponent1s2x3(), fixtures.GetImageSherlockHolmes1().GetId()): true,
				getImageCVEID(fixtures.GetEmbeddedImageCVE3456x0004(), fixtures.GetEmbeddedImageComponent1s2x3(), fixtures.GetImageDoctorJekyll2().GetId()):   true,
				getImageCVEID(fixtures.GetEmbeddedImageCVE3456x0005(), fixtures.GetEmbeddedImageComponent1s2x3(), fixtures.GetImageDoctorJekyll2().GetId()):   true,
				getImageCVEID(fixtures.GetEmbeddedImageCVE4567x0002(), fixtures.GetEmbeddedImageComponent2x5(), fixtures.GetImageDoctorJekyll2().GetId()):     true,
				getImageCVEID(fixtures.GetEmbeddedImageCVE2345x0006(), fixtures.GetEmbeddedImageComponent2x5(), fixtures.GetImageDoctorJekyll2().GetId()):     true,
				getImageCVEID(fixtures.GetEmbeddedImageCVE2345x0007(), fixtures.GetEmbeddedImageComponent2x5(), fixtures.GetImageDoctorJekyll2().GetId()):     true,
			},
		},
	}
)

func (s *cveV2DataStoreSACTestSuite) TestSACImageCVEExistsSingleScopeOnly() {
	// Inject the fixture graph, and test exists for CVE-1234-0001
	cveID := getImageCVEID(fixtures.GetEmbeddedImageCVE1234x0001(), fixtures.GetEmbeddedImageComponent1x1(), fixtures.GetImageSherlockHolmes1().GetId())
	s.runImageTest("TestSACImageCVEExistsSingleScopeOnly", func(c cveTestCase) {
		testCtx := s.imageTestContexts[c.contextKey]
		exists, err := s.imageCVEStore.Exists(testCtx, cveID)
		s.NoError(err)
		s.Equal(c.expectedCVEFound[cveID], exists)
	})
}

func (s *cveV2DataStoreSACTestSuite) TestSACImageCVEGetSingleScopeOnly() {
	// Inject the fixture graph, and test retrieval for CVE-1234-0001
	targetCVE := fixtures.GetEmbeddedImageCVE1234x0001()
	cveName := targetCVE.GetCve()
	cveID := getImageCVEID(fixtures.GetEmbeddedImageCVE1234x0001(), fixtures.GetEmbeddedImageComponent1x1(), fixtures.GetImageSherlockHolmes1().GetId())
	cvss := targetCVE.GetCvss()
	s.runImageTest("TestSACImageCVEGetSingleScopeOnly", func(c cveTestCase) {
		testCtx := s.imageTestContexts[c.contextKey]
		imageCVE, found, err := s.imageCVEStore.Get(testCtx, cveID)
		s.NoError(err)
		s.Equal(c.expectedCVEFound[cveID], found)
		if c.expectedCVEFound[cveID] {
			s.Require().NotNil(imageCVE)
			s.Equal(cveName, imageCVE.GetCveBaseInfo().GetCve())
			s.Equal(cvss, imageCVE.Cvss)
		} else {
			s.Nil(imageCVE)
		}
	})
}

func (s *cveV2DataStoreSACTestSuite) TestSACImageCVEGetBatch() {
	batchCVEs := []string{
		getImageCVEID(fixtures.GetEmbeddedImageCVE1234x0001(), fixtures.GetEmbeddedImageComponent1x1(), fixtures.GetImageSherlockHolmes1().GetId()),
		getImageCVEID(fixtures.GetEmbeddedImageCVE4567x0002(), fixtures.GetEmbeddedImageComponent1x1(), fixtures.GetImageSherlockHolmes1().GetId()),
		getImageCVEID(fixtures.GetEmbeddedImageCVE1234x0003(), fixtures.GetEmbeddedImageComponent1x2(), fixtures.GetImageSherlockHolmes1().GetId()),
		getImageCVEID(fixtures.GetEmbeddedImageCVE3456x0004(), fixtures.GetEmbeddedImageComponent1s2x3(), fixtures.GetImageSherlockHolmes1().GetId()),
		getImageCVEID(fixtures.GetEmbeddedImageCVE3456x0005(), fixtures.GetEmbeddedImageComponent1s2x3(), fixtures.GetImageSherlockHolmes1().GetId()),
		getImageCVEID(fixtures.GetEmbeddedImageCVE2345x0006(), fixtures.GetEmbeddedImageComponent2x5(), fixtures.GetImageDoctorJekyll2().GetId()),
		getImageCVEID(fixtures.GetEmbeddedImageCVE2345x0007(), fixtures.GetEmbeddedImageComponent2x5(), fixtures.GetImageDoctorJekyll2().GetId()),
	}

	s.runImageTest("TestSACImageCVEGetBatch", func(c cveTestCase) {
		testCtx := s.imageTestContexts[c.contextKey]
		imageCVEs, err := s.imageCVEStore.GetBatch(testCtx, batchCVEs)
		s.NoError(err)
		expectedCVEIDs := make([]string, 0, len(batchCVEs))
		for _, cveID := range batchCVEs {
			if c.expectedCVEFound[cveID] {
				expectedCVEIDs = append(expectedCVEIDs, cveID)
			}
		}
		fetchedCVEIDs := make([]string, 0, len(imageCVEs))
		for _, imageCVE := range imageCVEs {
			fetchedCVEIDs = append(fetchedCVEIDs, imageCVE.GetId())
		}
		s.ElementsMatch(expectedCVEIDs, fetchedCVEIDs)
	})
}

func (s *cveV2DataStoreSACTestSuite) TestSACImageCVESearch() {
	s.runImageTest("TestSACImageCVESearch", func(c cveTestCase) {

		testCtx := s.imageTestContexts[c.contextKey]
		results, err := s.imageCVEStore.Search(testCtx, nil)
		s.NoError(err)
		expectedCVEIDs := make([]string, 0, len(c.expectedCVEFound))
		for name, visible := range c.expectedCVEFound {
			if visible {
				expectedCVEIDs = append(expectedCVEIDs, name)
			}
		}

		fetchedCVEIDs := make([]string, 0, len(results))
		for _, result := range results {
			fetchedCVEIDs = append(fetchedCVEIDs, result.ID)
		}
		s.ElementsMatch(expectedCVEIDs, fetchedCVEIDs)
	})
}

func (s *cveV2DataStoreSACTestSuite) TestSACImageCVESearchCVEs() {
	s.runImageTest("TestSACImageCVESearchCVEs", func(c cveTestCase) {

		testCtx := s.imageTestContexts[c.contextKey]
		results, err := s.imageCVEStore.SearchImageCVEs(testCtx, nil)
		s.NoError(err)
		expectedCVEIDs := make([]string, 0, len(c.expectedCVEFound))
		for name, visible := range c.expectedCVEFound {
			if visible {
				expectedCVEIDs = append(expectedCVEIDs, name)
			}
		}

		fetchedCVEIDs := make([]string, 0, len(results))
		for _, result := range results {
			fetchedCVEIDs = append(fetchedCVEIDs, result.Id)
		}
		s.ElementsMatch(expectedCVEIDs, fetchedCVEIDs)
	})
}

func (s *cveV2DataStoreSACTestSuite) TestSACImageCVESearchRawCVEs() {
	s.runImageTest("TestSACImageCVESearchRawCVEs", func(c cveTestCase) {

		testCtx := s.imageTestContexts[c.contextKey]
		results, err := s.imageCVEStore.SearchRawImageCVEs(testCtx, nil)
		s.NoError(err)
		expectedCVEIDs := make([]string, 0, len(c.expectedCVEFound))
		for name, visible := range c.expectedCVEFound {
			if visible {
				expectedCVEIDs = append(expectedCVEIDs, name)
			}
		}

		fetchedCVEIDs := make([]string, 0, len(results))
		for _, result := range results {
			fetchedCVEIDs = append(fetchedCVEIDs, result.Id)
		}
		s.ElementsMatch(expectedCVEIDs, fetchedCVEIDs)
	})
}

func (s *cveV2DataStoreSACTestSuite) runImageTest(testName string, testFunc func(c cveTestCase)) {
	err := s.testGraphDatastore.PushImageToVulnerabilitiesGraph()
	defer s.cleanImageToVulnerabilitiesGraph()
	s.Require().NoError(err)

	imageGraphBefore := graphDBTestUtils.GetImageGraph(
		sac.WithAllAccess(context.Background()),
		s.T(),
		s.testGraphDatastore.GetPostgresPool(),
	)

	failed := false
	for _, c := range imageCVETestCases {
		caseSucceeded := s.Run(c.contextKey, func() {
			// When triggered in parallel, most tests fail.
			// TearDownTest is executed before the sub-tests.
			// See https://github.com/stretchr/testify/issues/934
			// s.T().Parallel()
			testFunc(c)
		})
		if !caseSucceeded {
			failed = true
		}
	}
	if failed {
		log.Infof("%s failed, dumping DB content.", testName)
		imageGraphBefore.Log()
	}
}
