//go:build sql_integration

package datastore

import (
	"context"
	"fmt"
	"sort"
	"testing"

	imageCVEDS "github.com/stackrox/rox/central/cve/image/v2/datastore"
	imageCVEPostgres "github.com/stackrox/rox/central/cve/image/v2/datastore/store/postgres"
	"github.com/stackrox/rox/central/image/datastore/keyfence"
	pgStoreV2 "github.com/stackrox/rox/central/image/datastore/store/v2/postgres"
	imageComponentDS "github.com/stackrox/rox/central/imagecomponent/v2/datastore"
	imageComponentSearch "github.com/stackrox/rox/central/imagecomponent/v2/datastore/search"
	imageComponentPostgres "github.com/stackrox/rox/central/imagecomponent/v2/datastore/store/postgres"
	"github.com/stackrox/rox/central/ranking"
	mockRisks "github.com/stackrox/rox/central/risk/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	pkgCVE "github.com/stackrox/rox/pkg/cve"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	postgresSchema "github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/scancomponent"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/scoped"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestImageFlatDataStoreWithPostgres(t *testing.T) {
	if !features.FlattenCVEData.Enabled() {
		t.Skip("CVE flattened data model is not enabled")
	}
	suite.Run(t, new(ImageFlatPostgresDataStoreTestSuite))
}

type ImageFlatPostgresDataStoreTestSuite struct {
	suite.Suite

	ctx                context.Context
	testDB             *pgtest.TestPostgres
	db                 postgres.DB
	datastore          DataStore
	mockRisk           *mockRisks.MockDataStore
	componentDataStore imageComponentDS.DataStore
	cveDataStore       imageCVEDS.DataStore
}

func (s *ImageFlatPostgresDataStoreTestSuite) SetupSuite() {
	s.ctx = context.Background()
	s.testDB = pgtest.ForT(s.T())
	s.db = s.testDB.DB
}

func (s *ImageFlatPostgresDataStoreTestSuite) SetupTest() {

	s.mockRisk = mockRisks.NewMockDataStore(gomock.NewController(s.T()))
	dbStore := pgStoreV2.New(s.db, false, keyfence.ImageKeyFenceSingleton())
	s.datastore = NewWithPostgres(dbStore, s.mockRisk, ranking.ImageRanker(), ranking.ComponentRanker())

	componentStorage := imageComponentPostgres.New(s.db)
	componentSearcher := imageComponentSearch.NewV2(componentStorage)
	s.componentDataStore = imageComponentDS.New(componentStorage, componentSearcher, s.mockRisk, ranking.NewRanker())

	cveStorage := imageCVEPostgres.New(s.db)
	cveDataStore := imageCVEDS.New(cveStorage)
	s.cveDataStore = cveDataStore
}

func (s *ImageFlatPostgresDataStoreTestSuite) TearDownTest() {
	s.truncateTable(postgresSchema.DeploymentsTableName)
	s.truncateTable(postgresSchema.ImagesTableName)
	s.truncateTable(postgresSchema.ImageComponentV2TableName)
	s.truncateTable(postgresSchema.ImageCvesV2TableName)
}

func (s *ImageFlatPostgresDataStoreTestSuite) TestSearchWithPostgres() {
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
	s.Equal([]string{"cve2", "cve1"}, splitFlattenedIDs(pkgSearch.ResultsToIDs(results)))

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
	s.Equal([]string{"cve1", "cve2"}, splitFlattenedIDs(pkgSearch.ResultsToIDs(results)))

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
		IDs:   []string{image.GetId()},
		Level: v1.SearchCategory_IMAGES,
	})
	results, err = s.datastore.Search(scopedCtx, pkgSearch.EmptyQuery())
	s.NoError(err)
	s.Len(results, 1)
	s.Equal(image.GetId(), results[0].ID)

	// Need to grab a CVE for the image to scope since we can not easily build the ID any longer.
	q = pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.CVE, "cve3").ProtoQuery()
	results, err = s.cveDataStore.Search(ctx, q)
	s.NoError(err)

	// Scope search by vulns.
	scopedCtx = scoped.Context(ctx, scoped.Scope{
		IDs:   []string{results[0].ID},
		Level: v1.SearchCategory_IMAGE_VULNERABILITIES_V2,
	})
	results, err = s.datastore.Search(scopedCtx, pkgSearch.EmptyQuery())
	s.NoError(err)
	s.Len(results, 1)
	s.Equal(newImage.GetId(), results[0].ID)
}

func (s *ImageFlatPostgresDataStoreTestSuite) TestFixableWithPostgres() {
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

	image.Scan.ScanTime = protocompat.TimestampNow()
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

func (s *ImageFlatPostgresDataStoreTestSuite) TestUpdateVulnStateWithPostgres() {
	image := fixtures.GetImageWithUniqueComponents(5)
	ctx := sac.WithAllAccess(context.Background())

	s.NoError(s.datastore.UpsertImage(ctx, image))
	_, found, err := s.datastore.GetImage(ctx, image.GetId())
	s.NoError(err)
	s.True(found)

	cloned := image.CloneVT()
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
func (s *ImageFlatPostgresDataStoreTestSuite) TestSortByComponent() {
	ctx := sac.WithAllAccess(context.Background())
	image := fixtures.GetImageWithUniqueComponents(5)
	componentIDs := make([]string, 0, len(image.GetScan().GetComponents()))
	for _, component := range image.GetScan().GetComponents() {
		compID, err := scancomponent.ComponentIDV2(
			component,
			image.GetId(),
		)
		s.NoError(err)
		componentIDs = append(componentIDs, compID)
	}

	s.NoError(s.datastore.UpsertImage(ctx, image))

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

func (s *ImageFlatPostgresDataStoreTestSuite) TestImageDeletes() {
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
	protoassert.Equal(s.T(), expectedImage, storedImage)

	// Verify that new scan with less components cleans up the old relations correctly.
	testImage.Scan.ScanTime = protocompat.TimestampNow()
	testImage.Scan.Components = testImage.Scan.Components[:len(testImage.Scan.Components)-1]
	cveIDsSet := set.NewStringSet()
	for _, component := range testImage.GetScan().GetComponents() {
		componentID, err := scancomponent.ComponentIDV2(component, testImage.GetId())
		s.NoError(err)
		for _, cve := range component.GetVulns() {
			cveID, err := pkgCVE.IDV2(cve, componentID)
			s.NoError(err)
			cveIDsSet.Add(cveID)
		}
	}
	s.NoError(s.datastore.UpsertImage(ctx, testImage))

	// Verify image is built correctly.
	storedImage, found, err = s.datastore.GetImage(ctx, testImage.GetId())
	s.NoError(err)
	s.True(found)
	expectedImage = cloneAndUpdateRiskPriority(testImage)
	protoassert.Equal(s.T(), expectedImage, storedImage)

	// Verify orphaned image components are removed.
	count, err := s.componentDataStore.Count(ctx, pkgSearch.EmptyQuery())
	s.NoError(err)
	s.Equal(len(testImage.Scan.Components), count)

	// Verify orphaned image vulnerabilities are removed.
	results, err := s.cveDataStore.Search(ctx, pkgSearch.EmptyQuery())
	s.NoError(err)
	s.ElementsMatch(cveIDsSet.AsSlice(), pkgSearch.ResultsToIDs(results))

	testImage2 := testImage.CloneVT()
	testImage2.Id = "2"
	for _, component := range testImage2.GetScan().GetComponents() {
		for _, cve := range component.GetVulns() {
			// Clone brings over the time, need to empty that out
			cve.FirstImageOccurrence = nil
		}
	}
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
	protoassert.Equal(s.T(), expectedImage, storedImage)

	s.mockRisk.EXPECT().RemoveRisk(gomock.Any(), testImage.GetId(), gomock.Any()).Return(nil)
	s.NoError(s.datastore.DeleteImages(ctx, testImage.GetId()))

	// Verify that second image is still constructed correctly.
	storedImage, found, err = s.datastore.GetImage(ctx, testImage2.GetId())
	s.NoError(err)
	s.True(found)
	expectedImage = cloneAndUpdateRiskPriority(testImage2)
	protoassert.Equal(s.T(), expectedImage, storedImage)

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
	testImage2.Scan.ScanTime = protocompat.TimestampNow()

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
	protoassert.Equal(s.T(), expectedImage, storedImage)

	// Verify orphaned image components are removed.
	count, err = s.componentDataStore.Count(ctx, pkgSearch.EmptyQuery())
	s.NoError(err)
	s.Equal(len(testImage2.Scan.Components), count)

	// Verify orphaned image vulnerabilities are removed.
	results, err = s.cveDataStore.Search(ctx, pkgSearch.EmptyQuery())
	s.NoError(err)
	// split the IDs to only get the CVE name and make sure they all match this specific one
	s.ElementsMatch([]string{"cve"}, splitFlattenedIDs(pkgSearch.ResultsToIDs(results)))

	// Verify that new scan with fewer components cleans up the old relations correctly.
	testImage2.Scan.ScanTime = protocompat.TimestampNow()
	testImage2.Scan.Components = testImage2.Scan.Components[:len(testImage2.Scan.Components)-1]
	s.NoError(s.datastore.UpsertImage(ctx, testImage2))

	// Verify image is built correctly.
	storedImage, found, err = s.datastore.GetImage(ctx, testImage2.GetId())
	s.NoError(err)
	s.True(found)
	expectedImage = cloneAndUpdateRiskPriority(testImage2)
	protoassert.Equal(s.T(), expectedImage, storedImage)

	// Verify orphaned image components are removed.
	count, err = s.componentDataStore.Count(ctx, pkgSearch.EmptyQuery())
	s.NoError(err)
	s.Equal(len(testImage2.Scan.Components), count)

	// Verify no vulnerability is removed since all vulns are still connected.
	results, err = s.cveDataStore.Search(ctx, pkgSearch.EmptyQuery())
	s.NoError(err)
	s.ElementsMatch([]string{"cve"}, splitFlattenedIDs(pkgSearch.ResultsToIDs(results)))

	// Verify that new scan with no components and vulns cleans up the old relations correctly.
	testImage2.Scan.ScanTime = protocompat.TimestampNow()
	testImage2.Scan.Components = nil
	s.NoError(s.datastore.UpsertImage(ctx, testImage2))

	// Verify image is built correctly.
	storedImage, found, err = s.datastore.GetImage(ctx, testImage2.GetId())
	s.NoError(err)
	s.True(found)
	expectedImage = cloneAndUpdateRiskPriority(testImage2)
	protoassert.Equal(s.T(), expectedImage, storedImage)

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

func (s *ImageFlatPostgresDataStoreTestSuite) TestGetManyImageMetadata() {
	ctx := sac.WithAllAccess(context.Background())
	testImage1 := fixtures.GetImageWithUniqueComponents(5)
	s.NoError(s.datastore.UpsertImage(ctx, testImage1))

	testImage2 := testImage1.CloneVT()
	testImage2.Id = "2"
	s.NoError(s.datastore.UpsertImage(ctx, testImage2))

	testImage3 := testImage1.CloneVT()
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
	protoassert.ElementsMatch(s.T(), []*storage.Image{testImage1, testImage2, testImage3}, storedImages)
}

func (s *ImageFlatPostgresDataStoreTestSuite) truncateTable(name string) {
	sql := fmt.Sprintf("TRUNCATE %s CASCADE", name)
	_, err := s.testDB.Exec(s.ctx, sql)
	s.NoError(err)
}
