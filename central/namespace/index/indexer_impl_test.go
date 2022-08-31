package index

import (
	"context"
	"testing"

	"github.com/blevesearch/bleve"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/suite"
)

var (
	ctx = sac.WithAllAccess(context.Background())
)

func TestNamespaceIndex(t *testing.T) {
	suite.Run(t, new(NamespaceIndexTestSuite))
}

type NamespaceIndexTestSuite struct {
	suite.Suite

	bleveIndex bleve.Index
	indexer    Indexer
}

func (suite *NamespaceIndexTestSuite) SetupTest() {
	var err error
	suite.bleveIndex, err = globalindex.MemOnlyIndex()
	suite.Require().NoError(err)

	suite.indexer = New(suite.bleveIndex)
}

func (suite *NamespaceIndexTestSuite) TearDownTest() {
	suite.NoError(suite.bleveIndex.Close())
}

func (suite *NamespaceIndexTestSuite) TestIndexing() {
	ns := &storage.NamespaceMetadata{
		Id:   "namespace1",
		Name: "namespace1",
	}

	suite.NoError(suite.indexer.AddNamespaceMetadata(ns))

	q := search.NewQueryBuilder().AddStrings(search.Namespace, "namespace1").ProtoQuery()
	results, err := suite.indexer.Search(ctx, q)
	suite.NoError(err)
	suite.Len(results, 1)
}
