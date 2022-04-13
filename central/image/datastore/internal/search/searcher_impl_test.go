package search

import (
	"context"
	"testing"

	"github.com/blevesearch/bleve"
	deployment "github.com/stackrox/stackrox/central/deployment/dackbox"
	deploymentIndexer "github.com/stackrox/stackrox/central/deployment/index"
	"github.com/stackrox/stackrox/central/globalindex"
	image "github.com/stackrox/stackrox/central/image/dackbox"
	imageStore "github.com/stackrox/stackrox/central/image/datastore/internal/store/dackbox"
	imageIndexer "github.com/stackrox/stackrox/central/image/index"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/concurrency"
	"github.com/stackrox/stackrox/pkg/dackbox"
	"github.com/stackrox/stackrox/pkg/dbhelper"
	"github.com/stackrox/stackrox/pkg/rocksdb"
	"github.com/stackrox/stackrox/pkg/sac"
	"github.com/stackrox/stackrox/pkg/search"
	"github.com/stackrox/stackrox/pkg/testutils/rocksdbtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

func TestImageSearcher(t *testing.T) {
	suite.Run(t, new(ImageSearcherTestSuite))
}

type ImageSearcherTestSuite struct {
	suite.Suite

	db                *rocksdb.RocksDB
	dacky             *dackbox.DackBox
	bleveIndex        bleve.Index
	deploymentIndexer deploymentIndexer.Indexer
	imageIndexer      imageIndexer.Indexer
	searcher          Searcher
}

func (suite *ImageSearcherTestSuite) SetupSuite() {
	tmpIndex, err := globalindex.TempInitializeIndices("")
	suite.Require().NoError(err)

	suite.bleveIndex = tmpIndex
	suite.deploymentIndexer = deploymentIndexer.New(tmpIndex, tmpIndex)
	suite.imageIndexer = imageIndexer.New(tmpIndex)

	db, err := rocksdb.NewTemp("temp")
	suite.Require().NoError(err, "failed to create DB")
	suite.db = db

	dacky, err := dackbox.NewRocksDBDackBox(suite.db, nil, []byte{}, []byte{}, []byte{})
	suite.Require().NoError(err, "failed to create dackbox")
	suite.dacky = dacky

	suite.searcher = New(imageStore.New(suite.dacky, concurrency.NewKeyFence(), true), suite.dacky, nil, nil, nil, nil, suite.imageIndexer, suite.deploymentIndexer, nil)

	d1ns1 := getDeployment("d1n1", "n1", "c1", 1)
	d2ns1 := getDeployment("d2n1", "n1", "c1", 2)
	d1ns2 := getDeployment("d1n2", "n2", "c1", 2)

	img1 := getImage("img1", 1)
	img2 := getImage("img2", 2)
	img3 := getImage("img3", 3)
	img4 := getImage("img4", 4)
	img5 := getImage("img5", 5)
	img6 := getImage("img6", 6)

	suite.Require().NoError(suite.deploymentIndexer.AddDeployments([]*storage.Deployment{d1ns1, d2ns1, d1ns2}))
	suite.Require().NoError(suite.imageIndexer.AddImages([]*storage.Image{img1, img2, img3, img4, img5, img6}))

	generateGraph(suite.T(), suite.dacky, map[string][]string{
		"d1n1": {"img1", "img2", "img3", "img4"},
		"d2n1": {"img1", "img3"},
		"d1n2": {"img1", "img2", "img5", "img6"},
	})
}

func (suite *ImageSearcherTestSuite) TearDownSuite() {
	suite.NoError(suite.bleveIndex.Close())
	rocksdbtest.TearDownRocksDB(suite.db)
}

func (suite *ImageSearcherTestSuite) TestRiskOrdering() {
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowAllAccessScopeChecker())

	// Sort by priority aka high risk to low risk.
	q := &v1.Query{
		Pagination: &v1.QueryPagination{
			SortOptions: []*v1.QuerySortOption{
				{
					Field: search.ImagePriority.String(),
				},
			},
		},
	}
	results, err := suite.searcher.Search(ctx, q)
	suite.NoError(err)
	suite.Equal([]string{"img6", "img5", "img4", "img3", "img2", "img1"}, search.ResultsToIDs(results))

	// Get images in namespace 'n1' sorted by priority in reverse order.
	q = search.NewQueryBuilder().AddExactMatches(search.Namespace, "n1").ProtoQuery()
	q.Pagination = &v1.QueryPagination{
		SortOptions: []*v1.QuerySortOption{
			{
				Field:    search.ImagePriority.String(),
				Reversed: true,
			},
		},
	}
	results, err = suite.searcher.Search(ctx, q)
	suite.NoError(err)
	suite.Equal([]string{"img1", "img2", "img3", "img4"}, search.ResultsToIDs(results))

	// Get images in namespace 'n2' sorted by priority.
	q = search.NewQueryBuilder().AddExactMatches(search.Namespace, "n2").ProtoQuery()
	q.Pagination = &v1.QueryPagination{
		SortOptions: []*v1.QuerySortOption{
			{
				Field: search.ImagePriority.String(),
			},
		},
	}
	results, err = suite.searcher.Search(ctx, q)
	suite.NoError(err)
	suite.Equal([]string{"img6", "img5", "img2", "img1"}, search.ResultsToIDs(results))

	// Sort by namespace.
	q = search.EmptyQuery()
	q.Pagination = &v1.QueryPagination{
		SortOptions: []*v1.QuerySortOption{
			{
				Field: search.Namespace.String(),
			},
		},
	}
	results, err = suite.searcher.Search(ctx, q)
	suite.NoError(err)
	ids := search.ResultsToIDs(results)
	suite.ElementsMatch([]string{"img1", "img2", "img3", "img4"}, ids[:4])
	suite.ElementsMatch([]string{"img5", "img6"}, ids[4:])
}

func getDeployment(id, namespace, cluster string, riskScore float32) *storage.Deployment {
	return &storage.Deployment{
		Id:        id,
		Name:      id,
		Namespace: namespace,
		ClusterId: cluster,
		RiskScore: riskScore,
	}
}

func getImage(id string, riskScore float32) *storage.Image {
	return &storage.Image{
		Id:        id,
		RiskScore: riskScore,
	}
}

func generateGraph(t *testing.T, dacky *dackbox.DackBox, links map[string][]string) {
	view, err := dacky.NewTransaction()
	assert.NoError(t, err)
	defer view.Discard()

	for from, tos := range links {
		for _, to := range tos {
			view.Graph().AddRefs(dbhelper.GetBucketKey(deployment.Bucket, []byte(from)), dbhelper.GetBucketKey(image.Bucket, []byte(to)))
		}
	}
	assert.NoError(t, view.Commit(), "commit should have succeeded")
}
