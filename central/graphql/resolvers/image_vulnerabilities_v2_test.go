//go:build sql_integration

package resolvers

import (
	"context"
	"testing"

	"github.com/graph-gophers/graphql-go"
	"github.com/stackrox/rox/central/graphql/resolvers/loaders"
	"github.com/stackrox/rox/central/views/imagecveflat"
	imagesView "github.com/stackrox/rox/central/views/images"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	"github.com/stackrox/rox/pkg/pointers"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search/scoped"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

const (
	cve111 = "cve1comp1image1"
	cve121 = "cve1comp2image1"
	cve231 = "cve2comp3image1"
	cve331 = "cve3comp3image1"
	cve112 = "cve1comp1image2"
	cve232 = "cve2comp3image2"
	cve332 = "cve3comp3image2"
	cve442 = "cve4comp4image2"
	cve542 = "cve5comp4image2"
)

var (
	distinctCVEs = []string{"cve-2018-1", "cve-2019-1", "cve-2019-2", "cve-2017-1", "cve-2017-2"}
)

func TestGraphQLImageVulnerabilityV2Endpoints(t *testing.T) {
	suite.Run(t, new(GraphQLImageVulnerabilityV2TestSuite))
}

/*
Remaining TODO tasks:
- As sub resolver via cluster
- As sub resolver via namespace
- As sub resolver via deployment
- TopImageVulnerability
- sub resolver values
	- EnvImpact
	- LastScanned
	- Deployments
	- DeploymentCount
	- DiscoveredAtImage
*/

type GraphQLImageVulnerabilityV2TestSuite struct {
	suite.Suite

	ctx      context.Context
	testDB   *pgtest.TestPostgres
	resolver *Resolver

	cveIDMap       map[string]string
	componentIDMap map[string]string
}

func (s *GraphQLImageVulnerabilityV2TestSuite) SetupSuite() {
	if !features.FlattenCVEData.Enabled() {
		s.T().Skip()
	}

	s.ctx = loaders.WithLoaderContext(sac.WithAllAccess(context.Background()))
	mockCtrl := gomock.NewController(s.T())
	s.testDB = pgtest.ForT(s.T())
	vulnReqDatastore, err := TestVulnReqDatastore(s.T(), s.testDB)
	s.Require().NoError(err)
	resolver, _ := SetupTestResolver(s.T(),
		imagesView.NewImageView(s.testDB.DB),
		CreateTestImageV2Datastore(s.T(), s.testDB, mockCtrl),
		CreateTestImageComponentV2Datastore(s.T(), s.testDB, mockCtrl),
		CreateTestImageCVEV2Datastore(s.T(), s.testDB),
		vulnReqDatastore,
		imagecveflat.NewCVEFlatView(s.testDB.DB),
	)
	s.resolver = resolver

	// Add Test Data to DataStores
	testImages := testImages()
	for _, image := range testImages {
		err := s.resolver.ImageDataStore.UpsertImage(s.ctx, image)
		s.NoError(err)
	}

	// Add test vulnerability exceptions
	for _, vulnReq := range []*storage.VulnerabilityRequest{
		fixtures.GetImageScopeDeferralRequest("reg1", "img1", "tag1", "cve-2018-1"),
		func() *storage.VulnerabilityRequest {
			req := fixtures.GetImageScopeDeferralRequest("reg1", "img1", ".*", "cve-2018-1")
			req.Status = storage.RequestStatus_APPROVED
			return req
		}(),
		fixtures.GetImageScopeDeferralRequest("reg2", "img2", ".*", "cve-2018-1"),
		func() *storage.VulnerabilityRequest {
			req := fixtures.GetImageScopeDeferralRequest("reg2", "img2", ".*", "cve-2017-2")
			req.Status = storage.RequestStatus_APPROVED_PENDING_UPDATE
			return req
		}(),
		fixtures.GetImageScopeDeferralRequest("reg2", "img2", "", "cve-2017-1"),
		fixtures.GetGlobalDeferralRequestV2("cve-2017-2"),
		fixtures.GetGlobalFPRequestV2("cve-2019-1"),
		func() *storage.VulnerabilityRequest {
			req := fixtures.GetGlobalFPRequestV2("cve-2019-2")
			req.Status = storage.RequestStatus_APPROVED
			return req
		}(),
	} {
		s.NoError(s.resolver.vulnReqStore.AddRequest(s.ctx, vulnReq))
	}

	s.componentIDMap = s.getComponentIDMap()
	s.cveIDMap = s.getIDMap()
}

func (s *GraphQLImageVulnerabilityV2TestSuite) TestUnauthorizedImageVulnerabilityEndpoint() {
	_, err := s.resolver.ImageVulnerability(s.ctx, IDQuery{})
	s.Error(err, "Unauthorized request got through")
}

func (s *GraphQLImageVulnerabilityV2TestSuite) TestUnauthorizedImageVulnerabilitiesEndpoint() {
	_, err := s.resolver.ImageVulnerabilities(s.ctx, PaginatedQuery{})
	s.Error(err, "Unauthorized request got through")
}

func (s *GraphQLImageVulnerabilityV2TestSuite) TestUnauthorizedImageVulnerabilityCountEndpoint() {
	_, err := s.resolver.ImageVulnerabilityCount(s.ctx, RawQuery{})
	s.Error(err, "Unauthorized request got through")
}

func (s *GraphQLImageVulnerabilityV2TestSuite) TestUnauthorizedImageVulnerabilityCounterEndpoint() {
	_, err := s.resolver.ImageVulnerabilityCounter(s.ctx, RawQuery{})
	s.Error(err, "Unauthorized request got through")
}

func (s *GraphQLImageVulnerabilityV2TestSuite) TestUnauthorizedTopImageVulnerabilityEndpoint() {
	_, err := s.resolver.TopImageVulnerability(s.ctx, RawQuery{})
	s.Error(err, "Unauthorized request got through")
}

func (s *GraphQLImageVulnerabilityV2TestSuite) TestImageVulnerabilities() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	expected := int32(len(distinctCVEs))

	vulns, err := s.resolver.ImageVulnerabilities(ctx, PaginatedQuery{})
	s.NoError(err)
	s.Equal(expected, int32(len(vulns)))
	cveList := getCVEList(ctx, vulns)
	s.ElementsMatch(distinctCVEs, cveList)

	count, err := s.resolver.ImageVulnerabilityCount(ctx, RawQuery{})
	s.NoError(err)
	s.Equal(expected, count)

	counter, err := s.resolver.ImageVulnerabilityCounter(ctx, RawQuery{})
	s.NoError(err)
	checkVulnerabilityCounter(s.T(), counter, expected, 1, 1, 2, 1, 1)
}

func (s *GraphQLImageVulnerabilityV2TestSuite) TestImageVulnerabilitiesFixable() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	expected := int32(1)

	query, err := getFixableRawQuery(true)
	s.NoError(err)

	vulns, err := s.resolver.ImageVulnerabilities(ctx, PaginatedQuery{Query: &query})
	s.NoError(err)
	s.Equal(expected, int32(len(vulns)))
	for _, vuln := range vulns {
		fixable, err := vuln.IsFixable(ctx, RawQuery{})
		s.NoError(err)
		s.Equal(true, fixable)
		// test fixed by is empty string because it requires component scoping
		fixedBy, err := vuln.FixedByVersion(ctx)
		s.NoError(err)
		s.Equal("", fixedBy)
	}
	cveVulns := set.NewStringSet()
	for _, vuln := range vulns {
		cveVulns.Add(vuln.CVE(ctx))
	}
	s.ElementsMatch([]string{"cve-2018-1"}, cveVulns.AsSlice())

	count, err := s.resolver.ImageVulnerabilityCount(ctx, RawQuery{Query: &query})
	s.NoError(err)
	s.Equal(expected, count)
}

func (s *GraphQLImageVulnerabilityV2TestSuite) TestImageVulnerabilitiesNonFixable() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	expected := int32(4)

	query, err := getFixableRawQuery(false)
	s.NoError(err)

	vulns, err := s.resolver.ImageVulnerabilities(ctx, PaginatedQuery{Query: &query})
	s.NoError(err)
	s.Equal(expected, int32(len(vulns)))
	for _, vuln := range vulns {
		fixable, err := vuln.IsFixable(ctx, RawQuery{})
		s.NoError(err)
		s.Equal(false, fixable)
	}
	expectedCVEs := []string{"cve-2019-1", "cve-2019-2", "cve-2017-1", "cve-2017-2"}
	cveList := getCVEList(ctx, vulns)
	s.ElementsMatch(expectedCVEs, cveList)

	count, err := s.resolver.ImageVulnerabilityCount(ctx, RawQuery{Query: &query})
	s.NoError(err)
	s.Equal(expected, count)
}

func (s *GraphQLImageVulnerabilityV2TestSuite) TestImageDiscoveredAt() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	vulns, err := s.resolver.ImageVulnerabilities(ctx, PaginatedQuery{})
	s.NoError(err)

	for _, vuln := range vulns {
		discovered, err := vuln.DiscoveredAtImage(ctx, RawQuery{})
		s.NoError(err)
		s.NotNil(discovered)
	}
}

func (s *GraphQLImageVulnerabilityV2TestSuite) TestImageVulnerabilitiesFixedByVersion() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	scopedCtx := scoped.Context(ctx, scoped.Scope{
		Level: v1.SearchCategory_IMAGE_COMPONENTS_V2,
		IDs:   []string{s.componentIDMap[comp11]},
	})
	vuln := s.getImageVulnerabilityResolver(scopedCtx, s.cveIDMap[cve111])
	fixedBy, err := vuln.FixedByVersion(ctx)
	s.NoError(err)
	s.Equal("1.1", fixedBy)

	scopedCtx = scoped.Context(ctx, scoped.Scope{
		Level: v1.SearchCategory_IMAGE_COMPONENTS_V2,
		IDs:   []string{s.componentIDMap[comp21]},
	})
	vuln = s.getImageVulnerabilityResolver(scopedCtx, s.cveIDMap[cve121])

	fixedBy, err = vuln.FixedByVersion(ctx)
	s.NoError(err)
	s.Equal("1.5", fixedBy)

	scopedCtx = scoped.Context(ctx, scoped.Scope{
		Level: v1.SearchCategory_IMAGE_COMPONENTS_V2,
		IDs:   []string{s.componentIDMap[comp42]},
	})
	vuln = s.getImageVulnerabilityResolver(scopedCtx, s.cveIDMap[cve442])

	fixedBy, err = vuln.FixedByVersion(ctx)
	s.NoError(err)
	s.Equal("", fixedBy)
}

func (s *GraphQLImageVulnerabilityV2TestSuite) TestImageVulnerabilitiesScoped() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	image := s.getImageResolver(ctx, "sha1")
	expected := int32(3)
	expectedCVEs := []string{"cve-2019-1", "cve-2019-2", "cve-2018-1"}

	vulns, err := image.ImageVulnerabilities(ctx, PaginatedQuery{})
	s.NoError(err)
	s.Equal(expected, int32(len(vulns)))

	cveList := getCVEList(ctx, vulns)
	s.ElementsMatch(expectedCVEs, cveList)

	count, err := image.ImageVulnerabilityCount(ctx, RawQuery{})
	s.NoError(err)
	s.Equal(expected, count)

	counter, err := image.ImageVulnerabilityCounter(ctx, RawQuery{})
	s.NoError(err)
	checkVulnerabilityCounter(s.T(), counter, expected, 1, 1, 0, 1, 1)

	image = s.getImageResolver(ctx, "sha2")
	expected = int32(5)
	expectedCVEs = []string{"cve-2019-1", "cve-2019-2", "cve-2018-1", "cve-2017-1", "cve-2017-2"}

	vulns, err = image.ImageVulnerabilities(ctx, PaginatedQuery{})
	s.NoError(err)
	s.Equal(expected, int32(len(vulns)))
	cveList = getCVEList(ctx, vulns)
	s.ElementsMatch(expectedCVEs, cveList)

	count, err = image.ImageVulnerabilityCount(ctx, RawQuery{})
	s.NoError(err)
	s.Equal(expected, count)

	counter, err = image.ImageVulnerabilityCounter(ctx, RawQuery{})
	s.NoError(err)
	checkVulnerabilityCounter(s.T(), counter, expected, 1, 1, 2, 1, 1)
}

func (s *GraphQLImageVulnerabilityV2TestSuite) TestImageVulnerabilityMiss() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	vulnID := graphql.ID("invalid")

	_, err := s.resolver.ImageVulnerability(ctx, IDQuery{ID: &vulnID})
	s.Error(err)
}

func (s *GraphQLImageVulnerabilityV2TestSuite) TestImageVulnerabilityHit() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	vulnID := graphql.ID(s.cveIDMap[cve111])

	vuln, err := s.resolver.ImageVulnerability(ctx, IDQuery{ID: &vulnID})
	s.NoError(err)
	s.Equal(vulnID, vuln.Id(ctx))
}

func (s *GraphQLImageVulnerabilityV2TestSuite) TestTopImageVulnerabilityUnscoped() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	_, err := s.resolver.TopImageVulnerability(ctx, RawQuery{})
	s.Error(err)
}

func (s *GraphQLImageVulnerabilityV2TestSuite) TestTopImageVulnerability() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	image := s.getImageResolver(ctx, "sha1")

	expected := graphql.ID(s.cveIDMap[cve231])
	topVuln, err := image.TopImageVulnerability(ctx, RawQuery{})
	s.NoError(err)
	s.Equal(expected, topVuln.Id(ctx))
}

func (s *GraphQLImageVulnerabilityV2TestSuite) TestImageVulnerabilityImages() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	vuln := s.getImageVulnerabilityResolver(ctx, s.cveIDMap[cve111])

	images, err := vuln.Images(ctx, PaginatedQuery{})
	s.NoError(err)
	s.Equal(1, len(images))
	idList := getIDList(ctx, images)
	s.ElementsMatch([]string{"sha1"}, idList)

	count, err := vuln.ImageCount(ctx, RawQuery{})
	s.NoError(err)
	s.Equal(int32(len(images)), count)

	vuln = s.getImageVulnerabilityResolver(ctx, s.cveIDMap[cve442])

	images, err = vuln.Images(ctx, PaginatedQuery{})
	s.NoError(err)
	s.Equal(1, len(images))
	idList = getIDList(ctx, images)
	s.ElementsMatch([]string{"sha2"}, idList)

	count, err = vuln.ImageCount(ctx, RawQuery{})
	s.NoError(err)
	s.Equal(int32(len(images)), count)
}

func (s *GraphQLImageVulnerabilityV2TestSuite) TestImageVulnerabilityImageComponents() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	vuln := s.getImageVulnerabilityResolver(ctx, s.cveIDMap[cve111])

	comps, err := vuln.ImageComponents(ctx, PaginatedQuery{})
	s.NoError(err)
	s.Equal(1, len(comps))
	idList := getIDList(ctx, comps)
	s.ElementsMatch([]string{s.componentIDMap[comp11]}, idList)

	count, err := vuln.ImageComponentCount(ctx, RawQuery{})
	s.NoError(err)
	s.Equal(int32(len(comps)), count)

	vuln = s.getImageVulnerabilityResolver(ctx, s.cveIDMap[cve442])

	comps, err = vuln.ImageComponents(ctx, PaginatedQuery{})
	s.NoError(err)
	s.Equal(1, len(comps))
	idList = getIDList(ctx, comps)
	s.ElementsMatch([]string{s.componentIDMap[comp42]}, idList)

	count, err = vuln.ImageComponentCount(ctx, RawQuery{})
	s.NoError(err)
	s.Equal(int32(len(comps)), count)
}

func (s *GraphQLImageVulnerabilityV2TestSuite) TestImageVulnerabilityExceptionCountAll() {
	// TODO(ROX-27780): Defer until vuln requests updates are made
	s.T().Skip()

	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())
	args := struct {
		RequestStatus *[]*string
	}{}

	// Deferral:
	// - sha1 all tags; sha1 one tag
	// - sha2 one tag
	vuln := s.getImageVulnerabilityResolver(ctx, "cve-2018-1#")
	count, err := vuln.ExceptionCount(ctx, args)
	s.NoError(err)
	s.Equal(int32(3), count)

	// Deferral:
	// - global
	// - sha2 all tags
	vuln = s.getImageVulnerabilityResolver(ctx, "cve-2017-2#")
	count, err = vuln.ExceptionCount(ctx, args)
	s.NoError(err)
	s.Equal(int32(2), count)

	// False-positive:
	// - global
	vuln = s.getImageVulnerabilityResolver(ctx, "cve-2019-1#")
	count, err = vuln.ExceptionCount(ctx, args)
	s.NoError(err)
	s.Equal(int32(1), count)
}

func (s *GraphQLImageVulnerabilityV2TestSuite) TestImageVulnerabilityExceptionCountPending() {
	// TODO(ROX-27780): Defer until vuln requests updates are made
	s.T().Skip()

	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())
	status := []*string{pointers.String(storage.RequestStatus_PENDING.String())}
	args := struct {
		RequestStatus *[]*string
	}{
		RequestStatus: &status,
	}

	// Deferral:
	// - sha1 one tag
	// - sha2 one tag
	vuln := s.getImageVulnerabilityResolver(ctx, "cve-2018-1#")
	count, err := vuln.ExceptionCount(ctx, args)
	s.NoError(err)
	s.Equal(int32(2), count)

	// Deferral:
	// - global
	vuln = s.getImageVulnerabilityResolver(ctx, "cve-2017-2#")
	count, err = vuln.ExceptionCount(ctx, args)
	s.NoError(err)
	s.Equal(int32(1), count)

	// False-positive:
	// - global
	vuln = s.getImageVulnerabilityResolver(ctx, "cve-2019-1#")
	count, err = vuln.ExceptionCount(ctx, args)
	s.NoError(err)
	s.Equal(int32(1), count)
}

func (s *GraphQLImageVulnerabilityV2TestSuite) TestImageVulnerabilityExceptionCountApproved() {
	// TODO(ROX-27780): Defer until vuln requests updates are made
	s.T().Skip()

	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())
	status := []*string{pointers.String(storage.RequestStatus_APPROVED.String())}
	args := struct {
		RequestStatus *[]*string
	}{
		RequestStatus: &status,
	}

	// Deferral:
	// - sha1 all tags
	vuln := s.getImageVulnerabilityResolver(ctx, "cve-2018-1#")
	count, err := vuln.ExceptionCount(ctx, args)
	s.NoError(err)
	s.Equal(int32(1), count)

	vuln = s.getImageVulnerabilityResolver(ctx, "cve-2017-2#")
	count, err = vuln.ExceptionCount(ctx, args)
	s.NoError(err)
	s.Equal(int32(0), count)

	vuln = s.getImageVulnerabilityResolver(ctx, "cve-2019-1#")
	count, err = vuln.ExceptionCount(ctx, args)
	s.NoError(err)
	s.Equal(int32(0), count)
}

func (s *GraphQLImageVulnerabilityV2TestSuite) TestImageVulnerabilityExceptionCountPendingUpdate() {
	// TODO(ROX-27780): Defer until vuln requests updates are made
	s.T().Skip()

	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())
	status := []*string{pointers.String(storage.RequestStatus_APPROVED_PENDING_UPDATE.String())}
	args := struct {
		RequestStatus *[]*string
	}{
		RequestStatus: &status,
	}

	vuln := s.getImageVulnerabilityResolver(ctx, "cve-2018-1#")
	count, err := vuln.ExceptionCount(ctx, args)
	s.NoError(err)
	s.Equal(int32(0), count)

	// Deferral:
	// - sha2 all tags
	vuln = s.getImageVulnerabilityResolver(ctx, "cve-2017-2#")
	count, err = vuln.ExceptionCount(ctx, args)
	s.NoError(err)
	s.Equal(int32(1), count)

	vuln = s.getImageVulnerabilityResolver(ctx, "cve-2019-1#")
	count, err = vuln.ExceptionCount(ctx, args)
	s.NoError(err)
	s.Equal(int32(0), count)
}

func (s *GraphQLImageVulnerabilityV2TestSuite) TestImageVulnerabilityExceptionCountAllWithImageScope() {
	// TODO(ROX-27780): Defer until vuln requests updates are made
	s.T().Skip()

	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())
	ctx = scoped.Context(ctx, scoped.Scope{
		IDs:   []string{"sha1"},
		Level: v1.SearchCategory_IMAGES,
	})
	args := struct {
		RequestStatus *[]*string
	}{}

	// Deferral:
	// - sha1 all tags; sha1 one tag
	vuln := s.getImageVulnerabilityResolver(ctx, "cve-2018-1#")
	count, err := vuln.ExceptionCount(ctx, args)
	s.NoError(err)
	s.Equal(int32(2), count)

	// Deferral:
	// - global (covers the sha1 image)
	vuln = s.getImageVulnerabilityResolver(ctx, "cve-2017-2#")
	count, err = vuln.ExceptionCount(ctx, args)
	s.NoError(err)
	s.Equal(int32(1), count)

	// False-positive:
	// - global (covers the sha1 image)
	vuln = s.getImageVulnerabilityResolver(ctx, "cve-2019-1#")
	count, err = vuln.ExceptionCount(ctx, args)
	s.NoError(err)
	s.Equal(int32(1), count)
}

func (s *GraphQLImageVulnerabilityV2TestSuite) TestImageVulnerabilityExceptionCountPendingWithImageScope() {
	// TODO(ROX-27780): Defer until vuln requests updates are made
	s.T().Skip()

	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())
	status := []*string{pointers.String(storage.RequestStatus_PENDING.String())}
	ctx = scoped.Context(ctx, scoped.Scope{
		IDs:   []string{"sha1"},
		Level: v1.SearchCategory_IMAGES,
	})
	args := struct {
		RequestStatus *[]*string
	}{
		RequestStatus: &status,
	}

	// Deferral:
	// - sha1 one tag
	vuln := s.getImageVulnerabilityResolver(ctx, "cve-2018-1#")
	count, err := vuln.ExceptionCount(ctx, args)
	s.NoError(err)
	s.Equal(int32(1), count)

	// Deferral:
	// - global (covers the sha1 image)
	vuln = s.getImageVulnerabilityResolver(ctx, "cve-2017-2#")
	count, err = vuln.ExceptionCount(ctx, args)
	s.NoError(err)
	s.Equal(int32(1), count)

	// False-positive:
	// - global (covers the sha1 image)
	vuln = s.getImageVulnerabilityResolver(ctx, "cve-2019-1#")
	count, err = vuln.ExceptionCount(ctx, args)
	s.NoError(err)
	s.Equal(int32(1), count)
}

func (s *GraphQLImageVulnerabilityV2TestSuite) TestImageVulnerabilityExceptionCountApprovedWithImageScope() {
	// TODO(ROX-27780): Defer until vuln requests updates are made
	s.T().Skip()

	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())
	status := []*string{pointers.String(storage.RequestStatus_APPROVED.String())}
	ctx = scoped.Context(ctx, scoped.Scope{
		IDs:   []string{"sha1"},
		Level: v1.SearchCategory_IMAGES,
	})
	args := struct {
		RequestStatus *[]*string
	}{
		RequestStatus: &status,
	}

	// Deferral:
	// - sha1 all tags (covers this specific tag)
	vuln := s.getImageVulnerabilityResolver(ctx, "cve-2018-1#")
	count, err := vuln.ExceptionCount(ctx, args)
	s.NoError(err)
	s.Equal(int32(1), count)

	vuln = s.getImageVulnerabilityResolver(ctx, "cve-2017-2#")
	count, err = vuln.ExceptionCount(ctx, args)
	s.NoError(err)
	s.Equal(int32(0), count)

	// False-positive:
	// global (covers this specific image)
	vuln = s.getImageVulnerabilityResolver(ctx, "cve-2019-2#")
	count, err = vuln.ExceptionCount(ctx, args)
	s.NoError(err)
	s.Equal(int32(1), count)
}

func (s *GraphQLImageVulnerabilityV2TestSuite) TestImageVulnerabilityExceptionCountPendingUpdateWithImageScope() {
	// TODO(ROX-27780): Defer until vuln requests updates are made
	s.T().Skip()

	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())
	status := []*string{pointers.String(storage.RequestStatus_APPROVED_PENDING_UPDATE.String())}
	ctx = scoped.Context(ctx, scoped.Scope{
		IDs:   []string{"sha2"},
		Level: v1.SearchCategory_IMAGES,
	})
	args := struct {
		RequestStatus *[]*string
	}{
		RequestStatus: &status,
	}

	// sha2 all tags
	vuln := s.getImageVulnerabilityResolver(ctx, "cve-2017-2#")
	count, err := vuln.ExceptionCount(ctx, args)
	s.NoError(err)
	s.Equal(int32(1), count)

	vuln = s.getImageVulnerabilityResolver(ctx, "cve-2019-1#")
	count, err = vuln.ExceptionCount(ctx, args)
	s.NoError(err)
	s.Equal(int32(0), count)
}

func (s *GraphQLImageVulnerabilityV2TestSuite) TestImageVulnerabilityExceptionCountTagless() {
	// TODO(ROX-27780): Defer until vuln requests updates are made
	s.T().Skip()

	taglessImage := testImages()[1]
	taglessImage.Id = "sha3"
	taglessImage.Name.Tag = ""
	err := s.resolver.ImageDataStore.UpsertImage(s.ctx, taglessImage)
	s.NoError(err)
	// Revert the upsert so that other tests are not affected.
	defer func() {
		s.NoError(s.resolver.ImageDataStore.DeleteImages(s.ctx, "sha3"))
	}()

	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())
	args := struct {
		RequestStatus *[]*string
	}{}

	// Deferral:
	// - sha3 tagless
	vuln := s.getImageVulnerabilityResolver(ctx, "cve-2017-1#")
	count, err := vuln.ExceptionCount(ctx, args)
	s.NoError(err)
	s.Equal(int32(1), count)

	ctx = scoped.Context(ctx, scoped.Scope{
		IDs:   []string{"sha1"},
		Level: v1.SearchCategory_IMAGES,
	})

	vuln = s.getImageVulnerabilityResolver(ctx, "cve-2017-1#")
	count, err = vuln.ExceptionCount(ctx, args)
	s.NoError(err)
	s.Equal(int32(0), count)

	ctx = SetAuthorizerOverride(s.ctx, allow.Anonymous())
	ctx = scoped.Context(ctx, scoped.Scope{
		IDs:   []string{"sha3"},
		Level: v1.SearchCategory_IMAGES,
	})

	// Deferral:
	// - sha3 tagless
	vuln = s.getImageVulnerabilityResolver(ctx, "cve-2017-1#")
	count, err = vuln.ExceptionCount(ctx, args)
	s.NoError(err)
	s.Equal(int32(1), count)
}

func (s *GraphQLImageVulnerabilityV2TestSuite) getImageResolver(ctx context.Context, id string) *imageResolver {
	imageID := graphql.ID(id)

	image, err := s.resolver.Image(ctx, struct{ ID graphql.ID }{ID: imageID})
	s.NoError(err)
	s.Equal(imageID, image.Id(ctx))
	return image
}

func (s *GraphQLImageVulnerabilityV2TestSuite) getImageVulnerabilityResolver(ctx context.Context, id string) ImageVulnerabilityResolver {
	vulnID := graphql.ID(id)

	vuln, err := s.resolver.ImageVulnerability(ctx, IDQuery{ID: &vulnID})
	s.NoError(err)
	s.Equal(vulnID, vuln.Id(ctx))
	return vuln
}

func getCVEList(ctx context.Context, vulns []ImageVulnerabilityResolver) []string {
	cveList := make([]string, 0, len(vulns))
	for _, vuln := range vulns {
		cveList = append(cveList, vuln.CVE(ctx))
	}
	return cveList
}

func (s *GraphQLImageVulnerabilityV2TestSuite) getIDMap() map[string]string {
	return map[string]string{
		cve111: getTestCVEID(s.T(), &storage.EmbeddedVulnerability{Cve: "cve-2018-1",
			SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
				FixedBy: "1.1",
			},
			Severity: storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
		}, s.componentIDMap[comp11]),
		cve121: getTestCVEID(s.T(), &storage.EmbeddedVulnerability{Cve: "cve-2018-1",
			SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
				FixedBy: "1.5",
			},
			Severity: storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
		}, s.componentIDMap[comp21]),
		cve231: getTestCVEID(s.T(), &storage.EmbeddedVulnerability{Cve: "cve-2019-1",
			Cvss:     4,
			Severity: storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
		}, s.componentIDMap[comp31]),
		cve331: getTestCVEID(s.T(), &storage.EmbeddedVulnerability{Cve: "cve-2019-2",
			Cvss:     3,
			Severity: storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY,
		}, s.componentIDMap[comp31]),
		cve112: getTestCVEID(s.T(), &storage.EmbeddedVulnerability{Cve: "cve-2018-1",
			SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
				FixedBy: "1.1",
			},
			Severity: storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
		}, s.componentIDMap[comp12]),
		cve232: getTestCVEID(s.T(), &storage.EmbeddedVulnerability{Cve: "cve-2019-1",
			Severity: storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
			Cvss:     4,
		}, s.componentIDMap[comp32]),
		cve332: getTestCVEID(s.T(), &storage.EmbeddedVulnerability{Cve: "cve-2019-2",
			Severity: storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY,
			Cvss:     3,
		}, s.componentIDMap[comp32]),
		cve442: getTestCVEID(s.T(), &storage.EmbeddedVulnerability{Cve: "cve-2017-1",
			Severity: storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY,
		}, s.componentIDMap[comp42]),
		cve542: getTestCVEID(s.T(), &storage.EmbeddedVulnerability{Cve: "cve-2017-2",
			Severity: storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY,
		}, s.componentIDMap[comp42]),
	}
}

func (s *GraphQLImageVulnerabilityV2TestSuite) getComponentIDMap() map[string]string {
	return map[string]string{
		comp11: getTestComponentID(s.T(), testImages()[0].GetScan().GetComponents()[0], "sha1"),
		comp12: getTestComponentID(s.T(), testImages()[1].GetScan().GetComponents()[0], "sha2"),
		comp21: getTestComponentID(s.T(), testImages()[0].GetScan().GetComponents()[1], "sha1"),
		comp31: getTestComponentID(s.T(), testImages()[0].GetScan().GetComponents()[2], "sha1"),
		comp32: getTestComponentID(s.T(), testImages()[1].GetScan().GetComponents()[1], "sha2"),
		comp42: getTestComponentID(s.T(), testImages()[1].GetScan().GetComponents()[2], "sha2"),
	}
}
