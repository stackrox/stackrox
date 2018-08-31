package index

import (
	"testing"

	"github.com/blevesearch/bleve"
	"github.com/stackrox/rox/central/globalindex"
	imageIndex "github.com/stackrox/rox/central/image/index"
	secretIndex "github.com/stackrox/rox/central/secret/index"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/suite"
)

const (
	fakeID   = "FAKEID"
	fakeName = "FAKENAME"
)

func TestDeploymentIndex(t *testing.T) {
	suite.Run(t, new(DeploymentIndexTestSuite))
}

type DeploymentIndexTestSuite struct {
	suite.Suite

	bleveIndex bleve.Index

	indexer Indexer
}

func (suite *DeploymentIndexTestSuite) SetupSuite() {
	tmpIndex, err := globalindex.TempInitializeIndices("")
	suite.Require().NoError(err)

	suite.bleveIndex = tmpIndex
	suite.indexer = New(tmpIndex)

	deployment := fixtures.GetDeployment()
	suite.NoError(suite.indexer.AddDeployment(deployment))
	suite.NoError(suite.indexer.AddDeployment(&v1.Deployment{Id: fakeID, Name: fakeName}))

	imageIndexer := imageIndex.New(tmpIndex)
	imageIndexer.AddImage(fixtures.GetImage())

	secretIndexer := secretIndex.New(tmpIndex)
	secretIndexer.UpsertSecret(&v1.Secret{
		Id: "ABC",
	})
}

func (suite *DeploymentIndexTestSuite) TeardownSuite() {
	suite.bleveIndex.Close()
}

func (suite *DeploymentIndexTestSuite) TestDeploymentsQuery() {
	cases := []struct {
		fieldValues map[string]string
		expectedIDs []string
	}{
		{
			fieldValues: map[string]string{search.DeploymentName: "nginx"},
			expectedIDs: []string{fixtures.GetDeployment().GetId()},
		},
		{
			fieldValues: map[string]string{search.DeploymentName: "!nginx"},
			expectedIDs: []string{fakeID},
		},
		{
			fieldValues: map[string]string{search.Label: "com.docker.stack.namespace=prevent"},
			expectedIDs: []string{fixtures.GetDeployment().GetId()},
		},
		{
			fieldValues: map[string]string{search.DeploymentName: "!nginx", search.Label: "com.docker.stack.namespace=prevent"},
			expectedIDs: []string{},
		},
		{
			fieldValues: map[string]string{search.DeploymentName: "!nomatch", search.Label: "com.docker.stack.namespace=/.*"},
			expectedIDs: []string{fixtures.GetDeployment().GetId()},
		},
		{
			fieldValues: map[string]string{search.DeploymentName: "!nomatch"},
			expectedIDs: []string{fixtures.GetDeployment().GetId(), fakeID},
		},
		{
			fieldValues: map[string]string{search.DeploymentName: "!nomatch", search.ImageRegistry: "stackrox"},
			expectedIDs: []string{fixtures.GetDeployment().GetId()},
		},
		{
			fieldValues: map[string]string{search.DeploymentName: "!nomatch", search.ImageRegistry: "nonexistent"},
			expectedIDs: []string{},
		},
	}

	for _, c := range cases {
		qb := search.NewQueryBuilder()
		for field, value := range c.fieldValues {
			qb.AddStrings(field, value)
		}
		results, err := suite.indexer.SearchDeployments(qb.ProtoQuery())
		suite.NoError(err)

		resultIDs := make([]string, 0, len(results))
		for _, r := range results {
			resultIDs = append(resultIDs, r.ID)
		}
		suite.ElementsMatch(resultIDs, c.expectedIDs, "Failed test case %#v", c)
	}
}
