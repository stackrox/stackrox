package dackbox

import (
	"testing"

	"github.com/dgraph-io/badger"
	cveStore "github.com/stackrox/rox/central/cve/store"
	cveDackBoxStore "github.com/stackrox/rox/central/cve/store/dackbox"
	"github.com/stackrox/rox/central/image/datastore/internal/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/badgerhelper"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/dackbox"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/suite"
)

func TestImageStore(t *testing.T) {
	suite.Run(t, new(ImageStoreTestSuite))
}

type ImageStoreTestSuite struct {
	suite.Suite

	db    *badger.DB
	dir   string
	dacky *dackbox.DackBox

	store      store.Store
	cveStorage cveStore.Store
}

func (suite *ImageStoreTestSuite) SetupSuite() {
	var err error
	suite.db, suite.dir, err = badgerhelper.NewTemp("reference")
	if err != nil {
		suite.FailNowf("failed to create DB: %+v", err.Error())
	}

	suite.dacky, err = dackbox.NewDackBox(suite.db, nil, []byte("graph"), []byte("dirty"), []byte("valid"))
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
}

func (suite *ImageStoreTestSuite) TearDownSuite() {
	testutils.TearDownBadger(suite.db, suite.dir)
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

	for _, d := range images {
		err := suite.store.Delete(d.GetId())
		suite.NoError(err)
	}

	// Test Count
	count, err = suite.store.CountImages()
	suite.NoError(err)
	suite.Equal(0, count)
}
