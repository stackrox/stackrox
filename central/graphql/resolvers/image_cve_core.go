package resolvers

import (
	"context"
	"time"

	"github.com/graph-gophers/graphql-go"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/views/imagecve"
	"github.com/stackrox/rox/pkg/features"
	pkgMetrics "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/utils"
)

func init() {
	schema := getBuilder()
	utils.Must(
		// NOTE: This list is and should remain alphabetically ordered
		schema.AddType("ImageCVECore",
			[]string{
				"affectedImages: Int!",

				"cve: String!",
				"firstDiscoveredInSystem: Time",
				"topCVSS: Float!",
			}),
		schema.AddQuery("imageCVEs(query: String, pagination: Pagination): [ImageCVECore!]!"),
	)
}

type imageCVECoreResolver struct {
	ctx  context.Context
	root *Resolver
	data imagecve.CveCore
}

func (resolver *Resolver) wrapImageCVECoresWithContext(ctx context.Context, values []imagecve.CveCore, err error) ([]*imageCVECoreResolver, error) {
	if err != nil || len(values) == 0 {
		return nil, err
	}
	output := make([]*imageCVECoreResolver, len(values))
	for i, v := range values {
		output[i] = &imageCVECoreResolver{ctx: ctx, root: resolver, data: v}
	}
	return output, nil
}

// ImageCVEs returns graphQL resolver for image cves satisfying the specified query.
// Note: Client must explicitly pass observed/deferred CVEs.
func (resolver *Resolver) ImageCVEs(ctx context.Context, q PaginatedQuery) ([]*imageCVECoreResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "ImageCVEs")

	if !features.VulnMgmtWorkloadCVEs.Enabled() {
		return nil, errors.Errorf("%s=false. Set %s=true and retry", features.VulnMgmtWorkloadCVEs.Name(), features.VulnMgmtWorkloadCVEs.Name())
	}
	if err := readImages(ctx); err != nil {
		return nil, err
	}
	query, err := q.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}

	cves, err := resolver.ImageCVEView.Get(ctx, query)
	return resolver.wrapImageCVECoresWithContext(ctx, cves, err)
}

func (resolver *imageCVECoreResolver) AffectedImages(_ context.Context) int32 {
	return int32(resolver.data.GetAffectedImages())
}

func (resolver *imageCVECoreResolver) CVE(_ context.Context) string {
	return resolver.data.GetCVE()
}

func (resolver *imageCVECoreResolver) TopCVSS(_ context.Context) float64 {
	return float64(resolver.data.GetTopCVSS())
}

func (resolver *imageCVECoreResolver) FirstDiscoveredInSystem(_ context.Context) *graphql.Time {
	return &graphql.Time{
		Time: resolver.data.GetFirstDiscoveredInSystem(),
	}
}
