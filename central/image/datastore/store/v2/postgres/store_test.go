//go:build sql_integration

package postgres

import (
	"context"
	"testing"

	cveStore "github.com/stackrox/rox/central/cve/image/v2/datastore/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/suite"
)

type ImagesStoreSuite struct {
	suite.Suite
}

func TestImagesStore(t *testing.T) {
	suite.Run(t, new(ImagesStoreSuite))
}

func (s *ImagesStoreSuite) TestStore() {
	if !features.FlattenCVEData.Enabled() {
		s.T().Skip("FlattenCVEData is not enabled")
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
}

func (s *ImagesStoreSuite) TestNVDCVSS() {
	if !features.FlattenCVEData.Enabled() {
		s.T().Skip("FlattenCVEData is not enabled")
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
}
