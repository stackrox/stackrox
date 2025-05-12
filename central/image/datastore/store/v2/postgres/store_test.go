//go:build sql_integration

package postgres

import (
	"context"
	"fmt"
	"testing"
	"time"

	cveStore "github.com/stackrox/rox/central/cve/image/v2/datastore/store/postgres"
	"github.com/stackrox/rox/central/image/datastore/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	pkgSchema "github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/suite"
)

var (
	lastWeek  = time.Now().Add(-7 * 24 * time.Hour)
	yesterday = time.Now().Add(-24 * time.Hour)
)

type ImagesStoreSuite struct {
	suite.Suite

	ctx        context.Context
	testDB     *pgtest.TestPostgres
	store      store.Store
	cvePgStore cveStore.Store
}

func TestImagesStore(t *testing.T) {
	suite.Run(t, new(ImagesStoreSuite))
}

func (s *ImagesStoreSuite) SetupSuite() {
	if !features.FlattenCVEData.Enabled() {
		s.T().Setenv("ROX_FLATTEN_CVE_DATA", "true")
	}

	s.ctx = sac.WithAllAccess(context.Background())
	s.testDB = pgtest.ForT(s.T())

	s.store = New(s.testDB.DB, false, concurrency.NewKeyFence())
	s.cvePgStore = cveStore.New(s.testDB.DB)
}

func (s *ImagesStoreSuite) SetupTest() {
	_, err := s.testDB.DB.Exec(s.ctx, "TRUNCATE "+pkgSchema.ImageCvesV2TableName+" CASCADE")
	s.Require().NoError(err)
	_, err = s.testDB.DB.Exec(s.ctx, "TRUNCATE "+pkgSchema.ImageComponentV2TableName+" CASCADE")
	s.Require().NoError(err)
	_, err = s.testDB.DB.Exec(s.ctx, "TRUNCATE "+pkgSchema.ImagesTableName+" CASCADE")
	s.Require().NoError(err)
}

func (s *ImagesStoreSuite) TearDownSuite() {

	s.T().Setenv("ROX_FLATTEN_CVE_DATA", "false")
}

func (s *ImagesStoreSuite) TestCountCVEs() {
	image := fixtures.GetImagewithDulicateVulnerabilities()
	s.NoError(s.store.Upsert(s.ctx, image))
	_, exists, err := s.store.Get(s.ctx, image.GetId())
	s.NoError(err)
	s.True(exists)
	cveCount, err := s.cvePgStore.Count(s.ctx, search.EmptyQuery())
	s.NoError(err, "Query to get CVE Count failed")
	s.Equal(cveCount, 252)
}
func (s *ImagesStoreSuite) TestStore() {
	image := fixtures.GetImage()
	s.NoError(testutils.FullInit(image, testutils.SimpleInitializer(), testutils.JSONFieldsFilter))
	for _, comp := range image.GetScan().GetComponents() {
		for _, vuln := range comp.GetVulns() {
			vuln.NvdCvss = 0
			vuln.Suppressed = false
			vuln.SuppressActivation = nil
			vuln.SuppressExpiry = nil
			vuln.Advisory = nil
		}
		comp.License = nil
	}

	foundImage, exists, err := s.store.Get(s.ctx, image.GetId())
	s.NoError(err)
	s.False(exists)
	s.Nil(foundImage)

	s.NoError(s.store.Upsert(s.ctx, image))
	foundImage, exists, err = s.store.Get(s.ctx, image.GetId())
	s.NoError(err)
	s.True(exists)
	cloned := image.CloneVT()

	protoassert.Equal(s.T(), cloned, foundImage)

	imageCount, err := s.store.Count(s.ctx, search.EmptyQuery())
	s.NoError(err)
	s.Equal(imageCount, 1)

	imageExists, err := s.store.Exists(s.ctx, image.GetId())
	s.NoError(err)
	s.True(imageExists)
	s.NoError(s.store.Upsert(s.ctx, image))

	foundImage, exists, err = s.store.Get(s.ctx, image.GetId())
	s.NoError(err)
	s.True(exists)

	// Reconcile the timestamps that are set during upsert.
	cloned.LastUpdated = foundImage.LastUpdated
	protoassert.Equal(s.T(), cloned, foundImage)

	s.NoError(s.store.Delete(s.ctx, image.GetId()))
	foundImage, exists, err = s.store.Get(s.ctx, image.GetId())
	s.NoError(err)
	s.False(exists)
	s.Nil(foundImage)
}

func (s *ImagesStoreSuite) TestNVDCVSS() {
	image := fixtures.GetImage()
	s.NoError(testutils.FullInit(image, testutils.SimpleInitializer(), testutils.JSONFieldsFilter))
	nvdCvss := &storage.CVSSScore{
		Source: storage.Source_SOURCE_NVD,
		CvssScore: &storage.CVSSScore_Cvssv3{
			Cvssv3: &storage.CVSSV3{
				Score: 10,
			},
		},
	}
	for _, component := range image.GetScan().GetComponents() {
		for _, vuln := range component.GetVulns() {
			vuln.CvssMetrics = []*storage.CVSSScore{nvdCvss}
		}

	}

	s.NoError(s.store.Upsert(s.ctx, image))
	foundImage, exists, err := s.store.Get(s.ctx, image.GetId())
	s.NoError(err)
	s.True(exists)
	s.NotEmpty(foundImage)

	cves, err := s.cvePgStore.GetIDs(s.ctx)
	s.Require().NoError(err)
	s.Require().NotEmpty(cves)
	id := cves[0]
	imageCve, _, err := s.cvePgStore.Get(s.ctx, id)
	s.Require().NoError(err)
	s.Require().NotEmpty(imageCve)
	s.Equal(float32(10), imageCve.GetNvdcvss())
	s.Require().NotEmpty(imageCve.GetCveBaseInfo().GetCvssMetrics())
	protoassert.Equal(s.T(), nvdCvss, imageCve.GetCveBaseInfo().GetCvssMetrics()[0])
}

func (s *ImagesStoreSuite) TestUpsert() {
	image := getTestImage("image1")

	s.NoError(s.store.Upsert(s.ctx, image))
	foundImage, exists, err := s.store.Get(s.ctx, image.GetId())
	s.NoError(err)
	s.True(exists)
	cloned := image.CloneVT()

	// Reconcile the timestamps that are set during upsert.
	cloned.LastUpdated = foundImage.LastUpdated
	// Because of times we need to reconcile the components to account
	// for first image occurrence and first system time of a CVE
	cloned.Scan.Components = getTestImageComponentsVerify()

	protoassert.Equal(s.T(), cloned, foundImage)

	// Add a new component with "cve1" that has new times
	// Ensure old times are associated with the CVE in the new component.
	image.Scan.Components = append(image.Scan.Components, getComponent3())
	s.NoError(s.store.Upsert(s.ctx, image))
	foundImage, exists, err = s.store.Get(s.ctx, image.GetId())
	s.NoError(err)
	s.True(exists)

	// Should pull the old CVE times for CVE1 even though it just appeared in
	// the component.  The CVE has still existed in the image even though it is
	// new to the component.
	cloned.LastUpdated = foundImage.LastUpdated
	cloned.Scan.Hashoneof = &storage.ImageScan_Hash{
		Hash: foundImage.GetScan().GetHash(),
	}
	cloned.Scan.Components = append(cloned.Scan.Components, getComponent3Verify())
	protoassert.Equal(s.T(), cloned, foundImage)

	// Replace all components removing "cve1".
	// Ensure "cve1" is not returned with the image.
	image.Scan.Components = getTestImageComponentsFixedCVE1()
	s.NoError(s.store.Upsert(s.ctx, image))
	foundImage, exists, err = s.store.Get(s.ctx, image.GetId())
	s.NoError(err)
	s.True(exists)
	cloned = image.CloneVT()

	// Should pull the old CVE times for CVE1 even though it just appeared in
	// the component.  The CVE has still existed in the image even though it is
	// new to the component.
	cloned.LastUpdated = foundImage.LastUpdated
	cloned.Scan.Hashoneof = &storage.ImageScan_Hash{
		Hash: foundImage.GetScan().GetHash(),
	}
	protoassert.Equal(s.T(), cloned, foundImage)

	s.NoError(s.store.Delete(s.ctx, image.GetId()))
	foundImage, exists, err = s.store.Get(s.ctx, image.GetId())
	s.NoError(err)
	s.False(exists)
	s.Nil(foundImage)
}

func (s *ImagesStoreSuite) TestUpdateVulnState() {
	image := getTestImage("image1")
	image2 := getTestImage("image2")

	// Add an image with CVE1
	s.NoError(s.store.Upsert(s.ctx, image))
	_, exists, err := s.store.Get(s.ctx, image.GetId())
	s.NoError(err)
	s.True(exists)

	// Add a second image with CVE1
	s.NoError(s.store.Upsert(s.ctx, image2))
	_, exists, err = s.store.Get(s.ctx, image2.GetId())
	s.NoError(err)
	s.True(exists)

	s.NoError(s.store.UpdateVulnState(s.ctx, "cve1", []string{image.GetId()}, storage.VulnerabilityState_FALSE_POSITIVE))

	walkFn := func(obj *storage.ImageCVEV2) error {
		switch obj.GetImageId() {
		case image.GetId():
			if obj.GetCveBaseInfo().GetCve() == "cve1" && obj.GetState() != storage.VulnerabilityState_FALSE_POSITIVE {
				return fmt.Errorf("expected CVE1 of image1 to be false positive but got %s", obj.GetState())
			}
		case image2.GetId():
			if obj.GetState() != storage.VulnerabilityState_OBSERVED {
				return fmt.Errorf("expected CVE1 of image2 to be observed but got %s", obj.GetState())
			}
		}
		return nil
	}

	s.NoError(s.cvePgStore.Walk(s.ctx, walkFn))
}

func (s *ImagesStoreSuite) TestGetManyImageMetadata() {
	image := getTestImage("image1")
	image2 := getTestImage("image2")

	// Add an image with CVE1
	s.NoError(s.store.Upsert(s.ctx, image))
	_, exists, err := s.store.Get(s.ctx, image.GetId())
	s.NoError(err)
	s.True(exists)

	// Add a second image with CVE1
	s.NoError(s.store.Upsert(s.ctx, image2))
	_, exists, err = s.store.Get(s.ctx, image2.GetId())
	s.NoError(err)
	s.True(exists)

	searchedIndexes := []string{image.GetId(), image2.GetId()}
	returnedImages, err := s.store.GetManyImageMetadata(s.ctx, searchedIndexes)
	s.NoError(err)
	s.Equal(2, len(returnedImages))

	for _, image := range returnedImages {
		s.Nil(image.GetScan().GetComponents())
		s.Contains(searchedIndexes, image.GetId())
	}

	searchedIndexes = []string{image.GetId(), image2.GetId(), "nonsense"}
	returnedImages, err = s.store.GetManyImageMetadata(s.ctx, searchedIndexes)
	s.NoError(err)
	s.Equal(2, len(returnedImages))
}

func (s *ImagesStoreSuite) TestWalkByQuery() {
	image := getTestImage("image1")
	image2 := getTestImage("image2")

	// Add an image with CVE1
	s.NoError(s.store.Upsert(s.ctx, image))
	_, exists, err := s.store.Get(s.ctx, image.GetId())
	s.NoError(err)
	s.True(exists)

	// Add a second image with CVE1
	s.NoError(s.store.Upsert(s.ctx, image2))
	_, exists, err = s.store.Get(s.ctx, image2.GetId())
	s.NoError(err)
	s.True(exists)

	walkFn := func(obj *storage.Image) error {
		if obj.GetId() != image.GetId() {
			return fmt.Errorf("expected image1 but got %s", obj.GetId())
		}
		return nil
	}

	q := search.NewQueryBuilder().AddExactMatches(search.ImageSHA, image.GetId()).ProtoQuery()
	s.NoError(s.store.WalkByQuery(s.ctx, q, walkFn))
}

func (s *ImagesStoreSuite) TestGetMany() {
	image := getTestImage("image1")
	image2 := getTestImage("image2")

	// Add an image with CVE1
	s.NoError(s.store.Upsert(s.ctx, image))
	_, exists, err := s.store.Get(s.ctx, image.GetId())
	s.NoError(err)
	s.True(exists)

	// Add a second image with CVE1
	s.NoError(s.store.Upsert(s.ctx, image2))
	_, exists, err = s.store.Get(s.ctx, image2.GetId())
	s.NoError(err)
	s.True(exists)

	searchedIndexes := []string{image.GetId(), image2.GetId()}
	returnedImages, err := s.store.GetByIDs(s.ctx, searchedIndexes)
	s.NoError(err)
	s.Equal(2, len(returnedImages))

	for _, image := range returnedImages {
		s.NotNil(image.GetScan().GetComponents())
		s.Contains(searchedIndexes, image.GetId())
	}

	searchedIndexes = []string{image.GetId(), image2.GetId(), "nonsense"}
	returnedImages, err = s.store.GetByIDs(s.ctx, searchedIndexes)
	s.NoError(err)
	s.Equal(2, len(returnedImages))
}

func getTestImage(id string) *storage.Image {
	return &storage.Image{
		Id: id,
		Scan: &storage.ImageScan{
			OperatingSystem: "blah",
			ScanTime:        protocompat.TimestampNow(),
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
							Cve:                "cve1",
							VulnerabilityType:  storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
							VulnerabilityTypes: []storage.EmbeddedVulnerability_VulnerabilityType{storage.EmbeddedVulnerability_IMAGE_VULNERABILITY},
							CvssV3: &storage.CVSSV3{
								ImpactScore: 10,
							},
							ScoreVersion:          storage.EmbeddedVulnerability_V3,
							PublishedOn:           protocompat.ConvertTimeToTimestampOrNil(&lastWeek),
							FirstImageOccurrence:  protocompat.ConvertTimeToTimestampOrNil(&lastWeek),
							FirstSystemOccurrence: protocompat.ConvertTimeToTimestampOrNil(&lastWeek),
						},
						{
							Cve:                "cve2",
							VulnerabilityType:  storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
							VulnerabilityTypes: []storage.EmbeddedVulnerability_VulnerabilityType{storage.EmbeddedVulnerability_IMAGE_VULNERABILITY},
							SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
								FixedBy: "ver3",
							},
							CvssV3: &storage.CVSSV3{
								ImpactScore: 1,
							},
							ScoreVersion:          storage.EmbeddedVulnerability_V3,
							PublishedOn:           protocompat.ConvertTimeToTimestampOrNil(&yesterday),
							FirstImageOccurrence:  protocompat.ConvertTimeToTimestampOrNil(&yesterday),
							FirstSystemOccurrence: protocompat.ConvertTimeToTimestampOrNil(&yesterday),
						},
					},
				},
				{
					Name:    "comp2",
					Version: "ver1",
					Vulns: []*storage.EmbeddedVulnerability{
						{
							Cve:                "cve1",
							VulnerabilityType:  storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
							VulnerabilityTypes: []storage.EmbeddedVulnerability_VulnerabilityType{storage.EmbeddedVulnerability_IMAGE_VULNERABILITY},
							SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
								FixedBy: "ver2",
							},
							CvssV3: &storage.CVSSV3{
								ImpactScore: 10,
							},
							ScoreVersion:          storage.EmbeddedVulnerability_V3,
							PublishedOn:           protocompat.ConvertTimeToTimestampOrNil(&lastWeek),
							FirstImageOccurrence:  protocompat.ConvertTimeToTimestampOrNil(&lastWeek),
							FirstSystemOccurrence: protocompat.ConvertTimeToTimestampOrNil(&lastWeek),
						},
						{
							Cve:                "cve2",
							VulnerabilityType:  storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
							VulnerabilityTypes: []storage.EmbeddedVulnerability_VulnerabilityType{storage.EmbeddedVulnerability_IMAGE_VULNERABILITY},
							CvssV3: &storage.CVSSV3{
								ImpactScore: 1,
							},
							ScoreVersion:          storage.EmbeddedVulnerability_V3,
							PublishedOn:           protocompat.ConvertTimeToTimestampOrNil(&lastWeek),
							FirstImageOccurrence:  protocompat.ConvertTimeToTimestampOrNil(&lastWeek),
							FirstSystemOccurrence: protocompat.ConvertTimeToTimestampOrNil(&lastWeek),
						},
					},
				},
			},
		},
		RiskScore: 30,
		Priority:  1,
	}
}

func getTestImageComponentsVerify() []*storage.EmbeddedImageScanComponent {
	return []*storage.EmbeddedImageScanComponent{
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
					Cve:                "cve1",
					VulnerabilityType:  storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
					VulnerabilityTypes: []storage.EmbeddedVulnerability_VulnerabilityType{storage.EmbeddedVulnerability_IMAGE_VULNERABILITY},
					CvssV3: &storage.CVSSV3{
						ImpactScore: 10,
					},
					ScoreVersion:          storage.EmbeddedVulnerability_V3,
					PublishedOn:           protocompat.ConvertTimeToTimestampOrNil(&lastWeek),
					FirstImageOccurrence:  protocompat.ConvertTimeToTimestampOrNil(&lastWeek),
					FirstSystemOccurrence: protocompat.ConvertTimeToTimestampOrNil(&lastWeek),
				},
				{
					Cve:                "cve2",
					VulnerabilityType:  storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
					VulnerabilityTypes: []storage.EmbeddedVulnerability_VulnerabilityType{storage.EmbeddedVulnerability_IMAGE_VULNERABILITY},
					SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
						FixedBy: "ver3",
					},
					CvssV3: &storage.CVSSV3{
						ImpactScore: 1,
					},
					ScoreVersion:          storage.EmbeddedVulnerability_V3,
					PublishedOn:           protocompat.ConvertTimeToTimestampOrNil(&yesterday),
					FirstImageOccurrence:  protocompat.ConvertTimeToTimestampOrNil(&lastWeek),
					FirstSystemOccurrence: protocompat.ConvertTimeToTimestampOrNil(&lastWeek),
				},
			},
		},
		{
			Name:    "comp2",
			Version: "ver1",
			Vulns: []*storage.EmbeddedVulnerability{
				{
					Cve:                "cve1",
					VulnerabilityType:  storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
					VulnerabilityTypes: []storage.EmbeddedVulnerability_VulnerabilityType{storage.EmbeddedVulnerability_IMAGE_VULNERABILITY},
					SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
						FixedBy: "ver2",
					},
					CvssV3: &storage.CVSSV3{
						ImpactScore: 10,
					},
					ScoreVersion:          storage.EmbeddedVulnerability_V3,
					PublishedOn:           protocompat.ConvertTimeToTimestampOrNil(&lastWeek),
					FirstImageOccurrence:  protocompat.ConvertTimeToTimestampOrNil(&lastWeek),
					FirstSystemOccurrence: protocompat.ConvertTimeToTimestampOrNil(&lastWeek),
				},
				{
					Cve:                "cve2",
					VulnerabilityType:  storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
					VulnerabilityTypes: []storage.EmbeddedVulnerability_VulnerabilityType{storage.EmbeddedVulnerability_IMAGE_VULNERABILITY},
					CvssV3: &storage.CVSSV3{
						ImpactScore: 1,
					},
					ScoreVersion:          storage.EmbeddedVulnerability_V3,
					PublishedOn:           protocompat.ConvertTimeToTimestampOrNil(&lastWeek),
					FirstImageOccurrence:  protocompat.ConvertTimeToTimestampOrNil(&lastWeek),
					FirstSystemOccurrence: protocompat.ConvertTimeToTimestampOrNil(&lastWeek),
				},
			},
		},
	}
}

func getComponent3() *storage.EmbeddedImageScanComponent {
	return &storage.EmbeddedImageScanComponent{
		Name:    "comp3",
		Version: "ver1",
		Vulns: []*storage.EmbeddedVulnerability{
			{
				Cve:                "cve1",
				VulnerabilityType:  storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
				VulnerabilityTypes: []storage.EmbeddedVulnerability_VulnerabilityType{storage.EmbeddedVulnerability_IMAGE_VULNERABILITY},
				SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
					FixedBy: "ver2",
				},
				CvssV3: &storage.CVSSV3{
					ImpactScore: 10,
				},
				ScoreVersion:          storage.EmbeddedVulnerability_V3,
				PublishedOn:           protocompat.ConvertTimeToTimestampOrNil(&yesterday),
				FirstImageOccurrence:  protocompat.ConvertTimeToTimestampOrNil(&yesterday),
				FirstSystemOccurrence: protocompat.ConvertTimeToTimestampOrNil(&yesterday),
			},
			{
				Cve:                "cve3",
				VulnerabilityType:  storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
				VulnerabilityTypes: []storage.EmbeddedVulnerability_VulnerabilityType{storage.EmbeddedVulnerability_IMAGE_VULNERABILITY},
				CvssV3: &storage.CVSSV3{
					ImpactScore: 1,
				},
				ScoreVersion:          storage.EmbeddedVulnerability_V3,
				PublishedOn:           protocompat.ConvertTimeToTimestampOrNil(&yesterday),
				FirstImageOccurrence:  protocompat.ConvertTimeToTimestampOrNil(&yesterday),
				FirstSystemOccurrence: protocompat.ConvertTimeToTimestampOrNil(&yesterday),
			},
		},
	}
}

func getComponent3Verify() *storage.EmbeddedImageScanComponent {
	return &storage.EmbeddedImageScanComponent{
		Name:    "comp3",
		Version: "ver1",
		Vulns: []*storage.EmbeddedVulnerability{
			{
				Cve:                "cve1",
				VulnerabilityType:  storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
				VulnerabilityTypes: []storage.EmbeddedVulnerability_VulnerabilityType{storage.EmbeddedVulnerability_IMAGE_VULNERABILITY},
				SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
					FixedBy: "ver2",
				},
				CvssV3: &storage.CVSSV3{
					ImpactScore: 10,
				},
				ScoreVersion:          storage.EmbeddedVulnerability_V3,
				PublishedOn:           protocompat.ConvertTimeToTimestampOrNil(&yesterday),
				FirstImageOccurrence:  protocompat.ConvertTimeToTimestampOrNil(&lastWeek),
				FirstSystemOccurrence: protocompat.ConvertTimeToTimestampOrNil(&lastWeek),
			},
			{
				Cve:                "cve3",
				VulnerabilityType:  storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
				VulnerabilityTypes: []storage.EmbeddedVulnerability_VulnerabilityType{storage.EmbeddedVulnerability_IMAGE_VULNERABILITY},
				CvssV3: &storage.CVSSV3{
					ImpactScore: 1,
				},
				ScoreVersion:          storage.EmbeddedVulnerability_V3,
				PublishedOn:           protocompat.ConvertTimeToTimestampOrNil(&yesterday),
				FirstImageOccurrence:  protocompat.ConvertTimeToTimestampOrNil(&yesterday),
				FirstSystemOccurrence: protocompat.ConvertTimeToTimestampOrNil(&yesterday),
			},
		},
	}
}

func getTestImageComponentsFixedCVE1() []*storage.EmbeddedImageScanComponent {
	return []*storage.EmbeddedImageScanComponent{
		{
			Name:    "comp1",
			Version: "ver1",
			Vulns:   []*storage.EmbeddedVulnerability{},
		},
		{
			Name:    "comp1",
			Version: "ver3",
			Vulns: []*storage.EmbeddedVulnerability{
				{
					Cve:                "cve2",
					VulnerabilityType:  storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
					VulnerabilityTypes: []storage.EmbeddedVulnerability_VulnerabilityType{storage.EmbeddedVulnerability_IMAGE_VULNERABILITY},
					SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
						FixedBy: "ver3",
					},
					CvssV3: &storage.CVSSV3{
						ImpactScore: 1,
					},
					ScoreVersion:          storage.EmbeddedVulnerability_V3,
					PublishedOn:           protocompat.ConvertTimeToTimestampOrNil(&yesterday),
					FirstImageOccurrence:  protocompat.ConvertTimeToTimestampOrNil(&lastWeek),
					FirstSystemOccurrence: protocompat.ConvertTimeToTimestampOrNil(&lastWeek),
				},
			},
		},
		{
			Name:    "comp2",
			Version: "ver2",
			Vulns: []*storage.EmbeddedVulnerability{
				{
					Cve:                "cve2",
					VulnerabilityType:  storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
					VulnerabilityTypes: []storage.EmbeddedVulnerability_VulnerabilityType{storage.EmbeddedVulnerability_IMAGE_VULNERABILITY},
					CvssV3: &storage.CVSSV3{
						ImpactScore: 1,
					},
					ScoreVersion:          storage.EmbeddedVulnerability_V3,
					PublishedOn:           protocompat.ConvertTimeToTimestampOrNil(&lastWeek),
					FirstImageOccurrence:  protocompat.ConvertTimeToTimestampOrNil(&lastWeek),
					FirstSystemOccurrence: protocompat.ConvertTimeToTimestampOrNil(&lastWeek),
				},
			},
		},
		{
			Name:    "comp3",
			Version: "ver2",
			Vulns: []*storage.EmbeddedVulnerability{
				{
					Cve:                "cve3",
					VulnerabilityType:  storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
					VulnerabilityTypes: []storage.EmbeddedVulnerability_VulnerabilityType{storage.EmbeddedVulnerability_IMAGE_VULNERABILITY},
					CvssV3: &storage.CVSSV3{
						ImpactScore: 1,
					},
					ScoreVersion:          storage.EmbeddedVulnerability_V3,
					PublishedOn:           protocompat.ConvertTimeToTimestampOrNil(&yesterday),
					FirstImageOccurrence:  protocompat.ConvertTimeToTimestampOrNil(&yesterday),
					FirstSystemOccurrence: protocompat.ConvertTimeToTimestampOrNil(&yesterday),
				},
			},
		},
	}
}
