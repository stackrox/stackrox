package dackbox

import (
	"context"
	"testing"

	"github.com/gogo/protobuf/types"
	cveStore "github.com/stackrox/rox/central/cve/store"
	cveDackBoxStore "github.com/stackrox/rox/central/cve/store/dackbox"
	"github.com/stackrox/rox/central/image/datastore/store"
	imageCVEEdgeStore "github.com/stackrox/rox/central/imagecveedge/store"
	imageCVEEdgeDackBox "github.com/stackrox/rox/central/imagecveedge/store/dackbox"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/dackbox"
	"github.com/stackrox/rox/pkg/dackbox/edges"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/testutils/rocksdbtest"
	"github.com/stretchr/testify/suite"
)

func TestImageStore(t *testing.T) {
	suite.Run(t, new(ImageStoreTestSuite))
}

type ImageStoreTestSuite struct {
	suite.Suite

	db    *rocksdb.RocksDB
	dacky *dackbox.DackBox

	store             store.Store
	cveStorage        cveStore.Store
	imageCVEEdgeStore imageCVEEdgeStore.Store
}

func (suite *ImageStoreTestSuite) SetupSuite() {
	var err error

	suite.db, err = rocksdb.NewTemp("reference")
	if err != nil {
		suite.FailNowf("failed to create DB: %+v", err.Error())
	}

	suite.dacky, err = dackbox.NewRocksDBDackBox(suite.db, nil, []byte("graph"), []byte("dirty"), []byte("valid"))
	if err != nil {
		suite.FailNowf("failed to create counter: %+v", err.Error())
	}
	suite.store = New(suite.dacky, concurrency.NewKeyFence(), false)
	suite.cveStorage = cveDackBoxStore.New(suite.dacky, concurrency.NewKeyFence())
	suite.imageCVEEdgeStore = imageCVEEdgeDackBox.New(suite.dacky, concurrency.NewKeyFence())
}

func (suite *ImageStoreTestSuite) TearDownSuite() {
	rocksdbtest.TearDownRocksDB(suite.db)
}

func (suite *ImageStoreTestSuite) TestImages() {
	allAccessCtx := sac.WithAllAccess(context.Background())
	images := []*storage.Image{
		{
			Id: "sha256:sha1",
			Name: &storage.ImageName{
				FullName: "name1",
			},
			Metadata: &storage.ImageMetadata{
				V1: &storage.V1Metadata{
					Created: types.TimestampNow(),
				},
			},
			Scan: &storage.ImageScan{
				ScanTime: types.TimestampNow(),
				Components: []*storage.EmbeddedImageScanComponent{
					{
						Name:    "comp1",
						Version: "ver1",
						HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{
							LayerIndex: 1,
						},
						Vulns: []*storage.EmbeddedVulnerability{},
					},
					{
						Name:    "comp1",
						Version: "ver2",
						HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{
							LayerIndex: 3,
						},
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
						HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{
							LayerIndex: 2,
						},
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
		},
		{
			Id: "sha256:sha2",
			Name: &storage.ImageName{
				FullName: "name2",
			},
		},
	}

	// Test Add
	for _, d := range images {
		suite.NoError(suite.store.Upsert(allAccessCtx, d))
	}

	for _, d := range images {
		got, exists, err := suite.store.Get(allAccessCtx, d.GetId())
		suite.NoError(err)
		suite.True(exists)
		// Upsert sets `createdAt` for every CVE that doesn't already exist in the store, which should be same as (*storage.Image).LastUpdated.
		for _, component := range d.GetScan().GetComponents() {
			for _, vuln := range component.GetVulns() {
				vuln.FirstSystemOccurrence = got.GetLastUpdated()
				vuln.FirstImageOccurrence = got.GetLastUpdated()
				vuln.VulnerabilityType = storage.EmbeddedVulnerability_UNKNOWN_VULNERABILITY
				vuln.VulnerabilityTypes = []storage.EmbeddedVulnerability_VulnerabilityType{storage.EmbeddedVulnerability_IMAGE_VULNERABILITY}
			}
		}
		suite.Equal(d, got)

		listGot, exists, err := suite.store.GetImageMetadata(allAccessCtx, d.GetId())
		suite.NoError(err)
		suite.True(exists)
		suite.Equal(d.GetName().GetFullName(), listGot.GetName().GetFullName())
	}

	// Check that the CVEs were written with the correct timestamp.
	vuln, _, err := suite.cveStorage.Get(allAccessCtx, "cve1")
	suite.NoError(err)
	suite.Equal(images[0].GetLastUpdated(), vuln.GetCreatedAt())
	vuln, _, err = suite.cveStorage.Get(allAccessCtx, "cve2")
	suite.NoError(err)
	suite.Equal(images[0].GetLastUpdated(), vuln.GetCreatedAt())

	// Check that the Image CVE Edges were written with the correct timestamp.
	// This test relies on dackbox that does not require the primary keys.
	imageCVEEdge, _, err := suite.imageCVEEdgeStore.Get(allAccessCtx, edges.EdgeID{ParentID: "sha256:sha1", ChildID: "cve1"}.ToString())
	suite.NoError(err)
	suite.Equal(images[0].GetLastUpdated(), imageCVEEdge.GetFirstImageOccurrence())
	// This test relies on dackbox that does not require the primary keys.
	imageCVEEdge, _, err = suite.imageCVEEdgeStore.Get(allAccessCtx, edges.EdgeID{ParentID: "sha256:sha1", ChildID: "cve1"}.ToString())
	suite.NoError(err)
	suite.Equal(images[0].GetLastUpdated(), imageCVEEdge.GetFirstImageOccurrence())

	// Test Update
	for _, d := range images {
		d.Name.FullName += "1"
	}

	for _, d := range images {
		suite.NoError(suite.store.Upsert(allAccessCtx, d))
	}

	for _, d := range images {
		got, exists, err := suite.store.Get(allAccessCtx, d.GetId())
		suite.NoError(err)
		suite.True(exists)
		suite.Equal(d, got)

		listGot, exists, err := suite.store.GetImageMetadata(allAccessCtx, d.GetId())
		suite.NoError(err)
		suite.True(exists)
		suite.Equal(d.GetName().GetFullName(), listGot.GetName().GetFullName())
	}

	// Test Count
	count, err := suite.store.Count(allAccessCtx)
	suite.NoError(err)
	suite.Equal(len(images), count)

	// Test no update
	cloned := images[0].Clone()
	cloned.Metadata.V1.Created.Seconds = cloned.Metadata.V1.Created.Seconds - 500
	cloned.Scan.ScanTime.Seconds = cloned.Scan.ScanTime.Seconds - 500
	cloned.Name.FullName = "newname"
	suite.NoError(suite.store.Upsert(allAccessCtx, cloned))
	got, exists, err := suite.store.Get(allAccessCtx, cloned.GetId())
	suite.NoError(err)
	suite.True(exists)
	suite.Equal(images[0].GetName().GetFullName(), got.GetName().GetFullName())

	// Test no components and cve update, only image bucket update
	cloned = images[0].Clone()
	cloned.Scan.ScanTime.Seconds = cloned.Scan.ScanTime.Seconds - 500
	cloned.Name.FullName = "newname"
	cloned.Scan.Components = nil
	cloned.RiskScore = 100
	suite.NoError(suite.store.Upsert(allAccessCtx, cloned))
	got, exists, err = suite.store.Get(allAccessCtx, cloned.GetId())
	suite.NoError(err)
	suite.True(exists)
	// Since the metadata is not outdated, image update goes through.
	suite.Equal("newname", got.GetName().GetFullName())
	// The image in store should still have components since it has fresher scan.
	suite.Len(got.GetScan().GetComponents(), len(images[0].GetScan().GetComponents()))
	// Risk score of stored image should be picked up.
	suite.Equal(images[0].GetRiskScore(), got.GetRiskScore())

	// Since imags[0] is updated in store, update the "expected" object
	images[0].LastUpdated = got.GetLastUpdated()
	images[0].Scan.ScanTime.Seconds = cloned.Scan.ScanTime.Seconds
	images[0].Name.FullName = "newname"

	// Test first image occurrence of CVE that is already discovered in system.
	images[1].Scan = &storage.ImageScan{
		Components: []*storage.EmbeddedImageScanComponent{
			{
				Name:    "comp1",
				Version: "ver1",
				HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{
					LayerIndex: 1,
				},
				Vulns: []*storage.EmbeddedVulnerability{
					{
						Cve:               "cve1",
						VulnerabilityType: storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
					},
				},
			},
		},
	}

	suite.NoError(suite.store.Upsert(allAccessCtx, images[1]))

	got, exists, err = suite.store.Get(allAccessCtx, images[1].GetId())
	suite.NoError(err)
	suite.True(exists)
	images[1].GetScan().GetComponents()[0].GetVulns()[0].FirstSystemOccurrence = images[0].GetScan().GetComponents()[1].GetVulns()[0].FirstSystemOccurrence
	images[1].GetScan().GetComponents()[0].GetVulns()[0].FirstImageOccurrence = got.GetLastUpdated()
	images[1].GetScan().GetComponents()[0].GetVulns()[0].VulnerabilityType = storage.EmbeddedVulnerability_UNKNOWN_VULNERABILITY
	images[1].GetScan().GetComponents()[0].GetVulns()[0].VulnerabilityTypes = []storage.EmbeddedVulnerability_VulnerabilityType{storage.EmbeddedVulnerability_IMAGE_VULNERABILITY}
	suite.Equal(images[1], got)

	// Test second occurrence of a CVE in an image
	images[0].GetScan().GetComponents()[0].Vulns = append(images[0].GetScan().GetComponents()[0].Vulns,
		&storage.EmbeddedVulnerability{
			Cve:               "cve1",
			VulnerabilityType: storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
		})

	suite.NoError(suite.store.Upsert(allAccessCtx, images[0]))

	got, exists, err = suite.store.Get(allAccessCtx, images[0].GetId())
	suite.NoError(err)
	suite.True(exists)
	images[0].GetScan().GetComponents()[0].GetVulns()[0].FirstSystemOccurrence = images[0].GetScan().GetComponents()[1].GetVulns()[0].FirstSystemOccurrence
	images[0].GetScan().GetComponents()[0].GetVulns()[0].FirstImageOccurrence = images[0].GetScan().GetComponents()[1].GetVulns()[0].FirstImageOccurrence
	images[0].GetScan().GetComponents()[0].GetVulns()[0].VulnerabilityType = storage.EmbeddedVulnerability_UNKNOWN_VULNERABILITY
	images[0].GetScan().GetComponents()[0].GetVulns()[0].VulnerabilityTypes = []storage.EmbeddedVulnerability_VulnerabilityType{storage.EmbeddedVulnerability_IMAGE_VULNERABILITY}
	suite.Equal(images[0], got)

	// Test Delete
	for _, d := range images {
		err := suite.store.Delete(allAccessCtx, d.GetId())
		suite.NoError(err)
	}

	// Test Count
	count, err = suite.store.Count(allAccessCtx)
	suite.NoError(err)
	suite.Equal(0, count)

	// Check that the CVEs are removed.
	count, err = suite.cveStorage.Count(allAccessCtx)
	suite.NoError(err)
	suite.Equal(0, count)

	// Check that the Image CVE Edges are removed.
	count, err = suite.imageCVEEdgeStore.Count(allAccessCtx)
	suite.NoError(err)
	suite.Equal(0, count)
}
