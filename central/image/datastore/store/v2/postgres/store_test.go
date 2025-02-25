//go:build sql_integration

package postgres

import (
	"context"
	"testing"
	"time"

	cveStore "github.com/stackrox/rox/central/cve/image/v2/datastore/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
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
}

func TestImagesStore(t *testing.T) {
	suite.Run(t, new(ImagesStoreSuite))
}

func (s *ImagesStoreSuite) TestStore() {
	if !features.FlattenCVEData.Enabled() {
		s.T().Setenv("ROX_FLATTEN_CVE_DATA", "true")
	}

	ctx := sac.WithAllAccess(context.Background())

	source := pgtest.GetConnectionString(s.T())
	config, err := postgres.ParseConfig(source)
	s.Require().NoError(err)
	pool, err := postgres.New(ctx, config)
	s.NoError(err)
	defer pool.Close()

	Destroy(ctx, pool)

	gormDB := pgtest.OpenGormDB(s.T(), source)
	defer pgtest.CloseGormDB(s.T(), gormDB)
	store := CreateTableAndNewStore(ctx, pool, gormDB, false)

	image := fixtures.GetImage()
	s.NoError(testutils.FullInit(image, testutils.SimpleInitializer(), testutils.JSONFieldsFilter))
	for _, comp := range image.GetScan().GetComponents() {
		for _, vuln := range comp.GetVulns() {
			vuln.NvdCvss = 0
			vuln.Suppressed = false
			vuln.SuppressActivation = nil
			vuln.SuppressExpiry = nil
		}
	}

	foundImage, exists, err := store.Get(ctx, image.GetId())
	s.NoError(err)
	s.False(exists)
	s.Nil(foundImage)

	s.NoError(store.Upsert(ctx, image))
	foundImage, exists, err = store.Get(ctx, image.GetId())
	s.NoError(err)
	s.True(exists)
	cloned := image.CloneVT()

	log.Infof("SHREWS -- cloned %v", cloned)
	log.Infof("SHREWS -- found1 %v", foundImage)
	protoassert.Equal(s.T(), cloned, foundImage)

	imageCount, err := store.Count(ctx, search.EmptyQuery())
	s.NoError(err)
	s.Equal(imageCount, 1)

	imageExists, err := store.Exists(ctx, image.GetId())
	s.NoError(err)
	s.True(imageExists)
	s.NoError(store.Upsert(ctx, image))

	foundImage, exists, err = store.Get(ctx, image.GetId())
	s.NoError(err)
	s.True(exists)

	// Reconcile the timestamps that are set during upsert.
	cloned.LastUpdated = foundImage.LastUpdated
	protoassert.Equal(s.T(), cloned, foundImage)

	s.NoError(store.Delete(ctx, image.GetId()))
	foundImage, exists, err = store.Get(ctx, image.GetId())
	s.NoError(err)
	s.False(exists)
	s.Nil(foundImage)

	s.T().Setenv("ROX_FLATTEN_CVE_DATA", "false")
}

func (s *ImagesStoreSuite) TestNVDCVSS() {
	if !features.FlattenCVEData.Enabled() {
		s.T().Setenv("ROX_FLATTEN_CVE_DATA", "true")
	}

	ctx := sac.WithAllAccess(context.Background())
	source := pgtest.GetConnectionString(s.T())
	config, err := postgres.ParseConfig(source)
	s.Require().NoError(err)
	pool, err := postgres.New(ctx, config)
	s.NoError(err)
	defer pool.Close()
	Destroy(ctx, pool)

	gormDB := pgtest.OpenGormDB(s.T(), source)
	defer pgtest.CloseGormDB(s.T(), gormDB)
	store := CreateTableAndNewStore(ctx, pool, gormDB, false)

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

	s.NoError(store.Upsert(ctx, image))
	foundImage, exists, err := store.Get(ctx, image.GetId())
	s.NoError(err)
	s.True(exists)
	s.NotEmpty(foundImage)

	cvePgStore := cveStore.CreateTableAndNewStore(ctx, pool, gormDB)
	cves, err := cvePgStore.GetIDs(ctx)
	s.Require().NoError(err)
	s.Require().NotEmpty(cves)
	id := cves[0]
	imageCve, _, err := cvePgStore.Get(ctx, id)
	s.Require().NoError(err)
	s.Require().NotEmpty(imageCve)
	s.Equal(float32(10), imageCve.GetNvdcvss())
	s.Require().NotEmpty(imageCve.GetCveBaseInfo().GetCvssMetrics())
	protoassert.Equal(s.T(), nvdCvss, imageCve.GetCveBaseInfo().GetCvssMetrics()[0])

	s.T().Setenv("ROX_FLATTEN_CVE_DATA", "false")
}

func (s *ImagesStoreSuite) TestUpsert() {
	if !features.FlattenCVEData.Enabled() {
		s.T().Setenv("ROX_FLATTEN_CVE_DATA", "true")
	}

	if !features.FlattenCVEData.Enabled() {
		s.T().Setenv("ROX_FLATTEN_CVE_DATA", "true")
	}

	ctx := sac.WithAllAccess(context.Background())

	source := pgtest.GetConnectionString(s.T())
	config, err := postgres.ParseConfig(source)
	s.Require().NoError(err)
	pool, err := postgres.New(ctx, config)
	s.NoError(err)
	defer pool.Close()

	Destroy(ctx, pool)

	gormDB := pgtest.OpenGormDB(s.T(), source)
	defer pgtest.CloseGormDB(s.T(), gormDB)
	store := CreateTableAndNewStore(ctx, pool, gormDB, false)

	image := getTestImage("image1")

	s.NoError(store.Upsert(ctx, image))
	foundImage, exists, err := store.Get(ctx, image.GetId())
	s.NoError(err)
	s.True(exists)
	cloned := image.CloneVT()

	// Reconcile the timestamps that are set during upsert.
	cloned.LastUpdated = foundImage.LastUpdated
	// Because of times we need to reconcile the components to account
	// for first image occurrence and first system time of a CVE
	cloned.Scan.Components = getTestImageComponentsVerify()

	log.Infof("SHREWS -- cloned %v", cloned)
	log.Infof("SHREWS -- found1 %v", foundImage)
	protoassert.Equal(s.T(), cloned, foundImage)

	// Add a new component with "cve1" that has new times
	// Ensure old times are associated with the CVE in the new component.
	image.Scan.Components = append(image.Scan.Components, getComponent3())
	s.NoError(store.Upsert(ctx, image))
	foundImage, exists, err = store.Get(ctx, image.GetId())
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
	log.Infof("SHREWS -- cloned %v", cloned)
	log.Infof("SHREWS -- found1 %v", foundImage)
	protoassert.Equal(s.T(), cloned, foundImage)

	// Replace all components removing "cve1".
	// Ensure "cve1" is not returned with the image.
	image.Scan.Components = getTestImageComponentsFixedCVE1()
	s.NoError(store.Upsert(ctx, image))
	foundImage, exists, err = store.Get(ctx, image.GetId())
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
	log.Infof("SHREWS -- cloned %v", cloned)
	log.Infof("SHREWS -- found1 %v", foundImage)
	protoassert.Equal(s.T(), cloned, foundImage)

	s.NoError(store.Delete(ctx, image.GetId()))
	foundImage, exists, err = store.Get(ctx, image.GetId())
	s.NoError(err)
	s.False(exists)
	s.Nil(foundImage)

	s.T().Setenv("ROX_FLATTEN_CVE_DATA", "false")
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
