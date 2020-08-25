package dackbox

import (
	"testing"

	cveStore "github.com/stackrox/rox/central/cve/store"
	cveDackBoxStore "github.com/stackrox/rox/central/cve/store/dackbox"
	"github.com/stackrox/rox/central/image/datastore/internal/store"
	imageCVEEdgeStore "github.com/stackrox/rox/central/imagecveedge/store"
	imageCVEEdgeDackBox "github.com/stackrox/rox/central/imagecveedge/store/dackbox"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/dackbox"
	"github.com/stackrox/rox/pkg/dackbox/edges"
	"github.com/stackrox/rox/pkg/rocksdb"
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
	suite.store, err = New(suite.dacky, concurrency.NewKeyFence(), false)
	if err != nil {
		suite.FailNowf("failed to create key fence: %+v", err.Error())
	}
	suite.cveStorage, err = cveDackBoxStore.New(suite.dacky, concurrency.NewKeyFence())
	if err != nil {
		suite.FailNowf("failed to create cve store: %+v", err.Error())
	}
	suite.imageCVEEdgeStore, err = imageCVEEdgeDackBox.New(suite.dacky)
	if err != nil {
		suite.FailNowf("failed to create imageCVEEdge store: %+v", err.Error())
	}
}

func (suite *ImageStoreTestSuite) TearDownSuite() {
	rocksdbtest.TearDownRocksDB(suite.db)
}

func (suite *ImageStoreTestSuite) TestImages() {
	images := []*storage.Image{
		{
			Id: "sha256:sha1",
			Name: &storage.ImageName{
				FullName: "name1",
			},
			Scan: &storage.ImageScan{
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
		suite.NoError(suite.store.Upsert(d))
	}

	for _, d := range images {
		got, exists, err := suite.store.GetImage(d.GetId())
		suite.NoError(err)
		suite.True(exists)
		// Upsert sets `createdAt` for every CVE that doesn't already exist in the store, which should be same as (*storage.Image).LastUpdated.
		for _, component := range d.GetScan().GetComponents() {
			for _, vuln := range component.GetVulns() {
				vuln.FirstSystemOccurrence = got.GetLastUpdated()
				vuln.FirstImageOccurrence = got.GetLastUpdated()
			}
		}
		suite.Equal(d, got)

		listGot, exists, err := suite.store.ListImage(d.GetId())
		suite.NoError(err)
		suite.True(exists)
		suite.Equal(d.GetName().GetFullName(), listGot.GetName())
	}

	// Check that the CVEs were written with the correct timestamp.
	vuln, _, err := suite.cveStorage.Get("cve1")
	suite.NoError(err)
	suite.Equal(images[0].GetLastUpdated(), vuln.GetCreatedAt())
	vuln, _, err = suite.cveStorage.Get("cve2")
	suite.NoError(err)
	suite.Equal(images[0].GetLastUpdated(), vuln.GetCreatedAt())

	// Check that the Image CVE Edges were written with the correct timestamp.
	imageCVEEdge, _, err := suite.imageCVEEdgeStore.Get(edges.EdgeID{ParentID: "sha256:sha1", ChildID: "cve1"}.ToString())
	suite.NoError(err)
	suite.Equal(images[0].GetLastUpdated(), imageCVEEdge.GetFirstImageOccurrence())
	imageCVEEdge, _, err = suite.imageCVEEdgeStore.Get(edges.EdgeID{ParentID: "sha256:sha1", ChildID: "cve1"}.ToString())
	suite.NoError(err)
	suite.Equal(images[0].GetLastUpdated(), imageCVEEdge.GetFirstImageOccurrence())

	// Test Update
	for _, d := range images {
		d.Name.FullName += "1"
	}

	for _, d := range images {
		suite.NoError(suite.store.Upsert(d))
	}

	for _, d := range images {
		got, exists, err := suite.store.GetImage(d.GetId())
		suite.NoError(err)
		suite.True(exists)
		suite.Equal(d, got)

		listGot, exists, err := suite.store.ListImage(d.GetId())
		suite.NoError(err)
		suite.True(exists)
		suite.Equal(d.GetName().GetFullName(), listGot.GetName())
	}

	// Test Count
	count, err := suite.store.CountImages()
	suite.NoError(err)
	suite.Equal(len(images), count)

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

	suite.NoError(suite.store.Upsert(images[1]))

	got, exists, err := suite.store.GetImage(images[1].GetId())
	suite.NoError(err)
	suite.True(exists)
	images[1].GetScan().GetComponents()[0].GetVulns()[0].FirstSystemOccurrence = images[0].GetScan().GetComponents()[1].GetVulns()[0].FirstSystemOccurrence
	images[1].GetScan().GetComponents()[0].GetVulns()[0].FirstImageOccurrence = got.GetLastUpdated()
	suite.Equal(images[1], got)

	// Test second occurrence of a CVE in an image
	images[0].GetScan().GetComponents()[0].Vulns = append(images[0].GetScan().GetComponents()[0].Vulns,
		&storage.EmbeddedVulnerability{
			Cve:               "cve1",
			VulnerabilityType: storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
		})

	suite.NoError(suite.store.Upsert(images[0]))

	got, exists, err = suite.store.GetImage(images[0].GetId())
	suite.NoError(err)
	suite.True(exists)
	images[0].GetScan().GetComponents()[0].GetVulns()[0].FirstSystemOccurrence = images[0].GetScan().GetComponents()[1].GetVulns()[0].FirstSystemOccurrence
	images[0].GetScan().GetComponents()[0].GetVulns()[0].FirstImageOccurrence = images[0].GetScan().GetComponents()[1].GetVulns()[0].FirstImageOccurrence
	suite.Equal(images[0], got)

	// Test Delete
	for _, d := range images {
		err := suite.store.Delete(d.GetId())
		suite.NoError(err)
	}

	// Test Count
	count, err = suite.store.CountImages()
	suite.NoError(err)
	suite.Equal(0, count)

	// Check that the CVEs are removed.
	count, err = suite.cveStorage.Count()
	suite.NoError(err)
	suite.Equal(0, count)

	// Check that the Image CVE Edges are removed.
	count, err = suite.imageCVEEdgeStore.Count()
	suite.NoError(err)
	suite.Equal(0, count)
}
