package inmem

import (
	"testing"

	"bitbucket.org/stack-rox/apollo/apollo/db"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestImageRules(t *testing.T) {
	suite.Run(t, new(ImageRulesTestSuite))
}

type ImageRulesTestSuite struct {
	suite.Suite
	*InMemoryStore
}

func (suite *ImageRulesTestSuite) SetupSuite() {
	persistent, err := createBoltDB()
	require.Nil(suite.T(), err)
	suite.InMemoryStore = New(persistent)
}

func (suite *ImageRulesTestSuite) TeardownSuite() {
	suite.Close()
}

func (suite *ImageRulesTestSuite) basicImageRulesTest(updateStore, retrievalStore db.Storage) {
	rule1 := &v1.ImageRule{
		Name:     "rule1",
		Severity: v1.Severity_LOW_SEVERITY,
	}
	err := updateStore.AddImageRule(rule1)
	suite.Nil(err)
	rule2 := &v1.ImageRule{
		Name:     "rule2",
		Severity: v1.Severity_HIGH_SEVERITY,
	}
	err = updateStore.AddImageRule(rule2)
	suite.Nil(err)

	// Verify add is persisted
	imageRules, err := retrievalStore.GetImageRules(&v1.GetImageRulesRequest{})
	suite.Nil(err)
	suite.Equal([]*v1.ImageRule{rule1, rule2}, imageRules)

	// Verify update works
	rule1.Severity = v1.Severity_HIGH_SEVERITY
	err = suite.UpdateImageRule(rule1)
	imageRules, err = retrievalStore.GetImageRules(&v1.GetImageRulesRequest{})
	suite.Nil(err)
	suite.Equal([]*v1.ImageRule{rule1, rule2}, imageRules)

	// Verify deletion is persisted
	err = suite.RemoveImageRule(rule1.Name)
	suite.Nil(err)
	err = suite.RemoveImageRule(rule2.Name)
	suite.Nil(err)
	imageRules, err = retrievalStore.GetImageRules(&v1.GetImageRulesRequest{})
	suite.Nil(err)
	suite.Len(imageRules, 0)
}

func (suite *ImageRulesTestSuite) TestPersistence() {
	suite.basicImageRulesTest(suite.InMemoryStore, suite.persistent)
}

func (suite *ImageRulesTestSuite) TestImageRules() {
	suite.basicImageRulesTest(suite.InMemoryStore, suite.InMemoryStore)
}

func (suite *ImageRulesTestSuite) TestGetImageRulesFilters() {
	rule1 := &v1.ImageRule{
		Name: "rule1",
	}
	err := suite.AddImageRule(rule1)
	suite.Nil(err)
	rule2 := &v1.ImageRule{
		Name: "rule2",
	}
	err = suite.AddImageRule(rule2)
	suite.Nil(err)
	// Get all image rules
	rules, err := suite.GetImageRules(&v1.GetImageRulesRequest{})
	suite.Nil(err)
	suite.Equal([]*v1.ImageRule{rule1, rule2}, rules)

	// Get by ID
	rules, err = suite.GetImageRules(&v1.GetImageRulesRequest{Name: rule1.Name})
	suite.Nil(err)
	suite.Equal([]*v1.ImageRule{rule1}, rules)

	// Cleanup
	err = suite.RemoveImageRule(rule1.Name)
	suite.Nil(err)

	err = suite.RemoveImageRule(rule2.Name)
	suite.Nil(err)
}
