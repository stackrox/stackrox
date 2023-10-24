//go:build sql_integration

package datastore

import (
	"context"
	"sort"
	"testing"

	protoTypes "github.com/gogo/protobuf/types"
	imageCVEDS "github.com/stackrox/rox/central/cve/image/datastore"
	imageCVESearch "github.com/stackrox/rox/central/cve/image/datastore/search"
	imageCVEPostgres "github.com/stackrox/rox/central/cve/image/datastore/store/postgres"
	pgStore "github.com/stackrox/rox/central/image/datastore/store/postgres"
	imageComponentDS "github.com/stackrox/rox/central/imagecomponent/datastore"
	imageComponentPostgres "github.com/stackrox/rox/central/imagecomponent/datastore/store/postgres"
	imageComponentSearch "github.com/stackrox/rox/central/imagecomponent/search"
	"github.com/stackrox/rox/central/ranking"
	mockRisks "github.com/stackrox/rox/central/risk/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	pkgCVE "github.com/stackrox/rox/pkg/cve"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/scancomponent"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/scoped"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	"gorm.io/gorm"
)

func TestImageDataStoreWithPostgres(t *testing.T) {
	suite.Run(t, new(ImagePostgresDataStoreTestSuite))
}

type ImagePostgresDataStoreTestSuite struct {
	suite.Suite

	ctx                context.Context
	db                 postgres.DB
	gormDB             *gorm.DB
	datastore          DataStore
	mockRisk           *mockRisks.MockDataStore
	componentDataStore imageComponentDS.DataStore
	cveDataStore       imageCVEDS.DataStore
}

func (s *ImagePostgresDataStoreTestSuite) SetupSuite() {
	s.ctx = context.Background()

	source := pgtest.GetConnectionString(s.T())
	config, err := postgres.ParseConfig(source)
	s.Require().NoError(err)

	pool, err := postgres.New(s.ctx, config)
	s.NoError(err)
	s.gormDB = pgtest.OpenGormDB(s.T(), source)
	s.db = pool
}

func (s *ImagePostgresDataStoreTestSuite) SetupTest() {
	pgStore.Destroy(s.ctx, s.db)

	s.mockRisk = mockRisks.NewMockDataStore(gomock.NewController(s.T()))
	s.datastore = NewWithPostgres(pgStore.CreateTableAndNewStore(s.ctx, s.db, s.gormDB, false), pgStore.NewIndexer(s.db), s.mockRisk, ranking.NewRanker(), ranking.NewRanker())

	componentStorage := imageComponentPostgres.CreateTableAndNewStore(s.ctx, s.db, s.gormDB)
	componentIndexer := imageComponentPostgres.NewIndexer(s.db)
	componentSearcher := imageComponentSearch.NewV2(componentStorage, componentIndexer)
	s.componentDataStore = imageComponentDS.New(componentStorage, componentSearcher, s.mockRisk, ranking.NewRanker())

	cveStorage := imageCVEPostgres.CreateTableAndNewStore(s.ctx, s.db, s.gormDB)
	cveIndexer := imageCVEPostgres.NewIndexer(s.db)
	cveSearcher := imageCVESearch.New(cveStorage, cveIndexer)
	cveDataStore := imageCVEDS.New(cveStorage, cveSearcher, concurrency.NewKeyFence())
	s.cveDataStore = cveDataStore
}

func (s *ImagePostgresDataStoreTestSuite) TearDownSuite() {
	s.db.Close()
	pgtest.CloseGormDB(s.T(), s.gormDB)
}

func (s *ImagePostgresDataStoreTestSuite) TestSearchWithPostgres() {
	image := getTestImage("id1")

	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowFixedScopes(
		sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
		sac.ResourceScopeKeys(resources.Image),
	))

	// Upsert image.
	s.NoError(s.datastore.UpsertImage(ctx, image))

	// Basic unscoped search.
	results, err := s.datastore.Search(ctx, pkgSearch.EmptyQuery())
	s.NoError(err)
	s.Len(results, 1)

	q := pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.ImageSHA, image.GetId()).ProtoQuery()
	results, err = s.datastore.Search(ctx, q)
	s.NoError(err)
	s.Len(results, 1)

	// Sort by impact score
	q = pkgSearch.EmptyQuery()
	q.Pagination = &v1.QueryPagination{
		SortOptions: []*v1.QuerySortOption{
			{
				Field: pkgSearch.ImpactScore.String(),
			},
		},
	}
	results, err = s.cveDataStore.Search(ctx, q)
	s.NoError(err)
	s.Equal([]string{"cve2#blah", "cve1#blah"}, pkgSearch.ResultsToIDs(results))

	// Sort by impact score: reversed
	q = pkgSearch.EmptyQuery()
	q.Pagination = &v1.QueryPagination{
		SortOptions: []*v1.QuerySortOption{
			{
				Field:    pkgSearch.ImpactScore.String(),
				Reversed: true,
			},
		},
	}
	results, err = s.cveDataStore.Search(ctx, q)
	s.NoError(err)
	s.Equal([]string{"cve1#blah", "cve2#blah"}, pkgSearch.ResultsToIDs(results))

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
	s.NoError(s.datastore.UpsertImage(ctx, newImage))

	// Search multiple images.
	images, err := s.datastore.SearchRawImages(ctx, pkgSearch.EmptyQuery())
	s.NoError(err)
	s.Len(images, 2)

	// Search for just one image.
	q = pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.ImageSHA, image.GetId()).ProtoQuery()
	images, err = s.datastore.SearchRawImages(ctx, q)
	s.NoError(err)
	s.Len(images, 1)

	// Search by CVE.
	q = pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.CVE, "cve1").ProtoQuery()
	images, err = s.datastore.SearchRawImages(ctx, q)
	s.NoError(err)
	s.Len(images, 2)

	q = pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.CVE, "cve3").ProtoQuery()
	results, err = s.datastore.Search(ctx, q)
	s.NoError(err)
	s.Len(results, 1)
	s.Equal("id2", results[0].ID)

	// Scope search by image.
	scopedCtx := scoped.Context(ctx, scoped.Scope{
		ID:    image.GetId(),
		Level: v1.SearchCategory_IMAGES,
	})
	results, err = s.datastore.Search(scopedCtx, pkgSearch.EmptyQuery())
	s.NoError(err)
	s.Len(results, 1)
	s.Equal(image.GetId(), results[0].ID)

	// Scope search by vulns.
	scopedCtx = scoped.Context(ctx, scoped.Scope{
		ID:    "cve3#blah",
		Level: v1.SearchCategory_IMAGE_VULNERABILITIES,
	})
	results, err = s.datastore.Search(scopedCtx, pkgSearch.EmptyQuery())
	s.NoError(err)
	s.Len(results, 1)
	s.Equal(newImage.GetId(), results[0].ID)
}

func (s *ImagePostgresDataStoreTestSuite) TestFixableWithPostgres() {
	image := fixtures.GetImageWithUniqueComponents(5)
	ctx := sac.WithAllAccess(context.Background())

	s.NoError(s.datastore.UpsertImage(ctx, image))
	_, found, err := s.datastore.GetImage(ctx, image.GetId())
	s.NoError(err)
	s.True(found)

	results, err := s.datastore.Search(ctx, pkgSearch.NewQueryBuilder().AddBools(pkgSearch.Fixable, true).ProtoQuery())
	s.NoError(err)
	s.Len(results, 1)
	s.Equal(image.GetId(), results[0].ID)

	image.Scan.ScanTime = protoTypes.TimestampNow()
	for _, component := range image.GetScan().GetComponents() {
		for _, vuln := range component.GetVulns() {
			vuln.SetFixedBy = nil
		}
	}
	s.NoError(s.datastore.UpsertImage(ctx, image))
	image, found, err = s.datastore.GetImage(ctx, image.GetId())
	s.NoError(err)
	s.True(found)
	s.Equal(image.GetId(), results[0].ID)

	results, err = s.datastore.Search(ctx, pkgSearch.NewQueryBuilder().AddBools(pkgSearch.Fixable, true).ProtoQuery())
	s.NoError(err)
	s.Len(results, 0)
}

func (s *ImagePostgresDataStoreTestSuite) TestUpdateVulnStateWithPostgres() {
	image := fixtures.GetImageWithUniqueComponents(5)
	ctx := sac.WithAllAccess(context.Background())

	s.NoError(s.datastore.UpsertImage(ctx, image))
	_, found, err := s.datastore.GetImage(ctx, image.GetId())
	s.NoError(err)
	s.True(found)

	cloned := image.Clone()
	cloned.Id = "cloned"
	s.NoError(s.datastore.UpsertImage(ctx, cloned))
	_, found, err = s.datastore.GetImage(ctx, cloned.GetId())
	s.NoError(err)
	s.True(found)

	results, err := s.datastore.Search(ctx, pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.VulnerabilityState, storage.VulnerabilityState_DEFERRED.String()).ProtoQuery())
	s.NoError(err)
	s.Len(results, 0)

	var snoozedCVEs, unsnoozedCVEs set.Set[string]
	for _, component := range cloned.GetScan().GetComponents() {
		s.Require().GreaterOrEqual(len(component.GetVulns()), 2)

		snoozedCVE := component.GetVulns()[0].GetCve()
		err := s.datastore.UpdateVulnerabilityState(ctx, snoozedCVE, []string{cloned.GetId()}, storage.VulnerabilityState_DEFERRED)
		s.NoError(err)
		snoozedCVEs.Add(snoozedCVE)
		unsnoozedCVEs.Add(component.GetVulns()[1].GetCve())
	}

	// Test serialized data is in sync.
	storedImage, found, err := s.datastore.GetImage(ctx, cloned.GetId())
	s.NoError(err)
	s.True(found)
	for _, component := range storedImage.GetScan().GetComponents() {
		for _, vuln := range component.GetVulns() {
			if snoozedCVEs.Contains(vuln.GetCve()) {
				s.Equal(vuln.GetState(), storage.VulnerabilityState_DEFERRED)
			} else {
				s.Equal(vuln.GetState(), storage.VulnerabilityState_OBSERVED)

			}
		}
	}

	results, err = s.datastore.Search(ctx, pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.VulnerabilityState, storage.VulnerabilityState_DEFERRED.String()).ProtoQuery())
	s.NoError(err)
	s.Len(results, 1)
	s.Equal(cloned.GetId(), results[0].ID)

	// There are still some unsnoozed vulnerabilities in both images.
	results, err = s.datastore.Search(ctx, pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.VulnerabilityState, storage.VulnerabilityState_OBSERVED.String()).ProtoQuery())
	s.NoError(err)
	s.Len(results, 2)
	s.ElementsMatch([]string{image.GetId(), cloned.GetId()}, pkgSearch.ResultsToIDs(results))

	results, err = s.datastore.Search(ctx, pkgSearch.ConjunctionQuery(
		pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.VulnerabilityState, storage.VulnerabilityState_DEFERRED.String()).ProtoQuery(),
		pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.CVE, unsnoozedCVEs.AsSlice()...).ProtoQuery(),
	))
	s.NoError(err)
	s.Len(results, 0)

	results, err = s.datastore.Search(ctx, pkgSearch.ConjunctionQuery(
		pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.VulnerabilityState, storage.VulnerabilityState_OBSERVED.String()).ProtoQuery(),
		pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.CVE, unsnoozedCVEs.AsSlice()...).ProtoQuery(),
	))
	s.NoError(err)
	s.Len(results, 2)
	s.ElementsMatch([]string{image.GetId(), cloned.GetId()}, pkgSearch.ResultsToIDs(results))
}

// Test sort by Component search label sorts by Component+Version to ensure backward compatibility.
func (s *ImagePostgresDataStoreTestSuite) TestSortByComponent() {
	ctx := sac.WithAllAccess(context.Background())
	node := fixtures.GetImageWithUniqueComponents(5)
	componentIDs := make([]string, 0, len(node.GetScan().GetComponents()))
	for _, component := range node.GetScan().GetComponents() {
		componentIDs = append(componentIDs,
			scancomponent.ComponentID(
				component.GetName(),
				component.GetVersion(),
				node.GetScan().GetOperatingSystem(),
			))
	}

	s.NoError(s.datastore.UpsertImage(ctx, node))

	// Verify sort by Component search label is transformed to sort by Component+Version.
	query := pkgSearch.EmptyQuery()
	query.Pagination = &v1.QueryPagination{
		SortOptions: []*v1.QuerySortOption{
			{
				Field: pkgSearch.Component.String(),
			},
		},
	}
	// Component ID is Name+Version+Operating System. Therefore, sort by ID is same as Component+Version.
	sort.SliceStable(componentIDs, func(i, j int) bool {
		return componentIDs[i] < componentIDs[j]
	})
	results, err := s.componentDataStore.Search(ctx, query)
	s.NoError(err)
	s.Equal(componentIDs, pkgSearch.ResultsToIDs(results))

	// Verify reverse sort.
	sort.SliceStable(componentIDs, func(i, j int) bool {
		return componentIDs[i] > componentIDs[j]
	})
	query.Pagination.SortOptions[0].Reversed = true
	results, err = s.componentDataStore.Search(ctx, query)
	s.NoError(err)
	s.Equal(componentIDs, pkgSearch.ResultsToIDs(results))
}

func (s *ImagePostgresDataStoreTestSuite) TestImageDeletes() {
	ctx := sac.WithAllAccess(context.Background())
	testImage := fixtures.GetImageWithUniqueComponents(5)
	s.NoError(s.datastore.UpsertImage(ctx, testImage))

	storedImage, found, err := s.datastore.GetImage(ctx, testImage.GetId())
	s.NoError(err)
	s.True(found)
	for _, component := range testImage.GetScan().GetComponents() {
		for _, cve := range component.GetVulns() {
			cve.FirstSystemOccurrence = storedImage.GetLastUpdated()
			cve.FirstImageOccurrence = storedImage.GetLastUpdated()
			cve.VulnerabilityTypes = []storage.EmbeddedVulnerability_VulnerabilityType{storage.EmbeddedVulnerability_IMAGE_VULNERABILITY}
		}
	}
	expectedImage := cloneAndUpdateRiskPriority(testImage)
	s.Equal(expectedImage, storedImage)

	// Verify that new scan with less components cleans up the old relations correctly.
	testImage.Scan.ScanTime = protoTypes.TimestampNow()
	testImage.Scan.Components = testImage.Scan.Components[:len(testImage.Scan.Components)-1]
	cveIDsSet := set.NewStringSet()
	for _, component := range testImage.GetScan().GetComponents() {
		for _, cve := range component.GetVulns() {
			cveIDsSet.Add(pkgCVE.ID(cve.GetCve(), testImage.GetScan().GetOperatingSystem()))
		}
	}
	s.NoError(s.datastore.UpsertImage(ctx, testImage))

	// Verify image is built correctly.
	storedImage, found, err = s.datastore.GetImage(ctx, testImage.GetId())
	s.NoError(err)
	s.True(found)
	expectedImage = cloneAndUpdateRiskPriority(testImage)
	s.Equal(expectedImage, storedImage)

	// Verify orphaned image components are removed.
	count, err := s.componentDataStore.Count(ctx, pkgSearch.EmptyQuery())
	s.NoError(err)
	s.Equal(len(testImage.Scan.Components), count)

	// Verify orphaned image vulnerabilities are removed.
	results, err := s.cveDataStore.Search(ctx, pkgSearch.EmptyQuery())
	s.NoError(err)
	s.ElementsMatch(cveIDsSet.AsSlice(), pkgSearch.ResultsToIDs(results))

	testImage2 := testImage.Clone()
	testImage2.Id = "2"
	s.NoError(s.datastore.UpsertImage(ctx, testImage2))
	storedImage, found, err = s.datastore.GetImage(ctx, testImage2.GetId())
	s.NoError(err)
	s.True(found)
	for _, component := range testImage2.GetScan().GetComponents() {
		for _, cve := range component.GetVulns() {
			// System Occurrence remains unchanged.
			cve.FirstImageOccurrence = storedImage.GetLastUpdated()
			cve.VulnerabilityTypes = []storage.EmbeddedVulnerability_VulnerabilityType{storage.EmbeddedVulnerability_IMAGE_VULNERABILITY}
		}
	}
	expectedImage = cloneAndUpdateRiskPriority(testImage2)
	s.Equal(expectedImage, storedImage)

	// Verify that number of image components remains unchanged since both images have same components.
	count, err = s.componentDataStore.Count(ctx, pkgSearch.EmptyQuery())
	s.NoError(err)
	s.Equal(len(testImage.Scan.Components), count)

	// Verify that number of image vulnerabilities remains unchanged since both images have same vulns.
	results, err = s.cveDataStore.Search(ctx, pkgSearch.EmptyQuery())
	s.NoError(err)
	s.ElementsMatch(cveIDsSet.AsSlice(), pkgSearch.ResultsToIDs(results))

	s.mockRisk.EXPECT().RemoveRisk(gomock.Any(), testImage.GetId(), gomock.Any()).Return(nil)
	s.NoError(s.datastore.DeleteImages(ctx, testImage.GetId()))

	// Verify that second image is still constructed correctly.
	storedImage, found, err = s.datastore.GetImage(ctx, testImage2.GetId())
	s.NoError(err)
	s.True(found)
	expectedImage = cloneAndUpdateRiskPriority(testImage2)
	s.Equal(expectedImage, storedImage)

	// Set all components to contain same cve.
	for _, component := range testImage2.GetScan().GetComponents() {
		component.Vulns = []*storage.EmbeddedVulnerability{
			{
				Cve:                "cve",
				VulnerabilityType:  storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
				VulnerabilityTypes: []storage.EmbeddedVulnerability_VulnerabilityType{storage.EmbeddedVulnerability_IMAGE_VULNERABILITY},
			},
		}
	}
	testImage2.Scan.ScanTime = protoTypes.TimestampNow()

	s.NoError(s.datastore.UpsertImage(ctx, testImage2))
	storedImage, found, err = s.datastore.GetImage(ctx, testImage2.GetId())
	s.NoError(err)
	s.True(found)
	for _, component := range testImage2.GetScan().GetComponents() {
		// Components and Vulns are deduped, therefore, update testImage structure.
		for _, cve := range component.GetVulns() {
			cve.FirstSystemOccurrence = storedImage.GetLastUpdated()
			cve.FirstImageOccurrence = storedImage.GetLastUpdated()
		}
	}
	expectedImage = cloneAndUpdateRiskPriority(testImage2)
	s.Equal(expectedImage, storedImage)

	// Verify orphaned image components are removed.
	count, err = s.componentDataStore.Count(ctx, pkgSearch.EmptyQuery())
	s.NoError(err)
	s.Equal(len(testImage2.Scan.Components), count)

	// Verify orphaned image vulnerabilities are removed.
	results, err = s.cveDataStore.Search(ctx, pkgSearch.EmptyQuery())
	s.NoError(err)
	s.ElementsMatch([]string{pkgCVE.ID("cve", "")}, pkgSearch.ResultsToIDs(results))

	// Verify that new scan with less components cleans up the old relations correctly.
	testImage2.Scan.ScanTime = protoTypes.TimestampNow()
	testImage2.Scan.Components = testImage2.Scan.Components[:len(testImage2.Scan.Components)-1]
	s.NoError(s.datastore.UpsertImage(ctx, testImage2))

	// Verify image is built correctly.
	storedImage, found, err = s.datastore.GetImage(ctx, testImage2.GetId())
	s.NoError(err)
	s.True(found)
	expectedImage = cloneAndUpdateRiskPriority(testImage2)
	s.Equal(expectedImage, storedImage)

	// Verify orphaned image components are removed.
	count, err = s.componentDataStore.Count(ctx, pkgSearch.EmptyQuery())
	s.NoError(err)
	s.Equal(len(testImage2.Scan.Components), count)

	// Verify no vulnerability is removed since all vulns are still connected.
	results, err = s.cveDataStore.Search(ctx, pkgSearch.EmptyQuery())
	s.NoError(err)
	s.ElementsMatch([]string{pkgCVE.ID("cve", "")}, pkgSearch.ResultsToIDs(results))

	// Verify that new scan with no components and vulns cleans up the old relations correctly.
	testImage2.Scan.ScanTime = protoTypes.TimestampNow()
	testImage2.Scan.Components = nil
	s.NoError(s.datastore.UpsertImage(ctx, testImage2))

	// Verify image is built correctly.
	storedImage, found, err = s.datastore.GetImage(ctx, testImage2.GetId())
	s.NoError(err)
	s.True(found)
	expectedImage = cloneAndUpdateRiskPriority(testImage2)
	s.Equal(expectedImage, storedImage)

	// Verify no components exist.
	count, err = s.componentDataStore.Count(ctx, pkgSearch.EmptyQuery())
	s.NoError(err)
	s.Equal(0, count)

	// Verify no vulnerabilities exist.
	count, err = s.cveDataStore.Count(ctx, pkgSearch.EmptyQuery())
	s.NoError(err)
	s.Equal(0, count)

	// Delete image.
	s.mockRisk.EXPECT().RemoveRisk(gomock.Any(), testImage2.GetId(), gomock.Any()).Return(nil)
	s.NoError(s.datastore.DeleteImages(ctx, testImage2.GetId()))

	// Verify no images exist.
	count, err = s.datastore.Count(ctx, pkgSearch.EmptyQuery())
	s.NoError(err)
	s.Equal(0, count)
}

func (s *ImagePostgresDataStoreTestSuite) TestGetManyImageMetadata() {
	ctx := sac.WithAllAccess(context.Background())
	testImage1 := fixtures.GetImageWithUniqueComponents(5)
	s.NoError(s.datastore.UpsertImage(ctx, testImage1))

	testImage2 := testImage1.Clone()
	testImage2.Id = "2"
	s.NoError(s.datastore.UpsertImage(ctx, testImage2))

	testImage3 := testImage1.Clone()
	testImage3.Id = "3"
	s.NoError(s.datastore.UpsertImage(ctx, testImage3))

	storedImages, err := s.datastore.GetManyImageMetadata(ctx, []string{testImage1.Id, testImage2.Id, testImage3.Id})
	s.NoError(err)
	s.Len(storedImages, 3)

	testImage1.Scan.Components = nil
	testImage1.Priority = 1
	testImage2.Scan.Components = nil
	testImage2.Priority = 1
	testImage3.Scan.Components = nil
	testImage3.Priority = 1
	s.ElementsMatch([]*storage.Image{testImage1, testImage2, testImage3}, storedImages)
}

func getTestImage(id string) *storage.Image {
	return &storage.Image{
		Id: id,
		Scan: &storage.ImageScan{
			OperatingSystem: "blah",
			ScanTime:        protoTypes.TimestampNow(),
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
							CvssV3: &storage.CVSSV3{
								ImpactScore: 10,
							},
							ScoreVersion: storage.EmbeddedVulnerability_V3,
						},
						{
							Cve:               "cve2",
							VulnerabilityType: storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
							SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
								FixedBy: "ver3",
							},
							CvssV3: &storage.CVSSV3{
								ImpactScore: 1,
							},
							ScoreVersion: storage.EmbeddedVulnerability_V3,
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
							CvssV3: &storage.CVSSV3{
								ImpactScore: 10,
							},
							ScoreVersion: storage.EmbeddedVulnerability_V3,
						},
						{
							Cve:               "cve2",
							VulnerabilityType: storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
							CvssV3: &storage.CVSSV3{
								ImpactScore: 1,
							},
							ScoreVersion: storage.EmbeddedVulnerability_V3,
						},
					},
				},
			},
		},
		RiskScore: 30,
		Priority:  1,
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
