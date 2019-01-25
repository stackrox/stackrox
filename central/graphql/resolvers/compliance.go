package resolvers

import (
	"context"

	"github.com/graph-gophers/graphql-go"
	"github.com/stackrox/rox/central/compliance/aggregation"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/search"
)

var (
	log = logging.LoggerForModule()
)

func init() {
	if !features.Compliance.Enabled() {
		return
	}
	schema := getBuilder()
	schema.AddQuery("complianceStandard(id:ID!): ComplianceStandardMetadata")
	schema.AddQuery("complianceStandards: [ComplianceStandardMetadata!]!")
	schema.AddQuery("aggregatedResults(groupBy:[ComplianceAggregation_Scope!],unit:ComplianceAggregation_Scope!,where:String): ComplianceAggregation_Response!")
	schema.AddExtraResolver("ComplianceStandardMetadata", "controls: [ComplianceControl!]!")
}

// ComplianceStandards returns graphql resolvers for all compliance standards
func (resolver *Resolver) ComplianceStandards(ctx context.Context) ([]*complianceStandardMetadataResolver, error) {
	if err := complianceAuth(ctx); err != nil {
		return nil, err
	}
	return resolver.wrapComplianceStandardMetadatas(
		resolver.ComplianceStandardStore.Standards())
}

// ComplianceStandard returns a graphql resolver for a named compliance standard
func (resolver *Resolver) ComplianceStandard(ctx context.Context, args struct{ graphql.ID }) (*complianceStandardMetadataResolver, error) {
	if err := complianceAuth(ctx); err != nil {
		return nil, err
	}
	return resolver.wrapComplianceStandardMetadata(
		resolver.ComplianceStandardStore.StandardMetadata(string(args.ID)))
}

type aggregatedResultQuery struct {
	GroupBy *[]string
	Unit    string
	Where   *string
}

// AggregatedResults returns the aggregration of the last runs aggregated by scope, unit and filtered by a query
func (resolver *Resolver) AggregatedResults(ctx context.Context, args aggregatedResultQuery) (*complianceAggregation_ResponseResolver, error) {
	if err := complianceAuth(ctx); err != nil {
		return nil, err
	}

	standards, err := resolver.ComplianceStandardStore.Standards()
	if err != nil {
		return nil, err
	}

	clusters, err := resolver.ClusterDataStore.GetClusters()
	if err != nil {
		return nil, err
	}

	var clusterIDs, standardIDs []string
	if args.Where != nil {
		searchMap := search.ParseRawQueryIntoMap(*args.Where)
		standardIDs = aggregation.FilterStandards(standards, searchMap[search.Standard.String()])
		clusterIDs = aggregation.FilterClusters(clusters, searchMap[search.Cluster.String()])
	} else {
		standardIDs = aggregation.FilterStandards(standards, nil)
		clusterIDs = aggregation.FilterClusters(clusters, nil)
	}

	runResults, err := resolver.ComplianceDataStore.GetLatestRunResultsBatch(clusterIDs, standardIDs)
	if err != nil {
		return nil, err
	}
	results := aggregation.GetAggregatedResults(toComplianceAggregation_Scopes(args.GroupBy), toComplianceAggregation_Scope(&args.Unit), runResults)
	return resolver.wrapComplianceAggregation_Response(&v1.ComplianceAggregation_Response{
		Results: results,
	}, true, nil)
}

// ComplianceResults returns graphql resolvers for all matching compliance results
func (resolver *Resolver) ComplianceResults(ctx context.Context, query rawQuery) ([]*complianceControlResultResolver, error) {
	if err := complianceAuth(ctx); err != nil {
		return nil, err
	}
	q, err := query.AsV1Query()
	if err != nil {
		return nil, err
	}
	return resolver.wrapComplianceControlResults(
		resolver.ComplianceDataStore.QueryControlResults(q))
}

func (resolver *complianceStandardMetadataResolver) Controls(ctx context.Context) ([]*complianceControlResolver, error) {
	if err := complianceAuth(ctx); err != nil {
		return nil, err
	}
	return resolver.root.wrapComplianceControls(
		resolver.root.ComplianceStandardStore.Controls(resolver.data.GetId()))
}
