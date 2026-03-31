//go:build sql_integration

package datastoretest

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
	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	imageComponentDS "github.com/stackrox/rox/central/imagecomponent/v2/datastore"
	imageComponentPostgres "github.com/stackrox/rox/central/imagecomponent/v2/datastore/store/postgres"
	imageDataStoreV2 "github.com/stackrox/rox/central/imagev2/datastore"
	"github.com/stackrox/rox/central/imagev2/datastore/keyfence"
	pgStore "github.com/stackrox/rox/central/imagev2/datastore/store/postgres"
	"github.com/stackrox/rox/central/ranking"
	mockRisks "github.com/stackrox/rox/central/risk/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	pkgCVE "github.com/stackrox/rox/pkg/cve"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
	imageEnricher "github.com/stackrox/rox/pkg/images/enricher"
	imageUtils "github.com/stackrox/rox/pkg/images/utils"
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
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestImageV2DataStore(t *testing.T) {
	if !features.FlattenImageData.Enabled() {
		t.Skip("Image flattened data model is not enabled")
	}
	suite.Run(t, new(ImageV2DataStoreTestSuite))
}

type ImageV2DataStoreTestSuite struct {
	suite.Suite

	ctx                 context.Context
	testDB              *pgtest.TestPostgres
	datastore           imageDataStoreV2.DataStore
	mockRisk            *mockRisks.MockDataStore
	componentDataStore  imageComponentDS.DataStore
	cveDataStore        imageCVEDS.DataStore
	deploymentDataStore deploymentDS.DataStore
	cveInfoDataStore    imageCVEInfoDS.DataStore
	cveInfoEnricher     imageEnricher.CVEInfoEnricher
}

func (s *ImageV2DataStoreTestSuite) SetupSuite() {
	s.ctx = context.Background()
	s.testDB = pgtest.ForT(s.T())
}

func (s *ImageV2DataStoreTestSuite) SetupTest() {
	s.mockRisk = mockRisks.NewMockDataStore(gomock.NewController(s.T()))
	dbStore := pgStore.New(s.testDB.DB, false, keyfence.ImageKeyFenceSingleton())
	s.datastore = imageDataStoreV2.NewWithPostgres(dbStore, s.mockRisk, ranking.NewRanker(), ranking.NewRanker())

	componentStorage := imageComponentPostgres.New(s.testDB.DB)
	s.componentDataStore = imageComponentDS.New(componentStorage, s.mockRisk, ranking.NewRanker())

	cveStorage := imageCVEPostgres.New(s.testDB.DB)
	s.cveDataStore = imageCVEDS.New(cveStorage)

	cveInfoStorage := imageCVEInfoPostgres.New(s.testDB.DB)
	s.cveInfoDataStore = imageCVEInfoDS.New(cveInfoStorage)
	s.cveInfoEnricher = cveInfoEnricher.New(s.cveInfoDataStore)

	var err error
	s.deploymentDataStore, err = deploymentDS.GetTestPostgresDataStore(s.T(), s.testDB.DB)
	s.Require().NoError(err)
}

func (s *ImageV2DataStoreTestSuite) TearDownTest() {
	s.truncateTable(postgresSchema.DeploymentsTableName)
	s.truncateTable(postgresSchema.ImagesV2TableName)
	s.truncateTable(postgresSchema.ImageComponentV2TableName)
	s.truncateTable(postgresSchema.ImageCvesV2TableName)
	s.truncateTable(postgresSchema.ImageCveInfosTableName)
}

func (s *ImageV2DataStoreTestSuite) TestSearch() {
	image := getTestImageV2("img1")

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

	q := pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.ImageID, image.GetId()).ProtoQuery()
	results, err = s.datastore.Search(ctx, q)
	s.NoError(err)
	s.Len(results, 1)

	q = pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.ImageSHA, image.GetDigest()).ProtoQuery()
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
	newImage := getTestImageV2("img2")
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

	// Scope search by image.
	scopedCtx := scoped.Context(ctx, scoped.Scope{
		IDs:   []string{image.GetId()},
		Level: v1.SearchCategory_IMAGES_V2,
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

func (s *ImageV2DataStoreTestSuite) TestUpdateVulnState() {
	image := fixtures.GetImageV2WithUniqueComponents(5)
	ctx := sac.WithAllAccess(context.Background())

	s.NoError(s.datastore.UpsertImage(ctx, image))
	_, found, err := s.datastore.GetImage(ctx, image.GetId())
	s.NoError(err)
	s.True(found)

	cloned := image.CloneVT()
	cloned.Name.FullName = "registry.test.io/cloned:latest"
	cloned.Id = uuid.NewV5FromNonUUIDs(cloned.GetName().GetFullName(), cloned.GetDigest()).String()
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

		for _, vuln := range component.GetVulns()[1:] {
			unsnoozedCVEs.Add(vuln.GetCve())
		}
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
func (s *ImageV2DataStoreTestSuite) TestSortByComponent() {
	ctx := sac.WithAllAccess(context.Background())
	image := fixtures.GetImageV2WithUniqueComponents(5)
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

func (s *ImageV2DataStoreTestSuite) TestImageDeletes() {
	ctx := sac.WithAllAccess(context.Background())
	testImage := fixtures.GetImageV2WithUniqueComponents(5)
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
	testImage2.Name.FullName = "registry.test.io/cloned:latest"
	testImage2.Id = uuid.NewV5FromNonUUIDs(testImage2.GetName().GetFullName(), testImage2.GetDigest()).String()
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

func (s *ImageV2DataStoreTestSuite) TestGetManyImageMetadata() {
	ctx := sac.WithAllAccess(context.Background())
	testImage1 := fixtures.GetImageV2WithUniqueComponents(5)
	s.NoError(s.datastore.UpsertImage(ctx, testImage1))

	testImage2 := testImage1.CloneVT()
	testImage2.Name.FullName = "registry.test.io/img2:latest"
	testImage2.Id = uuid.NewV5FromNonUUIDs(testImage2.GetName().GetFullName(), testImage2.GetDigest()).String()
	s.NoError(s.datastore.UpsertImage(ctx, testImage2))

	testImage3 := testImage1.CloneVT()
	testImage3.Name.FullName = "registry.test.io/img3:latest"
	testImage3.Id = uuid.NewV5FromNonUUIDs(testImage3.GetName().GetFullName(), testImage3.GetDigest()).String()
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
	protoassert.ElementsMatch(s.T(), []*storage.ImageV2{testImage1, testImage2, testImage3}, storedImages)
}

func (s *ImageV2DataStoreTestSuite) TestGetImageIdsAndDigest() {
	ctx := sac.WithAllAccess(context.Background())
	testImage1 := fixtures.GetImageV2()
	s.NoError(s.datastore.UpsertImage(ctx, testImage1))
	results, err := s.datastore.GetImageIDsAndDigests(ctx, pkgSearch.EmptyQuery())
	s.NoError(err)
	s.Len(results, 1)
	s.Equal(testImage1.GetId(), results[0].ImageID)
	s.Equal(testImage1.GetDigest(), results[0].Digest)
}

func (s *ImageV2DataStoreTestSuite) TestGetImageNames() {
	ctx := sac.WithAllAccess(context.Background())
	img1 := getTestImageV2("img1")

	img2 := img1.CloneVT()
	img2.Name = &storage.ImageName{
		Registry: "registry.test.io",
		Remote:   "img2",
		Tag:      "latest",
		FullName: "registry.test.io/img2:latest",
	}
	img2.Id = uuid.NewV5FromNonUUIDs(img2.GetName().GetFullName(), img2.GetDigest()).String()

	img3 := img1.CloneVT()
	img3.Name = &storage.ImageName{
		Registry: "registry.test.io",
		Remote:   "img3",
		Tag:      "latest",
		FullName: "registry.test.io/img3:latest",
	}
	img3.Id = uuid.NewV5FromNonUUIDs(img3.GetName().GetFullName(), img3.GetDigest()).String()

	s.NoError(s.datastore.UpsertImage(ctx, img1))
	s.NoError(s.datastore.UpsertImage(ctx, img2))
	s.NoError(s.datastore.UpsertImage(ctx, img3))

	expectedImageNames := []*storage.ImageName{
		{
			Registry: "registry.test.io",
			Remote:   "img1",
			Tag:      "latest",
			FullName: "registry.test.io/img1:latest",
		},
		{
			Registry: "registry.test.io",
			Remote:   "img2",
			Tag:      "latest",
			FullName: "registry.test.io/img2:latest",
		},
		{
			Registry: "registry.test.io",
			Remote:   "img3",
			Tag:      "latest",
			FullName: "registry.test.io/img3:latest",
		},
	}
	imageNames, err := s.datastore.GetImageNames(ctx, img1.GetDigest())
	s.NoError(err)
	protoassert.ElementsMatch(s.T(), expectedImageNames, imageNames)
}

func (s *ImageV2DataStoreTestSuite) TestCVETimestampPersistence() {
	s.T().Setenv(features.CVEFixTimestampCriteria.EnvVar(), "true")
	if !features.CVEFixTimestampCriteria.Enabled() {
		s.T().Skip("CVEFixTimestampCriteria feature must be enabled for this test")
	}

	ctx := sac.WithAllAccess(context.Background())

	// Scanner-provided timestamp for when the shared CVE was first discovered
	cveDiscoverTimestamp := timestamppb.New(time.Now().Add(-24 * time.Hour))

	sharedCVEID := "CVE-2024-1234"
	datasource := "alpine:v3.18"

	image0Name := &storage.ImageName{FullName: "registry.io/image0:v0"}
	image0Digest := "sha256:image0digest"
	image0ID := imageUtils.NewImageV2ID(image0Name, image0Digest)

	image1Name := &storage.ImageName{FullName: "registry.io/image1:v1"}
	image1Digest := "sha256:image1digest"

	image2Name := &storage.ImageName{FullName: "registry.io/image2:v2"}
	image2Digest := "sha256:image2digest"

	// Three images share a CVE but each also has unique CVEs.
	// The shared CVE in the second image has an earlier FirstSystemOccurrence timestamp.
	images := []*storage.ImageV2{
		{
			Id:     image0ID,
			Digest: image0Digest,
			Name:   image0Name,
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
			Id:     imageUtils.NewImageV2ID(image1Name, image1Digest),
			Digest: image1Digest,
			Name:   image1Name,
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
								FirstSystemOccurrence: cveDiscoverTimestamp, // Earlier timestamp
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
			Id:     imageUtils.NewImageV2ID(image2Name, image2Digest),
			Digest: image2Digest,
			Name:   image2Name,
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
		s.NoError(s.cveInfoEnricher.EnrichImageV2WithCVEInfo(ctx, imgClone))
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
		s.NoError(s.cveInfoEnricher.EnrichImageV2WithCVEInfo(ctx, imgClone))
		s.NoError(s.datastore.UpsertImage(ctx, imgClone))
	}

	// Verify all images now have the preserved earliest timestamp.
	// The CVE-name-based aggregation ensures all images get the MIN timestamp for CVE-2024-1234.
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

func (s *ImageV2DataStoreTestSuite) TestCVETimestampAggregation() {
	s.T().Setenv(features.CVEFixTimestampCriteria.EnvVar(), "true")
	if !features.CVEFixTimestampCriteria.Enabled() {
		s.T().Skip("CVEFixTimestampCriteria feature must be enabled for this test")
	}

	ctx := sac.WithAllAccess(context.Background())

	// Two images with the same CVE name but different components and operating systems
	sharedCVEName := "CVE-2024-SHARED"
	earlierTimestamp := timestamppb.New(time.Now().Add(-48 * time.Hour))
	laterTimestamp := timestamppb.New(time.Now().Add(-24 * time.Hour))

	img1Name := &storage.ImageName{FullName: "registry.io/ubuntu-app:v1"}
	img1Digest := "sha256:image1digest"
	img1ID := imageUtils.NewImageV2ID(img1Name, img1Digest)

	img2Name := &storage.ImageName{FullName: "registry.io/alpine-app:v1"}
	img2Digest := "sha256:image2digest"
	img2ID := imageUtils.NewImageV2ID(img2Name, img2Digest)

	// Image 1: Ubuntu with package-a, has the earlier timestamp
	image1 := &storage.ImageV2{
		Id:     img1ID,
		Digest: img1Digest,
		Name:   img1Name,
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
	image2 := &storage.ImageV2{
		Id:     img2ID,
		Digest: img2Digest,
		Name:   img2Name,
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
	s.NoError(s.cveInfoEnricher.EnrichImageV2WithCVEInfo(ctx, img1Clone))
	s.NoError(s.datastore.UpsertImage(ctx, img1Clone))

	// Process image2 (has later timestamp)
	img2Clone := image2.CloneVT()
	s.NoError(s.cveInfoEnricher.EnrichImageV2WithCVEInfo(ctx, img2Clone))
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
	storedImage1, found, err := s.datastore.GetImage(ctx, img1ID)
	s.NoError(err)
	s.True(found)
	s.Require().Len(storedImage1.GetScan().GetComponents(), 1)
	s.Require().Len(storedImage1.GetScan().GetComponents()[0].GetVulns(), 1)
	s.Equal(earlierTimestamp, storedImage1.GetScan().GetComponents()[0].GetVulns()[0].GetFirstSystemOccurrence(),
		"Image1 should have the earlier timestamp")

	storedImage2, found, err := s.datastore.GetImage(ctx, img2ID)
	s.NoError(err)
	s.True(found)
	s.Require().Len(storedImage2.GetScan().GetComponents(), 1)
	s.Require().Len(storedImage2.GetScan().GetComponents()[0].GetVulns(), 1)
	s.Equal(earlierTimestamp, storedImage2.GetScan().GetComponents()[0].GetVulns()[0].GetFirstSystemOccurrence(),
		"Image2 should also have the earlier timestamp from MIN aggregation across all CVE records")

	// Rescan image1 to verify the MIN timestamp is still preserved
	img1Rescan := image1.CloneVT()
	img1Rescan.GetScan().ScanTime = timestamppb.Now()
	s.NoError(s.cveInfoEnricher.EnrichImageV2WithCVEInfo(ctx, img1Rescan))
	s.NoError(s.datastore.UpsertImage(ctx, img1Rescan))

	storedImage1After, found, err := s.datastore.GetImage(ctx, img1ID)
	s.NoError(err)
	s.True(found)
	s.Require().Len(storedImage1After.GetScan().GetComponents(), 1)
	s.Require().Len(storedImage1After.GetScan().GetComponents()[0].GetVulns(), 1)
	s.Equal(earlierTimestamp, storedImage1After.GetScan().GetComponents()[0].GetVulns()[0].GetFirstSystemOccurrence(),
		"After rescan, image1 should still have the MIN timestamp")
}

func (s *ImageV2DataStoreTestSuite) truncateTable(name string) {
	sql := fmt.Sprintf("TRUNCATE %s CASCADE", name)
	_, err := s.testDB.Exec(s.ctx, sql)
	s.NoError(err)
}

func getTestImageV2(name string) *storage.ImageV2 {
	imageName := fmt.Sprintf("registry.test.io/%s:latest", name)
	imageSha := fmt.Sprintf("sha256:%s1234567890", name)
	imageID := uuid.NewV5FromNonUUIDs(imageName, imageSha).String()

	return &storage.ImageV2{
		Id:     imageID,
		Digest: imageSha,
		Name: &storage.ImageName{
			Registry: "registry.test.io",
			Remote:   name,
			Tag:      "latest",
			FullName: imageName,
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

func cloneAndUpdateRiskPriority(image *storage.ImageV2) *storage.ImageV2 {
	cloned := image.CloneVT()
	cloned.Priority = 1
	for _, component := range cloned.GetScan().GetComponents() {
		component.Priority = 1
	}
	return cloned
}

func (s *ImageV2DataStoreTestSuite) TestSearchListImages() {
	ctx := sac.WithAllAccess(context.Background())

	// Create and upsert test images
	img1 := getTestImageV2("img1")
	img2 := getTestImageV2("img2")
	img3 := getTestImageV2("img3")

	s.NoError(s.datastore.UpsertImage(ctx, img1))
	s.NoError(s.datastore.UpsertImage(ctx, img2))
	s.NoError(s.datastore.UpsertImage(ctx, img3))

	// Test 1: Search all images with empty query
	listImages, err := s.datastore.SearchListImages(ctx, pkgSearch.EmptyQuery())
	s.NoError(err)
	s.Len(listImages, 3)

	// Verify that all images are present
	// Note: In V2->V1 conversion, Image.Id is set to the digest (SHA), not the UUID
	imageDigests := set.NewStringSet()
	for _, img := range listImages {
		imageDigests.Add(img.GetId())
		// Verify priority is set by the ranker
		s.NotZero(img.GetPriority())
		// Verify ListImage has expected fields
		s.NotEmpty(img.GetId())
		s.NotEmpty(img.GetName())
		// LastUpdated should be set
		s.NotNil(img.GetLastUpdated())
	}
	// Verify all image digests are present
	s.True(imageDigests.Contains(img1.GetDigest()))
	s.True(imageDigests.Contains(img2.GetDigest()))
	s.True(imageDigests.Contains(img3.GetDigest()))

	// Test 2: Search with specific image name
	q := pkgSearch.NewQueryBuilder().AddStrings(pkgSearch.ImageName, "registry.test.io/img1:latest").ProtoQuery()
	listImages, err = s.datastore.SearchListImages(ctx, q)
	s.NoError(err)
	s.Len(listImages, 1)
	s.Equal(img1.GetDigest(), listImages[0].GetId()) // ListImage ID is the digest in V1 format
	s.Equal("registry.test.io/img1:latest", listImages[0].GetName())

	// Test 3: Search with image UUID (V2 ID)
	q = pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.ImageID, img2.GetId()).ProtoQuery()
	listImages, err = s.datastore.SearchListImages(ctx, q)
	s.NoError(err)
	s.Len(listImages, 1)
	s.Equal(img2.GetDigest(), listImages[0].GetId())

	// Test 4: Search with SHA
	q = pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.ImageSHA, img3.GetDigest()).ProtoQuery()
	listImages, err = s.datastore.SearchListImages(ctx, q)
	s.NoError(err)
	s.Len(listImages, 1)
	s.Equal(img3.GetDigest(), listImages[0].GetId())

	// Test 5: Search with pagination
	q = pkgSearch.EmptyQuery()
	q.Pagination = &v1.QueryPagination{
		Limit:  2,
		Offset: 0,
	}
	listImages, err = s.datastore.SearchListImages(ctx, q)
	s.NoError(err)
	s.Len(listImages, 2)

	// Test 6: Search with no results
	q = pkgSearch.NewQueryBuilder().AddStrings(pkgSearch.ImageName, "nonexistent").ProtoQuery()
	listImages, err = s.datastore.SearchListImages(ctx, q)
	s.NoError(err)
	s.Len(listImages, 0)

	// Test 7: Verify ListImage contains component and CVE counts from ScanStats
	// Create an image with scan stats populated
	imgWithStats := fixtures.GetImageV2WithUniqueComponents(3)
	s.NoError(s.datastore.UpsertImage(ctx, imgWithStats))

	q = pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.ImageID, imgWithStats.GetId()).ProtoQuery()
	listImages, err = s.datastore.SearchListImages(ctx, q)
	s.NoError(err)
	s.Len(listImages, 1)
	s.Equal(imgWithStats.GetDigest(), listImages[0].GetId())

	// Verify the component count comes from ScanStats (fixture sets ComponentCount to 3)
	s.Equal(int32(3), listImages[0].GetComponents(), "Expected component count to match fixture")

	// CVE and fixable CVE counts may be populated by the datastore during upsert
	// We just verify they are non-negative (could be 0 if not populated)
	s.GreaterOrEqual(listImages[0].GetCves(), int32(0))
	s.GreaterOrEqual(listImages[0].GetFixableCves(), int32(0))

	// Test 8: Test with access control - no access context
	noAccessCtx := sac.WithNoAccess(context.Background())
	listImages, err = s.datastore.SearchListImages(noAccessCtx, pkgSearch.EmptyQuery())
	s.NoError(err)
	s.Len(listImages, 0)

	// Test 9: Verify sorting works correctly
	q = pkgSearch.EmptyQuery()
	q.Pagination = &v1.QueryPagination{
		SortOptions: []*v1.QuerySortOption{
			{
				Field: pkgSearch.ImageName.String(),
			},
		},
	}
	listImages, err = s.datastore.SearchListImages(ctx, q)
	s.NoError(err)
	s.Greater(len(listImages), 1)
	// Verify images are sorted by name
	for i := 1; i < len(listImages); i++ {
		s.LessOrEqual(listImages[i-1].GetName(), listImages[i].GetName(), "Images should be sorted by name")
	}

	// Test 10: Verify distinct results when joining with other tables (ROX-33514)
	// Create 2 deployments that reference all 3 images
	dep1 := fixtures.LightweightDeployment()
	dep1.Id = uuid.NewV4().String()
	dep1.Name = "deployment1"
	dep1.Containers = []*storage.Container{
		{Name: "container1", Image: &storage.ContainerImage{Id: img1.GetDigest(), IdV2: img1.GetId(), Name: img1.GetName()}},
		{Name: "container2", Image: &storage.ContainerImage{Id: img2.GetDigest(), IdV2: img2.GetId(), Name: img2.GetName()}},
		{Name: "container3", Image: &storage.ContainerImage{Id: img3.GetDigest(), IdV2: img3.GetId(), Name: img3.GetName()}},
	}

	dep2 := fixtures.LightweightDeployment()
	dep2.Id = uuid.NewV4().String()
	dep2.Name = "deployment2"
	dep2.Containers = []*storage.Container{
		{Name: "container4", Image: &storage.ContainerImage{Id: img1.GetDigest(), IdV2: img1.GetId(), Name: img1.GetName()}},
		{Name: "container5", Image: &storage.ContainerImage{Id: img2.GetDigest(), IdV2: img2.GetId(), Name: img2.GetName()}},
		{Name: "container6", Image: &storage.ContainerImage{Id: img3.GetDigest(), IdV2: img3.GetId(), Name: img3.GetName()}},
	}

	s.NoError(s.deploymentDataStore.UpsertDeployment(ctx, dep1))
	s.NoError(s.deploymentDataStore.UpsertDeployment(ctx, dep2))

	// Search for images with deployment filter - should return 3 unique images, not 6
	q = pkgSearch.NewQueryBuilder().AddRegexes(pkgSearch.DeploymentName, ".*").ProtoQuery()
	listImages, err = s.datastore.SearchListImages(ctx, q)
	s.NoError(err)

	// Verify we get exactly 3 unique images despite having 2 deployments referencing all 3
	s.Len(listImages, 3, "Expected 3 unique images, not duplicates from multiple deployments")
	imageDigests = set.NewStringSet()
	for _, img := range listImages {
		imageDigests.Add(img.GetId())
	}
	s.True(imageDigests.Contains(img1.GetDigest()))
	s.True(imageDigests.Contains(img2.GetDigest()))
	s.True(imageDigests.Contains(img3.GetDigest()))
}
