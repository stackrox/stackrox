package resolvers

import (
	"context"

	"github.com/stackrox/rox/central/graphql/resolvers/loaders"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/utils"
)

func init() {
	schema := getBuilder()
	utils.Must(
		schema.AddType("PlottedImageVulnerabilities", []string{
			"basicImageVulnerabilityCounter: VulnerabilityCounter!",
			"imageVulnerabilities(pagination: Pagination): [ImageVulnerability]!",
		}),
	)
}

// PlottedImageVulnerabilitiesResolver returns the data required by top risky images scatter-plot on vuln mgmt dashboard
type PlottedImageVulnerabilitiesResolver struct {
	root    *Resolver
	all     []string
	fixable int
}

func (resolver *Resolver) wrapPlottedImageVulnerabilities(all []string, fixable int) (*PlottedImageVulnerabilitiesResolver, error) {
	return &PlottedImageVulnerabilitiesResolver{
		root:    resolver,
		all:     all,
		fixable: fixable,
	}, nil
}

func (resolver *Resolver) PlottedImageVulnerabilities(ctx context.Context, args RawQuery) (*PlottedImageVulnerabilitiesResolver, error) {
	if !features.PostgresDatastore.Enabled() {
		query := withImageCveTypeFiltering(args.String())
		allCveIds, fixableCount, err := getPlottedVulnsIdsAndFixableCount(ctx, resolver, RawQuery{Query: &query})
		if err != nil {
			return nil, err
		}

		return resolver.wrapPlottedImageVulnerabilities(allCveIds, fixableCount)
	}

	query, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}
	query = tryUnsuppressedQuery(query)
	logErrorOnQueryContainingField(query, search.Fixable, "PlottedImageVulnerabilities")

	vulnLoader, err := loaders.GetImageCVELoader(ctx)
	if err != nil {
		return nil, err
	}
	allCves, err := vulnLoader.FromQuery(ctx, query)
	if err != nil {
		return nil, err
	}
	allCveIds := make([]string, 0, len(allCves))
	for _, cve := range allCves {
		allCveIds = append(allCveIds, cve.GetId())
	}

	fixableCount, err := vulnLoader.CountFromQuery(ctx,
		search.ConjunctionQuery(query, search.NewQueryBuilder().AddBools(search.Fixable, true).ProtoQuery()))
	if err != nil {
		return nil, err
	}

	return resolver.wrapPlottedImageVulnerabilities(allCveIds, int(fixableCount))
}

// BasicImageVulnerabilityCounter returns the ImageVulnerabilityCounter for scatter-plot with only total and fixable
func (resolver *PlottedImageVulnerabilitiesResolver) BasicImageVulnerabilityCounter(_ context.Context) (*VulnerabilityCounterResolver, error) {
	return &VulnerabilityCounterResolver{
		all: &VulnerabilityFixableCounterResolver{
			total:   int32(len(resolver.all)),
			fixable: int32(resolver.fixable),
		},
	}, nil
}

// ImageVulnerabilities returns the image vulnerabilities for top risky images scatter-plot
func (resolver *PlottedImageVulnerabilitiesResolver) ImageVulnerabilities(ctx context.Context, args PaginatedQuery) ([]ImageVulnerabilityResolver, error) {
	if len(resolver.all) == 0 {
		return nil, nil
	}
	q := search.NewQueryBuilder().AddExactMatches(search.CVEID, resolver.all...).Query()
	return resolver.root.ImageVulnerabilities(ctx, PaginatedQuery{Query: &q, Pagination: args.Pagination})
}
