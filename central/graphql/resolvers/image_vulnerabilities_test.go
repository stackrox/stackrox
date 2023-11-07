//go:build sql_integration

package resolvers

import (
	"context"
	"testing"

	"github.com/graph-gophers/graphql-go"
	"github.com/stackrox/rox/central/graphql/resolvers/loaders"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	"github.com/stackrox/rox/pkg/pointers"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search/scoped"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestGraphQLImageVulnerabilityEndpoints(t *testing.T) {
	suite.Run(t, new(GraphQLImageVulnerabilityTestSuite))
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

type GraphQLImageVulnerabilityTestSuite struct {
	suite.Suite

	ctx      context.Context
	testDB   *pgtest.TestPostgres
	resolver *Resolver
}

func (s *GraphQLImageVulnerabilityTestSuite) SetupSuite() {

	s.ctx = loaders.WithLoaderContext(sac.WithAllAccess(context.Background()))
	mockCtrl := gomock.NewController(s.T())
	s.testDB = SetupTestPostgresConn(s.T())
	resolver, _ := SetupTestResolver(s.T(),
		CreateTestImageDatastore(s.T(), s.testDB, mockCtrl),
		CreateTestImageComponentDatastore(s.T(), s.testDB, mockCtrl),
		CreateTestImageCVEDatastore(s.T(), s.testDB),
		CreateTestImageComponentCVEEdgeDatastore(s.T(), s.testDB),
		CreateTestImageCVEEdgeDatastore(s.T(), s.testDB),
		TestVulnReqDatastore(s.T(), s.testDB),
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
}

func (s *GraphQLImageVulnerabilityTestSuite) TearDownSuite() {
	s.testDB.Teardown(s.T())
}

func (s *GraphQLImageVulnerabilityTestSuite) TestUnauthorizedImageVulnerabilityEndpoint() {
	_, err := s.resolver.ImageVulnerability(s.ctx, IDQuery{})
	s.Error(err, "Unauthorized request got through")
}

func (s *GraphQLImageVulnerabilityTestSuite) TestUnauthorizedImageVulnerabilitiesEndpoint() {
	_, err := s.resolver.ImageVulnerabilities(s.ctx, PaginatedQuery{})
	s.Error(err, "Unauthorized request got through")
}

func (s *GraphQLImageVulnerabilityTestSuite) TestUnauthorizedImageVulnerabilityCountEndpoint() {
	_, err := s.resolver.ImageVulnerabilityCount(s.ctx, RawQuery{})
	s.Error(err, "Unauthorized request got through")
}

func (s *GraphQLImageVulnerabilityTestSuite) TestUnauthorizedImageVulnerabilityCounterEndpoint() {
	_, err := s.resolver.ImageVulnerabilityCounter(s.ctx, RawQuery{})
	s.Error(err, "Unauthorized request got through")
}

func (s *GraphQLImageVulnerabilityTestSuite) TestUnauthorizedTopImageVulnerabilityEndpoint() {
	_, err := s.resolver.TopImageVulnerability(s.ctx, RawQuery{})
	s.Error(err, "Unauthorized request got through")
}

func (s *GraphQLImageVulnerabilityTestSuite) TestImageVulnerabilities() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	expected := int32(5)

	vulns, err := s.resolver.ImageVulnerabilities(ctx, PaginatedQuery{})
	s.NoError(err)
	s.Equal(expected, int32(len(vulns)))
	idList := getIDList(ctx, vulns)
	s.ElementsMatch([]string{"cve-2018-1#", "cve-2019-1#", "cve-2019-2#", "cve-2017-1#", "cve-2017-2#"}, idList)

	count, err := s.resolver.ImageVulnerabilityCount(ctx, RawQuery{})
	s.NoError(err)
	s.Equal(expected, count)

	counter, err := s.resolver.ImageVulnerabilityCounter(ctx, RawQuery{})
	s.NoError(err)
	checkVulnerabilityCounter(s.T(), counter, expected, 1, 1, 2, 1, 1)
}

func (s *GraphQLImageVulnerabilityTestSuite) TestImageVulnerabilitiesFixable() {
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
	idList := getIDList(ctx, vulns)
	s.ElementsMatch([]string{"cve-2018-1#"}, idList)

	count, err := s.resolver.ImageVulnerabilityCount(ctx, RawQuery{Query: &query})
	s.NoError(err)
	s.Equal(expected, count)
}

func (s *GraphQLImageVulnerabilityTestSuite) TestImageVulnerabilitiesNonFixable() {
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
	idList := getIDList(ctx, vulns)
	s.ElementsMatch([]string{"cve-2019-1#", "cve-2019-2#", "cve-2017-1#", "cve-2017-2#"}, idList)

	count, err := s.resolver.ImageVulnerabilityCount(ctx, RawQuery{Query: &query})
	s.NoError(err)
	s.Equal(expected, count)
}

func (s *GraphQLImageVulnerabilityTestSuite) TestImageVulnerabilitiesFixedByVersion() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	scopedCtx := scoped.Context(ctx, scoped.Scope{
		Level: v1.SearchCategory_IMAGE_COMPONENTS,
		ID:    "comp1#0.9#",
	})
	vuln := s.getImageVulnerabilityResolver(scopedCtx, "cve-2018-1#")

	fixedBy, err := vuln.FixedByVersion(ctx)
	s.NoError(err)
	s.Equal("1.1", fixedBy)

	scopedCtx = scoped.Context(ctx, scoped.Scope{
		Level: v1.SearchCategory_IMAGE_COMPONENTS,
		ID:    "comp2#1.1#",
	})
	vuln = s.getImageVulnerabilityResolver(scopedCtx, "cve-2018-1#")

	fixedBy, err = vuln.FixedByVersion(ctx)
	s.NoError(err)
	s.Equal("1.5", fixedBy)

	scopedCtx = scoped.Context(ctx, scoped.Scope{
		Level: v1.SearchCategory_IMAGE_COMPONENTS,
		ID:    "comp2#1.1#",
	})
	vuln = s.getImageVulnerabilityResolver(scopedCtx, "cve-2017-1#")

	fixedBy, err = vuln.FixedByVersion(ctx)
	s.NoError(err)
	s.Equal("", fixedBy)
}

func (s *GraphQLImageVulnerabilityTestSuite) TestImageVulnerabilitiesScoped() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	image := s.getImageResolver(ctx, "sha1")
	expected := int32(3)

	vulns, err := image.ImageVulnerabilities(ctx, PaginatedQuery{})
	s.NoError(err)
	s.Equal(expected, int32(len(vulns)))
	idList := getIDList(ctx, vulns)
	s.ElementsMatch([]string{"cve-2018-1#", "cve-2019-1#", "cve-2019-2#"}, idList)

	count, err := image.ImageVulnerabilityCount(ctx, RawQuery{})
	s.NoError(err)
	s.Equal(expected, count)

	counter, err := image.ImageVulnerabilityCounter(ctx, RawQuery{})
	s.NoError(err)
	checkVulnerabilityCounter(s.T(), counter, expected, 1, 1, 0, 1, 1)

	image = s.getImageResolver(ctx, "sha2")
	expected = int32(5)

	vulns, err = image.ImageVulnerabilities(ctx, PaginatedQuery{})
	s.NoError(err)
	s.Equal(expected, int32(len(vulns)))
	idList = getIDList(ctx, vulns)
	s.ElementsMatch([]string{"cve-2018-1#", "cve-2019-1#", "cve-2019-2#", "cve-2017-1#", "cve-2017-2#"}, idList)

	count, err = image.ImageVulnerabilityCount(ctx, RawQuery{})
	s.NoError(err)
	s.Equal(expected, count)

	counter, err = image.ImageVulnerabilityCounter(ctx, RawQuery{})
	s.NoError(err)
	checkVulnerabilityCounter(s.T(), counter, expected, 1, 1, 2, 1, 1)
}

func (s *GraphQLImageVulnerabilityTestSuite) TestImageVulnerabilityMiss() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	vulnID := graphql.ID("invalid")

	_, err := s.resolver.ImageVulnerability(ctx, IDQuery{ID: &vulnID})
	s.Error(err)
}

func (s *GraphQLImageVulnerabilityTestSuite) TestImageVulnerabilityHit() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	vulnID := graphql.ID("cve-2018-1#")

	vuln, err := s.resolver.ImageVulnerability(ctx, IDQuery{ID: &vulnID})
	s.NoError(err)
	s.Equal(vulnID, vuln.Id(ctx))
}

func (s *GraphQLImageVulnerabilityTestSuite) TestTopImageVulnerabilityUnscoped() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	_, err := s.resolver.TopImageVulnerability(ctx, RawQuery{})
	s.Error(err)
}

func (s *GraphQLImageVulnerabilityTestSuite) TestTopImageVulnerability() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	image := s.getImageResolver(ctx, "sha1")

	expected := graphql.ID("cve-2019-1#")
	topVuln, err := image.TopImageVulnerability(ctx, RawQuery{})
	s.NoError(err)
	s.Equal(expected, topVuln.Id(ctx))
}

func (s *GraphQLImageVulnerabilityTestSuite) TestImageVulnerabilityImages() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	vuln := s.getImageVulnerabilityResolver(ctx, "cve-2018-1#")

	images, err := vuln.Images(ctx, PaginatedQuery{})
	s.NoError(err)
	s.Equal(2, len(images))
	idList := getIDList(ctx, images)
	s.ElementsMatch([]string{"sha1", "sha2"}, idList)

	count, err := vuln.ImageCount(ctx, RawQuery{})
	s.NoError(err)
	s.Equal(int32(len(images)), count)

	vuln = s.getImageVulnerabilityResolver(ctx, "cve-2017-1#")

	images, err = vuln.Images(ctx, PaginatedQuery{})
	s.NoError(err)
	s.Equal(1, len(images))
	idList = getIDList(ctx, images)
	s.ElementsMatch([]string{"sha2"}, idList)

	count, err = vuln.ImageCount(ctx, RawQuery{})
	s.NoError(err)
	s.Equal(int32(len(images)), count)
}

func (s *GraphQLImageVulnerabilityTestSuite) TestImageVulnerabilityImageComponents() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	vuln := s.getImageVulnerabilityResolver(ctx, "cve-2018-1#")

	comps, err := vuln.ImageComponents(ctx, PaginatedQuery{})
	s.NoError(err)
	s.Equal(2, len(comps))
	idList := getIDList(ctx, comps)
	s.ElementsMatch([]string{"comp1#0.9#", "comp2#1.1#"}, idList)

	count, err := vuln.ImageComponentCount(ctx, RawQuery{})
	s.NoError(err)
	s.Equal(int32(len(comps)), count)

	vuln = s.getImageVulnerabilityResolver(ctx, "cve-2017-1#")

	comps, err = vuln.ImageComponents(ctx, PaginatedQuery{})
	s.NoError(err)
	s.Equal(1, len(comps))
	idList = getIDList(ctx, comps)
	s.ElementsMatch([]string{"comp4#1.0#"}, idList)

	count, err = vuln.ImageComponentCount(ctx, RawQuery{})
	s.NoError(err)
	s.Equal(int32(len(comps)), count)
}

func (s *GraphQLImageVulnerabilityTestSuite) TestImageVulnerabilityExceptionCountAll() {
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

func (s *GraphQLImageVulnerabilityTestSuite) TestImageVulnerabilityExceptionCountPending() {
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

func (s *GraphQLImageVulnerabilityTestSuite) TestImageVulnerabilityExceptionCountApproved() {
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

func (s *GraphQLImageVulnerabilityTestSuite) TestImageVulnerabilityExceptionCountPendingUpdate() {
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

func (s *GraphQLImageVulnerabilityTestSuite) TestImageVulnerabilityExceptionCountAllWithImageScope() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())
	ctx = scoped.Context(ctx, scoped.Scope{
		ID:    "sha1",
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

func (s *GraphQLImageVulnerabilityTestSuite) TestImageVulnerabilityExceptionCountPendingWithImageScope() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())
	status := []*string{pointers.String(storage.RequestStatus_PENDING.String())}
	ctx = scoped.Context(ctx, scoped.Scope{
		ID:    "sha1",
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

func (s *GraphQLImageVulnerabilityTestSuite) TestImageVulnerabilityExceptionCountApprovedWithImageScope() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())
	status := []*string{pointers.String(storage.RequestStatus_APPROVED.String())}
	ctx = scoped.Context(ctx, scoped.Scope{
		ID:    "sha1",
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

func (s *GraphQLImageVulnerabilityTestSuite) TestImageVulnerabilityExceptionCountPendingUpdateWithImageScope() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())
	status := []*string{pointers.String(storage.RequestStatus_APPROVED_PENDING_UPDATE.String())}
	ctx = scoped.Context(ctx, scoped.Scope{
		ID:    "sha2",
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

func (s *GraphQLImageVulnerabilityTestSuite) TestImageVulnerabilityExceptionCountTagless() {
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
		ID:    "sha1",
		Level: v1.SearchCategory_IMAGES,
	})

	vuln = s.getImageVulnerabilityResolver(ctx, "cve-2017-1#")
	count, err = vuln.ExceptionCount(ctx, args)
	s.NoError(err)
	s.Equal(int32(0), count)

	ctx = SetAuthorizerOverride(s.ctx, allow.Anonymous())
	ctx = scoped.Context(ctx, scoped.Scope{
		ID:    "sha3",
		Level: v1.SearchCategory_IMAGES,
	})

	// Deferral:
	// - sha3 tagless
	vuln = s.getImageVulnerabilityResolver(ctx, "cve-2017-1#")
	count, err = vuln.ExceptionCount(ctx, args)
	s.NoError(err)
	s.Equal(int32(1), count)
}

func (s *GraphQLImageVulnerabilityTestSuite) getImageResolver(ctx context.Context, id string) *imageResolver {
	imageID := graphql.ID(id)

	image, err := s.resolver.Image(ctx, struct{ ID graphql.ID }{ID: imageID})
	s.NoError(err)
	s.Equal(imageID, image.Id(ctx))
	return image
}

func (s *GraphQLImageVulnerabilityTestSuite) getImageComponentResolver(ctx context.Context, id string) ImageComponentResolver {
	compID := graphql.ID(id)

	comp, err := s.resolver.ImageComponent(ctx, IDQuery{ID: &compID})
	s.NoError(err)
	s.Equal(compID, comp.Id(ctx))
	return comp
}

func (s *GraphQLImageVulnerabilityTestSuite) getImageVulnerabilityResolver(ctx context.Context, id string) ImageVulnerabilityResolver {
	vulnID := graphql.ID(id)

	vuln, err := s.resolver.ImageVulnerability(ctx, IDQuery{ID: &vulnID})
	s.NoError(err)
	s.Equal(vulnID, vuln.Id(ctx))
	return vuln
}
