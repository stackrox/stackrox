package index

import (
	"testing"

	"github.com/blevesearch/bleve"
	deploymentIndex "github.com/stackrox/rox/central/deployment/index"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/suite"
)

const (
	fakeClusterName = "FAKE CLUSTER NAME"
)

func TestImageIndex(t *testing.T) {
	suite.Run(t, new(ImageIndexTestSuite))
}

type ImageIndexTestSuite struct {
	suite.Suite

	bleveIndex        bleve.Index
	deploymentIndexer deploymentIndex.Indexer

	indexer Indexer
}

func (suite *ImageIndexTestSuite) SetupSuite() {
	tmpIndex, err := globalindex.TempInitializeIndices("")
	suite.Require().NoError(err)

	suite.bleveIndex = tmpIndex
	suite.deploymentIndexer = deploymentIndex.New(tmpIndex)

	suite.indexer = New(tmpIndex)

	suite.NoError(suite.deploymentIndexer.AddDeployment(fixtures.GetDeployment()))

	// The following is tightly coupled to the fixtures.GetDeployment() object having
	// two containers, the first with docker.io as the registry and the second with stackrox.io.
	// If you change the fixtures, the tests below will break!
	secondDeployment := fixtures.GetDeployment()
	secondDeployment.Id = "FAKESECONDID"
	secondDeployment.ClusterName = fakeClusterName
	secondDeployment.Containers = fixtures.GetDeployment().GetContainers()[:1]
	secondDeployment.Containers[0].Image.Id = "FAKENEWSHA"

	suite.NoError(suite.deploymentIndexer.AddDeployment(secondDeployment))
	suite.NoError(suite.indexer.AddImage(secondDeployment.GetContainers()[0].GetImage()))

	for _, c := range fixtures.GetDeployment().GetContainers() {
		suite.NoError(suite.indexer.AddImage(c.GetImage()))
	}
}

func (suite *ImageIndexTestSuite) TeardownSuite() {
	suite.bleveIndex.Close()
}

func (suite *ImageIndexTestSuite) TestSearchImages() {
	// No filter on either => should return everything.
	results, err := suite.indexer.SearchImages(search.EmptyQuery())
	suite.NoError(err)
	suite.Len(results, 3)

	// Filter on a deployment property.
	q := search.NewQueryBuilder().AddStrings(search.Cluster, "prod cluster").ProtoQuery()
	results, err = suite.indexer.SearchImages(q)
	suite.NoError(err)
	suite.Len(results, 2)

	// Filter on both deployment and image properties => should return intersection.
	q = search.NewQueryBuilder().AddStrings(search.Cluster, "prod cluster").AddStrings(search.ImageRegistry, "docker.io").ProtoQuery()
	results, err = suite.indexer.SearchImages(q)
	suite.NoError(err)
	suite.Len(results, 1)

	// Filter on only image properties => should work as expected.
	q = search.NewQueryBuilder().AddStrings(search.ImageRegistry, "docker.io").ProtoQuery()
	results, err = suite.indexer.SearchImages(q)
	suite.NoError(err)
	suite.Len(results, 2)
}
