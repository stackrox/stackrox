package index

import (
	"testing"

	"github.com/blevesearch/bleve"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stretchr/testify/suite"
)

func TestPolicyIndex(t *testing.T) {
	suite.Run(t, new(PolicyIndexTestSuite))
}

type PolicyIndexTestSuite struct {
	suite.Suite

	bleveIndex bleve.Index

	indexer Indexer
}

func (suite *PolicyIndexTestSuite) SetupSuite() {
	tmpIndex, err := globalindex.TempInitializeIndices("")
	suite.Require().NoError(err)

	suite.bleveIndex = tmpIndex
	suite.indexer = New(tmpIndex)

	policy := fixtures.GetPolicy()
	suite.NoError(suite.indexer.AddPolicy(policy))
}

func (suite *PolicyIndexTestSuite) TeardownSuite() {
	suite.bleveIndex.Close()
}
