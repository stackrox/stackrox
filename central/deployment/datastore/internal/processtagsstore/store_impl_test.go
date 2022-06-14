package processtagsstore

import (
	"fmt"
	"testing"

	"github.com/stackrox/stackrox/central/analystnotes"
	"github.com/stackrox/stackrox/pkg/testutils"
	"github.com/stretchr/testify/suite"
	bolt "go.etcd.io/bbolt"
)

func getDeploymentID(deploymentSeed int) string {
	return fmt.Sprintf("DEPLOY:%d", deploymentSeed)
}

func getKey(deploymentSeed, containerSeed int) *analystnotes.ProcessNoteKey {
	key := &analystnotes.ProcessNoteKey{
		DeploymentID:  getDeploymentID(deploymentSeed),
		ContainerName: fmt.Sprintf("CONTAINER%d", containerSeed),
		ExecFilePath:  "EXEC",
	}
	if deploymentSeed%2 == 0 {
		key.Args = "ARGS"
	}
	return key
}

func TestStore(t *testing.T) {
	suite.Run(t, new(StoreTestSuite))
}

type StoreTestSuite struct {
	suite.Suite

	db *bolt.DB

	store Store
}

func (suite *StoreTestSuite) SetupTest() {
	suite.db = testutils.DBForSuite(suite)
	suite.store = New(suite.db)
}

func (suite *StoreTestSuite) TearDownTest() {
	testutils.TearDownDB(suite.db)
}

func (suite *StoreTestSuite) mustGetTags(deploymentSeed, containerSeed int) []string {
	tags, err := suite.store.GetTagsForProcessKey(getKey(deploymentSeed, containerSeed))
	suite.Require().NoError(err)
	return tags
}

func (suite *StoreTestSuite) TestStore() {
	suite.Empty(suite.mustGetTags(1, 1))
	suite.NoError(suite.store.RemoveProcessTags(getKey(1, 1), []string{"blah"}))

	suite.NoError(suite.store.UpsertProcessTags(getKey(1, 1), []string{"one", "two", "three"}))

	suite.Equal([]string{"one", "three", "two"}, suite.mustGetTags(1, 1))

	suite.NoError(suite.store.RemoveProcessTags(getKey(1, 1), []string{"blah", "two"}))
	suite.Equal([]string{"one", "three"}, suite.mustGetTags(1, 1))

	suite.NoError(suite.store.UpsertProcessTags(getKey(1, 1), []string{"three", "four"}))
	suite.Equal([]string{"four", "one", "three"}, suite.mustGetTags(1, 1))

	suite.NoError(suite.store.UpsertProcessTags(getKey(0, 2), []string{"blah"}))
	suite.Equal([]string{"blah"}, suite.mustGetTags(0, 2))

	suite.NoError(suite.store.UpsertProcessTags(getKey(1, 2), []string{"five", "four"}))

	// Test walk
	var seenTags []string
	suite.NoError(suite.store.WalkTagsForDeployment(getDeploymentID(1), func(tag string) bool {
		seenTags = append(seenTags, tag)
		return true
	}))
	suite.ElementsMatch([]string{"one", "three", "four", "five"}, seenTags)

	var seenTag string
	suite.NoError(suite.store.WalkTagsForDeployment(getDeploymentID(1), func(tag string) bool {
		suite.Require().Empty(seenTag)
		seenTag = tag
		return false
	}))
	suite.Contains([]string{"one", "three", "four", "five"}, seenTag)

	suite.NoError(suite.store.RemoveProcessTags(getKey(1, 1), []string{"one", "three", "four"}))
	suite.Empty(suite.mustGetTags(1, 1))

}
