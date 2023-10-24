//go:build sql_integration

package suppress

import (
	"context"
	"sort"
	"testing"
	"time"

	cveDS "github.com/stackrox/rox/central/cve/image/datastore"
	cveSearcher "github.com/stackrox/rox/central/cve/image/datastore/search"
	cvePG "github.com/stackrox/rox/central/cve/image/datastore/store/postgres"
	imageDS "github.com/stackrox/rox/central/image/datastore"
	imagePG "github.com/stackrox/rox/central/image/datastore/store/postgres"
	"github.com/stackrox/rox/central/ranking"
	mockRisks "github.com/stackrox/rox/central/risk/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	"gorm.io/gorm"
)

func TestReprocessorWithPostgres(t *testing.T) {
	suite.Run(t, new(ReprocessorPostgresTestSuite))
}

type ReprocessorPostgresTestSuite struct {
	suite.Suite

	ctx             context.Context
	db              postgres.DB
	gormDB          *gorm.DB
	imageDataStore  imageDS.DataStore
	cveDataStore    cveDS.DataStore
	mockRisk        *mockRisks.MockDataStore
	reprocessorLoop *cveUnsuppressLoopImpl
}

func (s *ReprocessorPostgresTestSuite) SetupSuite() {

	s.ctx = context.Background()

	source := pgtest.GetConnectionString(s.T())
	config, err := postgres.ParseConfig(source)
	s.Require().NoError(err)

	pool, err := postgres.New(s.ctx, config)
	s.NoError(err)
	s.gormDB = pgtest.OpenGormDB(s.T(), source)
	s.db = pool
}

func (s *ReprocessorPostgresTestSuite) SetupTest() {
	imagePG.Destroy(s.ctx, s.db)

	s.mockRisk = mockRisks.NewMockDataStore(gomock.NewController(s.T()))
	s.imageDataStore = imageDS.NewWithPostgres(imagePG.CreateTableAndNewStore(s.ctx, s.db, s.gormDB, false), imagePG.NewIndexer(s.db), s.mockRisk, ranking.ImageRanker(), ranking.ComponentRanker())

	cveStore := cvePG.New(s.db)
	cveIndexer := cvePG.NewIndexer(s.db)
	cveDataStore := cveDS.New(cveStore, cveSearcher.New(cveStore, cveIndexer), concurrency.NewKeyFence())
	s.cveDataStore = cveDataStore

	s.reprocessorLoop = NewLoop(cveDataStore).(*cveUnsuppressLoopImpl)
}

func (s *ReprocessorPostgresTestSuite) TearDownSuite() {
	s.db.Close()
	pgtest.CloseGormDB(s.T(), s.gormDB)
}

func (s *ReprocessorPostgresTestSuite) TestUnsuppressWithPostgres() {
	ctx := sac.WithAllAccess(context.Background())
	image := fixtures.GetImageWithUniqueComponents(5)

	image.Priority = 1
	for _, component := range image.GetScan().GetComponents() {
		for _, vuln := range component.GetVulns() {
			vuln.Suppressed = true
			vuln.SuppressExpiry = protoconv.ConvertTimeToTimestamp(time.Now().Add(-2 * 24 * time.Hour))
			vuln.VulnerabilityTypes = []storage.EmbeddedVulnerability_VulnerabilityType{storage.EmbeddedVulnerability_IMAGE_VULNERABILITY}
		}
	}

	components := image.GetScan().GetComponents()
	sort.SliceStable(components, func(i, j int) bool {
		return components[i].GetName() < components[j].GetName()
	})
	for _, comp := range components {
		sort.SliceStable(comp.Vulns, func(i, j int) bool {
			return comp.Vulns[i].GetCve() < comp.Vulns[j].GetCve()
		})
	}

	s.NoError(s.imageDataStore.UpsertImage(ctx, image))

	storedImage, found, err := s.imageDataStore.GetImage(ctx, image.GetId())
	s.NoError(err)
	s.True(found)
	for _, component := range image.GetScan().GetComponents() {
		for _, cve := range component.GetVulns() {
			cve.FirstSystemOccurrence = storedImage.GetLastUpdated()
			cve.FirstImageOccurrence = storedImage.GetLastUpdated()
		}
	}
	expectedImage := cloneAndUpdateRiskPriority(image)
	s.Equal(expectedImage, storedImage)

	s.reprocessorLoop.unsuppressCVEsWithExpiredSuppressState()

	storedImage, found, err = s.imageDataStore.GetImage(ctx, image.GetId())
	s.NoError(err)
	s.True(found)
	for _, component := range storedImage.GetScan().GetComponents() {
		for _, vuln := range component.GetVulns() {
			s.False(vuln.GetSuppressed())
		}
	}
}

func cloneAndUpdateRiskPriority(image *storage.Image) *storage.Image {
	cloned := image.Clone()
	cloned.Priority = 1
	for _, component := range cloned.GetScan().GetComponents() {
		component.Priority = 1
	}
	return cloned
}
