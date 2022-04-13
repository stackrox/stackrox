package index

import (
	"testing"

	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/document"
	deploymentIndex "github.com/stackrox/stackrox/central/deployment/index"
	"github.com/stackrox/stackrox/central/globalindex"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/pkg/fixtures"
	"github.com/stackrox/stackrox/pkg/search"
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
	suite.deploymentIndexer = deploymentIndex.New(tmpIndex, tmpIndex)

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

	fixtureImage := fixtures.LightweightDeploymentImage()
	fixtureImage.Id = "FAKENEWSHA"

	labeledImage := fixtures.LightweightDeploymentImage()
	labeledImage.Id = "LABELEDIMAGE"
	fixtureImage.Metadata.V1.Labels = map[string]string{
		"required-label": "required-value",
	}

	suite.NoError(suite.deploymentIndexer.AddDeployment(secondDeployment))
	suite.NoError(suite.indexer.AddImage(fixtureImage))
	suite.NoError(suite.indexer.AddImage(labeledImage))

	for _, img := range fixtures.DeploymentImages() {
		suite.NoError(suite.indexer.AddImage(img))
	}
}

func (suite *ImageIndexTestSuite) TearDownSuite() {
	suite.NoError(suite.bleveIndex.Close())
}

func (suite *ImageIndexTestSuite) TestSearchImages() {
	// No filter on either => should return everything.
	results, err := suite.indexer.Search(search.EmptyQuery())
	suite.NoError(err)
	suite.Len(results, 4)

	// Filter on only image properties => should work as expected.
	q := search.NewQueryBuilder().AddStrings(search.ImageRegistry, "docker.io").ProtoQuery()
	results, err = suite.indexer.Search(q)
	suite.NoError(err)
	suite.Len(results, 3)

	q = search.NewQueryBuilder().AddStrings(search.ImageLabel, "r/required-label.*=").ProtoQuery()
	results, err = suite.indexer.Search(q)
	suite.NoError(err)
	suite.Len(results, 1)

	q = search.NewQueryBuilder().AddStrings(search.ImageLabel, "r/required-label.*=!r/required-value.*").ProtoQuery()
	results, err = suite.indexer.Search(q)
	suite.NoError(err)
	suite.Len(results, 0)

	q = search.NewQueryBuilder().AddStrings(search.ImageLabel, "!required-label=").ProtoQuery()
	results, err = suite.indexer.Search(q)
	suite.NoError(err)
	suite.Len(results, 3)
}

func (suite *ImageIndexTestSuite) TestMapping() {
	wrapper := &imageWrapper{
		Image: fixtures.GetImage(),
		Type:  v1.SearchCategory_IMAGES.String(),
	}

	doc := document.NewDocument(wrapper.GetId())
	suite.NoError(suite.bleveIndex.Mapping().MapDocument(doc, wrapper))

	docNew, err := suite.indexer.(*indexerImpl).optimizedMapDocument(wrapper)
	suite.NoError(err)

	suite.ElementsMatch(doc.Fields, docNew.Fields)
}
