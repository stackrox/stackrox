package datastoretest

import (
	"testing"

	imageCVEDataStore "github.com/stackrox/rox/central/cve/image/datastore"
	nodeCVEDataStore "github.com/stackrox/rox/central/cve/node/datastore"
	"github.com/stackrox/rox/central/dackbox/testutils"
	"github.com/stretchr/testify/suite"
)

func TestCVEDataStoreSAC(t *testing.T) {
	suite.Run(t, new(cveDataStoreSACTestSuite))
}

type cveDataStoreSACTestSuite struct {
	suite.Suite

	dackboxTestStore testutils.DackboxTestDataStore
	imageCVEStore    imageCVEDataStore.DataStore
	nodeCVEStore     nodeCVEDataStore.DataStore
}

func (s *cveDataStoreSACTestSuite) SetupSuite() {

}

func (s *cveDataStoreSACTestSuite) TearDownSuite() {

}

func (s *cveDataStoreSACTestSuite) TestSACImageCVEExistsSingleScopeOnly() {

}
func (s *cveDataStoreSACTestSuite) TestSACImageCVEExistsSharedAcrossComponents() {

}
func (s *cveDataStoreSACTestSuite) TestSACImageCVEExistsFromSharedComponent() {

}

func (s *cveDataStoreSACTestSuite) TestSACImageCVEGetSingleScopeOnly() {

}
func (s *cveDataStoreSACTestSuite) TestSACImageCVEGetSharedAcrossComponents() {

}
func (s *cveDataStoreSACTestSuite) TestSACImageCVEGetFromSharedComponent() {

}

func (s *cveDataStoreSACTestSuite) TestSACImageCVEGetBatch() {

}

func (s *cveDataStoreSACTestSuite) TestSACImageCVECount() {

}

func (s *cveDataStoreSACTestSuite) TestSACImageCVESearch() {

}

func (s *cveDataStoreSACTestSuite) TestSACImageCVESearchCVEs() {

}

func (s *cveDataStoreSACTestSuite) TestSACImageCVESearchRawCVEs() {

}

func (s *cveDataStoreSACTestSuite) TestSACImageCVESuppress() {

}

func (s *cveDataStoreSACTestSuite) TestSACImageCVEUnsuppress() {

}

func (s *cveDataStoreSACTestSuite) TestSACEnrichImageWithSuppressedCVEs() {

}

func (s *cveDataStoreSACTestSuite) TestSACNodeCVEExistsSingleScopeOnly() {

}

func (s *cveDataStoreSACTestSuite) TestSACNodeCVEExistsSharedAcrossComponents() {

}
func (s *cveDataStoreSACTestSuite) TestSACNodeCVEExistsFromSharedComponent() {

}

func (s *cveDataStoreSACTestSuite) TestSACNodeCVEGetSingleScopeOnly() {

}
func (s *cveDataStoreSACTestSuite) TestSACNodeCVEGetSharedAcrossComponents() {

}
func (s *cveDataStoreSACTestSuite) TestSACNodeCVEGetFromSharedComponent() {

}

func (s *cveDataStoreSACTestSuite) TestSACNodeCVEGetBatch() {

}

func (s *cveDataStoreSACTestSuite) TestSACNodeCVESearch() {

}

func (s *cveDataStoreSACTestSuite) TestSACNodeCVESearchCVEs() {

}

func (s *cveDataStoreSACTestSuite) TestSACNodeCVESearchRawCVEs() {

}

func (s *cveDataStoreSACTestSuite) TestSACNodeCVESuppress() {

}

func (s *cveDataStoreSACTestSuite) TestSACNodeCVEUnsuppress() {

}

func (s *cveDataStoreSACTestSuite) TestSACEnrichNodeWithSuppressedCVEs() {

}
