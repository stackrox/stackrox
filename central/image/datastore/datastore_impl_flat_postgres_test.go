//go:build sql_integration

package datastore

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

	imageCVEInfoDS "github.com/stackrox/rox/central/cve/image/info/datastore"
	imageCVEInfoPostgres "github.com/stackrox/rox/central/cve/image/info/datastore/store/postgres"
	cveInfoEnricher "github.com/stackrox/rox/central/cve/image/info/enricher"
	imageCVEDS "github.com/stackrox/rox/central/cve/image/v2/datastore"
	imageCVEPostgres "github.com/stackrox/rox/central/cve/image/v2/datastore/store/postgres"
	"github.com/stackrox/rox/central/image/datastore/keyfence"
	pgStoreV2 "github.com/stackrox/rox/central/image/datastore/store/v2/postgres"
	imageComponentDS "github.com/stackrox/rox/central/imagecomponent/v2/datastore"
	imageComponentPostgres "github.com/stackrox/rox/central/imagecomponent/v2/datastore/store/postgres"
	"github.com/stackrox/rox/central/ranking"
	mockRisks "github.com/stackrox/rox/central/risk/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	pkgCVE "github.com/stackrox/rox/pkg/cve"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
	imageEnricher "github.com/stackrox/rox/pkg/images/enricher"
	imageTypes "github.com/stackrox/rox/pkg/images/types"
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
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestImageFlatDataStoreWithPostgres(t *testing.T) {
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
	cveInfoDataStore   imageCVEInfoDS.DataStore
	cveInfoEnricher    imageEnricher.CVEInfoEnricher
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
	s.componentDataStore = imageComponentDS.New(componentStorage, s.mockRisk, ranking.NewRanker())

	cveStorage := imageCVEPostgres.New(s.db)
	cveDataStore := imageCVEDS.New(cveStorage)
	s.cveDataStore = cveDataStore

	cveInfoStorage := imageCVEInfoPostgres.New(s.db)
	s.cveInfoDataStore = imageCVEInfoDS.New(cveInfoStorage)
	s.cveInfoEnricher = cveInfoEnricher.New(s.cveInfoDataStore)
}

func (s *ImageFlatPostgresDataStoreTestSuite) TearDownTest() {
	s.truncateTable(postgresSchema.DeploymentsTableName)
	s.truncateTable(postgresSchema.ImagesTableName)
	s.truncateTable(postgresSchema.ImageComponentV2TableName)
	s.truncateTable(postgresSchema.ImageCvesV2TableName)
	s.truncateTable(postgresSchema.ImageCveInfosTableName)
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
	searchRes, errRes := s.datastore.SearchImages(ctx, q)
	s.NoError(errRes)
	s.Len(searchRes, 1)
	s.Equal(imageTypes.NewDigest(image.GetId()).Digest(), searchRes[0].GetId())
	s.Equal(image.GetName().GetFullName(), searchRes[0].GetName())

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
	for index, component := range image.GetScan().GetComponents() {
		compID := scancomponent.ComponentIDV2(
			component,
			image.GetId(),
			index,
		)
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
	for compI, component := range testImage.GetScan().GetComponents() {
		for cveI, cve := range component.GetVulns() {
			if features.CVEFixTimestampCriteria.Enabled() {
				cve.FirstSystemOccurrence = storedImage.GetScan().GetComponents()[compI].GetVulns()[cveI].GetFirstSystemOccurrence()
				cve.FixAvailableTimestamp = storedImage.GetScan().GetComponents()[compI].GetVulns()[cveI].GetFixAvailableTimestamp()
			} else {
				cve.FirstSystemOccurrence = storedImage.GetLastUpdated()
			}
			cve.FirstImageOccurrence = storedImage.GetLastUpdated()
			cve.VulnerabilityTypes = []storage.EmbeddedVulnerability_VulnerabilityType{storage.EmbeddedVulnerability_IMAGE_VULNERABILITY}
		}
	}
	expectedImage := cloneAndUpdateRiskPriority(testImage)
	protoassert.Equal(s.T(), expectedImage, storedImage)

	// Verify that new scan with less components cleans up the old relations correctly.
	testImage.Scan.ScanTime = protocompat.TimestampNow()
	testImage.Scan.Components = testImage.GetScan().GetComponents()[:len(testImage.GetScan().GetComponents())-1]
	cveIDsSet := set.NewStringSet()
	for compIndex, component := range testImage.GetScan().GetComponents() {
		componentID := scancomponent.ComponentIDV2(component, testImage.GetId(), compIndex)
		for cveIndex, cve := range component.GetVulns() {
			cveID := pkgCVE.IDV2(cve, componentID, cveIndex)
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
	s.Equal(len(testImage.GetScan().GetComponents()), count)

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
			// System Occurrence and fix available times remain unchanged.
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
	for compI, component := range testImage2.GetScan().GetComponents() {
		// Components and Vulns are deduped, therefore, update testImage structure.
		for cveI, cve := range component.GetVulns() {
			if features.CVEFixTimestampCriteria.Enabled() {
				cve.FirstSystemOccurrence = storedImage.GetScan().GetComponents()[compI].GetVulns()[cveI].GetFirstSystemOccurrence()
				cve.FixAvailableTimestamp = storedImage.GetScan().GetComponents()[compI].GetVulns()[cveI].GetFixAvailableTimestamp()
			} else {
				cve.FirstSystemOccurrence = storedImage.GetLastUpdated()
			}
			cve.FirstImageOccurrence = storedImage.GetLastUpdated()
		}
	}
	expectedImage = cloneAndUpdateRiskPriority(testImage2)
	protoassert.Equal(s.T(), expectedImage, storedImage)

	// Verify orphaned image components are removed.
	count, err = s.componentDataStore.Count(ctx, pkgSearch.EmptyQuery())
	s.NoError(err)
	s.Equal(len(testImage2.GetScan().GetComponents()), count)

	// Verify orphaned image vulnerabilities are removed.
	results, err = s.cveDataStore.Search(ctx, pkgSearch.EmptyQuery())
	s.NoError(err)
	// split the IDs to only get the CVE name and make sure they all match this specific one
	s.ElementsMatch([]string{"cve"}, splitFlattenedIDs(pkgSearch.ResultsToIDs(results)))

	// Verify that new scan with fewer components cleans up the old relations correctly.
	testImage2.Scan.ScanTime = protocompat.TimestampNow()
	testImage2.Scan.Components = testImage2.GetScan().GetComponents()[:len(testImage2.GetScan().GetComponents())-1]
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
	s.Equal(len(testImage2.GetScan().GetComponents()), count)

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

	storedImages, err := s.datastore.GetManyImageMetadata(ctx, []string{testImage1.GetId(), testImage2.GetId(), testImage3.GetId()})
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

func (s *ImageFlatPostgresDataStoreTestSuite) TestCVETimestampPersistence() {
	s.T().Setenv(features.CVEFixTimestampCriteria.EnvVar(), "true")
	if !features.CVEFixTimestampCriteria.Enabled() {
		s.T().Skip("CVEFixTimestampCriteria feature must be enabled for this test")
	}

	ctx := sac.WithAllAccess(context.Background())

	// Scanner-provided timestamp for when the shared CVE was first discovered
	cveDiscoverTimestamp := timestamppb.New(time.Now().Add(-24 * time.Hour))

	sharedCVEID := "CVE-2024-1234"
	datasource := "alpine:v3.18"
	image0ID := "image0-sha"

	// Three images share a CVE but each also has unique CVEs.
	// The shared CVE in the second image has an earlier FirstSystemOccurrence timestamp.
	images := []*storage.Image{
		{
			Id: image0ID,
			Name: &storage.ImageName{
				FullName: "registry.io/image0:v0",
			},
			Scan: &storage.ImageScan{
				OperatingSystem: "alpine",
				ScanTime:        timestamppb.Now(),
				Components: []*storage.EmbeddedImageScanComponent{
					{
						Name:    "shared-component",
						Version: "1.0.0",
						Source:  storage.SourceType_OS,
						Vulns: []*storage.EmbeddedVulnerability{
							{
								Cve:                   sharedCVEID,
								VulnerabilityType:     storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
								Datasource:            datasource,
								Severity:              storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
								FirstSystemOccurrence: timestamppb.Now(),
							},
							{
								Cve:                   "CVE-2024-1111",
								VulnerabilityType:     storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
								Datasource:            datasource,
								FirstSystemOccurrence: timestamppb.Now(),
							},
						},
					},
				},
			},
		},
		{
			Id: "image1-sha",
			Name: &storage.ImageName{
				FullName: "registry.io/image1:v1",
			},
			Scan: &storage.ImageScan{
				OperatingSystem: "alpine",
				ScanTime:        timestamppb.Now(),
				Components: []*storage.EmbeddedImageScanComponent{
					{
						Name:    "shared-component",
						Version: "1.0.0",
						Source:  storage.SourceType_OS,
						Vulns: []*storage.EmbeddedVulnerability{
							{
								Cve:                   sharedCVEID,
								VulnerabilityType:     storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
								Datasource:            datasource,
								Severity:              storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
								FirstSystemOccurrence: cveDiscoverTimestamp, // Earlier time stamp
							},
							{
								Cve:                   "CVE-2024-5678",
								VulnerabilityType:     storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
								Datasource:            datasource,
								FirstSystemOccurrence: timestamppb.Now(),
							},
						},
					},
				},
			},
		},
		{
			Id: "image2-sha",
			Name: &storage.ImageName{
				FullName: "registry.io/image2:v2",
			},
			Scan: &storage.ImageScan{
				OperatingSystem: "alpine",
				ScanTime:        timestamppb.Now(),
				Components: []*storage.EmbeddedImageScanComponent{
					{
						Name:    "shared-component",
						Version: "1.0.0",
						Source:  storage.SourceType_OS,
						Vulns: []*storage.EmbeddedVulnerability{
							{
								Cve:                   sharedCVEID,
								VulnerabilityType:     storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
								Datasource:            datasource,
								Severity:              storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
								FirstSystemOccurrence: timestamppb.Now(),
							},
							{
								Cve:                   "CVE-2024-9999",
								VulnerabilityType:     storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
								Datasource:            datasource,
								FirstSystemOccurrence: timestamppb.Now(),
							},
						},
					},
				},
			},
		},
	}

	// First pass: process all images in order.
	// With the new CVE-name-based aggregation, images processed after image1 will be enriched
	// with the earliest timestamp across ALL records for that CVE name.
	for _, img := range images {
		imgClone := img.CloneVT()
		s.NoError(s.cveInfoEnricher.EnrichImageWithCVEInfo(ctx, imgClone))
		s.NoError(s.datastore.UpsertImage(ctx, imgClone))
	}

	// Verify ImageCVEInfo persisted the earliest timestamp for the specific composite ID
	cveInfoID := pkgCVE.ImageCVEInfoID(sharedCVEID, "shared-component", datasource)
	cveInfo, found, err := s.cveInfoDataStore.Get(ctx, cveInfoID)
	s.NoError(err)
	s.True(found)
	s.Equal(cveDiscoverTimestamp, cveInfo.GetFirstSystemOccurrence())

	// Verify first pass results:
	// With CVE-name-based aggregation:
	// - image0: shared CVE should NOT have cveDiscoverTimestamp (processed before image1)
	// - image1: shared CVE has cveDiscoverTimestamp (source of early timestamp)
	// - image2: shared CVE should have cveDiscoverTimestamp (enriched from MIN across all CVE-2024-1234 records)
	// - All unique CVEs should have their own timestamps
	for i, img := range images {
		stored, found, err := s.datastore.GetImage(ctx, img.GetId())
		s.NoError(err)
		s.True(found)

		components := stored.GetScan().GetComponents()
		s.Require().Len(components, 1)
		s.Require().Len(components[0].GetVulns(), 2)

		for _, vuln := range components[0].GetVulns() {
			if vuln.GetCve() != sharedCVEID {
				s.NotEqual(cveDiscoverTimestamp, vuln.GetFirstSystemOccurrence(),
					"Unique CVE should not have the shared timestamp (image %d)", i)
			} else if img.GetId() == image0ID {
				s.NotEqual(cveDiscoverTimestamp, vuln.GetFirstSystemOccurrence(),
					"image0 was processed before image1, should not have the earlier timestamp yet")
			} else {
				s.Equal(cveDiscoverTimestamp, vuln.GetFirstSystemOccurrence(),
					"Shared CVE should have the earlier timestamp from image1 (image %d)", i)
			}
		}
	}

	// Second pass: rescan all images.
	// All images should now have the preserved earliest timestamp for the shared CVE,
	// since the enricher queries by CVE name and returns the MIN across all records.
	for _, img := range images {
		imgClone := img.CloneVT()
		imgClone.GetScan().ScanTime = timestamppb.Now()
		s.NoError(s.cveInfoEnricher.EnrichImageWithCVEInfo(ctx, imgClone))
		s.NoError(s.datastore.UpsertImage(ctx, imgClone))
	}

	// Verify all images now have the preserved earliest timestamp
	// The CVE-name-based aggregation ensures all images get the MIN timestamp for CVE-2024-1234
	for i, img := range images {
		stored, found, err := s.datastore.GetImage(ctx, img.GetId())
		s.NoError(err)
		s.True(found)

		components := stored.GetScan().GetComponents()
		s.Require().Len(components, 1)
		s.Require().Len(components[0].GetVulns(), 2)

		for _, vuln := range components[0].GetVulns() {
			if vuln.GetCve() == sharedCVEID {
				s.Equal(cveDiscoverTimestamp, vuln.GetFirstSystemOccurrence(),
					"After second pass, all images should have the preserved earliest timestamp for shared CVE (image %d)", i)
			} else {
				s.NotEqual(cveDiscoverTimestamp, vuln.GetFirstSystemOccurrence(),
					"Unique CVE should not have the shared timestamp (image %d)", i)
			}
		}
	}
}

func (s *ImageFlatPostgresDataStoreTestSuite) TestCVETimestampAggregation() {
	s.T().Setenv(features.CVEFixTimestampCriteria.EnvVar(), "true")
	if !features.CVEFixTimestampCriteria.Enabled() {
		s.T().Skip("CVEFixTimestampCriteria feature must be enabled for this test")
	}

	ctx := sac.WithAllAccess(context.Background())

	// Two images with the same CVE name but different components and operating systems
	sharedCVEName := "CVE-2024-SHARED"
	earlierTimestamp := timestamppb.New(time.Now().Add(-48 * time.Hour))
	laterTimestamp := timestamppb.New(time.Now().Add(-24 * time.Hour))

	// Image 1: Ubuntu with package-a, has the earlier timestamp
	image1 := &storage.Image{
		Id: "image1-sha",
		Name: &storage.ImageName{
			FullName: "registry.io/ubuntu-app:v1",
		},
		Scan: &storage.ImageScan{
			OperatingSystem: "ubuntu",
			ScanTime:        timestamppb.Now(),
			Components: []*storage.EmbeddedImageScanComponent{
				{
					Name:    "package-a",
					Version: "1.0.0",
					Source:  storage.SourceType_OS,
					Vulns: []*storage.EmbeddedVulnerability{
						{
							Cve:                   sharedCVEName,
							VulnerabilityType:     storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
							Datasource:            "ubuntu-updater::ubuntu:20.04",
							Severity:              storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
							FirstSystemOccurrence: earlierTimestamp,
						},
					},
				},
			},
		},
	}

	// Image 2: Alpine with package-b (different component and OS), has the later timestamp
	image2 := &storage.Image{
		Id: "image2-sha",
		Name: &storage.ImageName{
			FullName: "registry.io/alpine-app:v1",
		},
		Scan: &storage.ImageScan{
			OperatingSystem: "alpine",
			ScanTime:        timestamppb.Now(),
			Components: []*storage.EmbeddedImageScanComponent{
				{
					Name:    "package-b",
					Version: "2.0.0",
					Source:  storage.SourceType_OS,
					Vulns: []*storage.EmbeddedVulnerability{
						{
							Cve:                   sharedCVEName,
							VulnerabilityType:     storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
							Datasource:            "alpine-updater::alpine:3.18",
							Severity:              storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY,
							FirstSystemOccurrence: laterTimestamp,
						},
					},
				},
			},
		},
	}

	// Process image1 first (has earlier timestamp)
	img1Clone := image1.CloneVT()
	s.NoError(s.cveInfoEnricher.EnrichImageWithCVEInfo(ctx, img1Clone))
	s.NoError(s.datastore.UpsertImage(ctx, img1Clone))

	// Process image2 (has later timestamp)
	img2Clone := image2.CloneVT()
	s.NoError(s.cveInfoEnricher.EnrichImageWithCVEInfo(ctx, img2Clone))
	s.NoError(s.datastore.UpsertImage(ctx, img2Clone))

	// Verify that two separate ImageCVEInfo records exist (different composite IDs)
	cveInfo1ID := pkgCVE.ImageCVEInfoID(sharedCVEName, "package-a", "ubuntu-updater::ubuntu:20.04")
	cveInfo2ID := pkgCVE.ImageCVEInfoID(sharedCVEName, "package-b", "alpine-updater::alpine:3.18")

	cveInfo1, found, err := s.cveInfoDataStore.Get(ctx, cveInfo1ID)
	s.NoError(err)
	s.True(found, "Should have ImageCVEInfo for image1's composite ID")
	s.Equal(earlierTimestamp, cveInfo1.GetFirstSystemOccurrence())

	cveInfo2, found, err := s.cveInfoDataStore.Get(ctx, cveInfo2ID)
	s.NoError(err)
	s.True(found, "Should have ImageCVEInfo for image2's composite ID")
	s.Equal(laterTimestamp, cveInfo2.GetFirstSystemOccurrence())

	// Verify both images have the MIN timestamp (from image1) after enrichment
	storedImage1, found, err := s.datastore.GetImage(ctx, "image1-sha")
	s.NoError(err)
	s.True(found)
	s.Require().Len(storedImage1.GetScan().GetComponents(), 1)
	s.Require().Len(storedImage1.GetScan().GetComponents()[0].GetVulns(), 1)
	s.Equal(earlierTimestamp, storedImage1.GetScan().GetComponents()[0].GetVulns()[0].GetFirstSystemOccurrence(),
		"Image1 should have the earlier timestamp")

	storedImage2, found, err := s.datastore.GetImage(ctx, "image2-sha")
	s.NoError(err)
	s.True(found)
	s.Require().Len(storedImage2.GetScan().GetComponents(), 1)
	s.Require().Len(storedImage2.GetScan().GetComponents()[0].GetVulns(), 1)
	s.Equal(earlierTimestamp, storedImage2.GetScan().GetComponents()[0].GetVulns()[0].GetFirstSystemOccurrence(),
		"Image2 should also have the earlier timestamp from MIN aggregation across all CVE records")

	// Rescan image1 to verify the MIN timestamp is still preserved
	img1Rescan := image1.CloneVT()
	img1Rescan.GetScan().ScanTime = timestamppb.Now()
	s.NoError(s.cveInfoEnricher.EnrichImageWithCVEInfo(ctx, img1Rescan))
	s.NoError(s.datastore.UpsertImage(ctx, img1Rescan))

	storedImage1After, found, err := s.datastore.GetImage(ctx, "image1-sha")
	s.NoError(err)
	s.True(found)
	s.Require().Len(storedImage1After.GetScan().GetComponents(), 1)
	s.Require().Len(storedImage1After.GetScan().GetComponents()[0].GetVulns(), 1)
	s.Equal(earlierTimestamp, storedImage1After.GetScan().GetComponents()[0].GetVulns()[0].GetFirstSystemOccurrence(),
		"After rescan, image1 should still have the MIN timestamp")
}

func (s *ImageFlatPostgresDataStoreTestSuite) truncateTable(name string) {
	sql := fmt.Sprintf("TRUNCATE %s CASCADE", name)
	_, err := s.testDB.Exec(s.ctx, sql)
	s.NoError(err)
}

func getTestImage(id string) *storage.Image {
	return &storage.Image{
		Id: id,
		Name: &storage.ImageName{
			FullName: "remote1/repo1:tag1",
		},
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
	cloned := image.CloneVT()
	cloned.Priority = 1
	for _, component := range cloned.GetScan().GetComponents() {
		component.Priority = 1
	}
	cloned.LastUpdated = image.GetLastUpdated()
	return cloned
}
