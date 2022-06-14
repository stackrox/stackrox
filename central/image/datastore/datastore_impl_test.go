package datastore

import (
	"context"
	"testing"

	"github.com/blevesearch/bleve"
	"github.com/gogo/protobuf/types"
	"github.com/golang/mock/gomock"
	componentCVEEdgeDackBox "github.com/stackrox/rox/central/componentcveedge/dackbox"
	componentCVEEdgeIndex "github.com/stackrox/rox/central/componentcveedge/index"
	cveDackbox "github.com/stackrox/rox/central/cve/dackbox"
	cveIndex "github.com/stackrox/rox/central/cve/index"
	"github.com/stackrox/rox/central/globalindex"
	imageDackBox "github.com/stackrox/rox/central/image/dackbox"
	imageIndex "github.com/stackrox/rox/central/image/index"
	componentDackBox "github.com/stackrox/rox/central/imagecomponent/dackbox"
	componentIndex "github.com/stackrox/rox/central/imagecomponent/index"
	imageComponentEdgeDackBox "github.com/stackrox/rox/central/imagecomponentedge/dackbox"
	imageComponentEdgeIndex "github.com/stackrox/rox/central/imagecomponentedge/index"
	"github.com/stackrox/rox/central/ranking"
	mockRisks "github.com/stackrox/rox/central/risk/datastore/mocks"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/dackbox"
	"github.com/stackrox/rox/pkg/dackbox/indexer"
	"github.com/stackrox/rox/pkg/dackbox/utils/queue"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/sac"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/scoped"
	"github.com/stackrox/rox/pkg/testutils/rocksdbtest"
	"github.com/stretchr/testify/suite"
)

func TestImageDataStore(t *testing.T) {
	suite.Run(t, new(ImageDataStoreTestSuite))
}

type ImageDataStoreTestSuite struct {
	suite.Suite

	db        *rocksdb.RocksDB
	index     bleve.Index
	indexQ    queue.WaitableQueue
	datastore DataStore

	mockRisk *mockRisks.MockDataStore
}

func (suite *ImageDataStoreTestSuite) SetupSuite() {
	if features.PostgresDatastore.Enabled() {
		suite.T().Skip("Skip non-postgres store tests if postgres is enabled")
		suite.T().SkipNow()
	}

	suite.db = rocksdbtest.RocksDBForT(suite.T())

	suite.indexQ = queue.NewWaitableQueue()

	dacky, err := dackbox.NewRocksDBDackBox(suite.db, suite.indexQ, []byte("graph"), []byte("dirty"), []byte("valid"))
	suite.Require().NoError(err, "failed to create dackbox")

	suite.index, err = globalindex.MemOnlyIndex()
	suite.Require().NoError(err, "failed to create bleve index")

	reg := indexer.NewWrapperRegistry()
	indexer.NewLazy(suite.indexQ, reg, suite.index, dacky.AckIndexed).Start()
	reg.RegisterWrapper(cveDackbox.Bucket, cveIndex.Wrapper{})
	reg.RegisterWrapper(componentDackBox.Bucket, componentIndex.Wrapper{})
	reg.RegisterWrapper(componentCVEEdgeDackBox.Bucket, componentCVEEdgeIndex.Wrapper{})
	reg.RegisterWrapper(imageDackBox.Bucket, imageIndex.Wrapper{})
	reg.RegisterWrapper(imageComponentEdgeDackBox.Bucket, imageComponentEdgeIndex.Wrapper{})

	suite.mockRisk = mockRisks.NewMockDataStore(gomock.NewController(suite.T()))

	suite.datastore = New(dacky, concurrency.NewKeyFence(), suite.index, suite.index, false, suite.mockRisk, ranking.ImageRanker(), ranking.ComponentRanker())
}

func (suite *ImageDataStoreTestSuite) TearDownSuite() {
	rocksdbtest.TearDownRocksDB(suite.db)
	suite.Require().NoError(suite.index.Close())
}

func (suite *ImageDataStoreTestSuite) TestSearch() {
	image := getTestImage("id1")

	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowFixedScopes(
		sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
		sac.ResourceScopeKeys(resources.Image),
	))

	// Upsert image.
	suite.Require().NoError(suite.datastore.UpsertImage(ctx, image))

	// Ensure the CVEs are indexed.
	indexingDone := concurrency.NewSignal()
	suite.indexQ.PushSignal(&indexingDone)
	indexingDone.Wait()

	// Basic unscoped search.
	results, err := suite.datastore.Search(ctx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.Len(results, 1)

	scopedCtx := scoped.Context(ctx, scoped.Scope{
		ID:    image.GetId(),
		Level: v1.SearchCategory_IMAGES,
	})

	// Basic scoped search.
	results, err = suite.datastore.Search(scopedCtx, pkgSearch.EmptyQuery())
	suite.Require().NoError(err)
	suite.Len(results, 1)

	// Search Images.
	images, err := suite.datastore.SearchRawImages(scopedCtx, pkgSearch.EmptyQuery())
	suite.Require().NoError(err)
	suite.NotNil(images)
	suite.Len(images, 1)
	for _, component := range image.GetScan().GetComponents() {
		for _, cve := range component.GetVulns() {
			cve.FirstSystemOccurrence = images[0].GetLastUpdated()
			cve.FirstImageOccurrence = images[0].GetLastUpdated()
			cve.VulnerabilityType = storage.EmbeddedVulnerability_UNKNOWN_VULNERABILITY
			cve.VulnerabilityTypes = []storage.EmbeddedVulnerability_VulnerabilityType{storage.EmbeddedVulnerability_IMAGE_VULNERABILITY}
		}
	}
	suite.Equal(image, images[0])

	// Upsert new image.
	newImage := getTestImage("id2")
	newImage.GetScan().Components = append(newImage.GetScan().GetComponents(), &storage.EmbeddedImageScanComponent{
		Name:    "comp3",
		Version: "ver1",
		Vulns: []*storage.EmbeddedVulnerability{
			{
				Cve:               "cve3",
				VulnerabilityType: storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
			},
		},
	})
	suite.Require().NoError(suite.datastore.UpsertImage(ctx, newImage))

	// Ensure the CVEs are indexed.
	indexingDone = concurrency.NewSignal()
	suite.indexQ.PushSignal(&indexingDone)
	indexingDone.Wait()

	// Search multiple images.
	images, err = suite.datastore.SearchRawImages(ctx, pkgSearch.EmptyQuery())
	suite.Require().NoError(err)
	suite.Len(images, 2)

	// Search for just one image.
	images, err = suite.datastore.SearchRawImages(scopedCtx, pkgSearch.EmptyQuery())
	suite.Require().NoError(err)
	suite.Len(images, 1)

	// Search by CVE.
	scopedCtx = scoped.Context(ctx, scoped.Scope{
		ID:    "cve1",
		Level: v1.SearchCategory_VULNERABILITIES,
	})
	images, err = suite.datastore.SearchRawImages(scopedCtx, pkgSearch.EmptyQuery())
	suite.Require().NoError(err)
	suite.Len(images, 2)
	scopedCtx = scoped.Context(ctx, scoped.Scope{
		ID:    "cve3",
		Level: v1.SearchCategory_VULNERABILITIES,
	})
	results, err = suite.datastore.Search(scopedCtx, pkgSearch.EmptyQuery())
	suite.Require().NoError(err)
	suite.Len(results, 1)
	suite.Equal("id2", results[0].ID)
}

func getTestImage(id string) *storage.Image {
	return &storage.Image{
		Id: id,
		Scan: &storage.ImageScan{
			OperatingSystem: "blah",
			ScanTime:        types.TimestampNow(),
			Components: []*storage.EmbeddedImageScanComponent{
				{
					Name:    "comp1",
					Version: "ver1",
					Vulns:   []*storage.EmbeddedVulnerability{},
				},
				{
					Name:    "comp1",
					Version: "ver2",
					Vulns: []*storage.EmbeddedVulnerability{
						{
							Cve:               "cve1",
							VulnerabilityType: storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
						},
						{
							Cve:               "cve2",
							VulnerabilityType: storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
							SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
								FixedBy: "ver3",
							},
						},
					},
				},
				{
					Name:    "comp2",
					Version: "ver1",
					Vulns: []*storage.EmbeddedVulnerability{
						{
							Cve:               "cve1",
							VulnerabilityType: storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
							SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
								FixedBy: "ver2",
							},
						},
						{
							Cve:               "cve2",
							VulnerabilityType: storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
						},
					},
				},
			},
		},
		RiskScore: 30,
		Priority:  1,
	}
}
