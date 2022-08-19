package index

import (
	"testing"

	"github.com/blevesearch/bleve/v2"
	"github.com/stackrox/rox/central/activecomponent/converter"
	"github.com/stackrox/rox/central/activecomponent/dackbox"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/suite"
)

func TestActiveComponentIndex(t *testing.T) {
	suite.Run(t, new(ActiveComponentIndexTestSuite))
}

type ActiveComponentIndexTestSuite struct {
	suite.Suite

	bleveIndex bleve.Index
	indexer    Indexer
	wrapper    Wrapper
}

func (suite *ActiveComponentIndexTestSuite) SetupTest() {
	var err error
	suite.bleveIndex, err = globalindex.MemOnlyIndex()
	suite.Require().NoError(err)

	suite.indexer = New(suite.bleveIndex)
	suite.wrapper = Wrapper{}
}

func (suite *ActiveComponentIndexTestSuite) TearDownTest() {
	suite.NoError(suite.bleveIndex.Close())
}

func (suite *ActiveComponentIndexTestSuite) TestIndexing() {
	containerName := "containerName"
	imageID := "SHA:232399292"
	deploymentID := "deployID"
	componentID := "component:id"
	id := converter.ComposeID(deploymentID, componentID)
	ac := &storage.ActiveComponent{
		Id: id,
		ActiveContextsSlice: []*storage.ActiveComponent_ActiveContext{
			{
				ContainerName: containerName,
				ImageId:       imageID,
			},
		},
	}

	q := search.NewQueryBuilder().AddExactMatches(search.ImageSHA, imageID).ProtoQuery()

	results, err := suite.indexer.Search(q)
	suite.NoError(err)
	suite.Len(results, 0)

	suite.NoError(suite.addComponent(ac))
	results, err = suite.indexer.Search(q)
	suite.NoError(err)
	suite.Len(results, 1)
}

func (suite *ActiveComponentIndexTestSuite) addComponent(ac *storage.ActiveComponent) error {
	id, value := suite.wrapper.Wrap(dackbox.KeyFunc(ac), ac)
	return suite.bleveIndex.Index(id, value)
}
