package resolvers

import (
	"context"

	"github.com/gogo/protobuf/types"
	"github.com/graph-gophers/graphql-go"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/utils"
)

func init() {
	schema := getBuilder()
	utils.Must(
		schema.AddQuery("complianceRecentRuns(clusterId:ID, standardId:ID, since:Time): [ComplianceRun!]!"),
		schema.AddQuery("complianceRun(id:ID!): ComplianceRun"),
		schema.AddMutation("complianceTriggerRuns(clusterId:ID!,standardId:ID!): [ComplianceRun!]!"),
		schema.AddQuery("complianceRunStatuses(ids: [ID!]!, latest Boolean): GetComplianceRunStatusesResponse!"),
	)
}

// ComplianceTriggerRuns is a mutation to trigger compliance runs on a specific cluster and standard (or all clusters/all standards)
func (resolver *Resolver) ComplianceTriggerRuns(ctx context.Context, args struct{ ClusterID, StandardID graphql.ID }) ([]*complianceRunResolver, error) {
	if err := writeCompliance(ctx); err != nil {
		return nil, err
	}

	resp, err := resolver.processWithAuditLog(ctx, args, "ComplianceTriggerRuns", func() (interface{}, error) {
		resp, err := resolver.ComplianceManagementService.TriggerRuns(ctx, &v1.TriggerComplianceRunsRequest{
			Selection: &v1.ComplianceRunSelection{
				ClusterId:  string(args.ClusterID),
				StandardId: string(args.StandardID),
			},
		})

		return resolver.wrapComplianceRuns(resp.GetStartedRuns(), err)
	})

	if resp == nil {
		return nil, err
	}

	return resp.([]*complianceRunResolver), err
}

// ComplianceRunStatuses is a query to obtain the statuses of a list of compliance runs.
func (resolver *Resolver) ComplianceRunStatuses(ctx context.Context, args struct{ Ids []graphql.ID }) (*getComplianceRunStatusesResponseResolver, error) {
	if err := readCompliance(ctx); err != nil {
		return nil, err
	}
	idStrings := make([]string, len(args.Ids))
	for i, id := range args.Ids {
		idStrings[i] = string(id)
	}
	resp, err := resolver.ComplianceManagementService.GetRunStatuses(ctx, &v1.GetComplianceRunStatusesRequest{
		RunIds: idStrings,
	})
	return resolver.wrapGetComplianceRunStatusesResponse(resp, resp != nil, err)
}

// ComplianceRecentRuns is a resolver for recent compliance runs
func (resolver *Resolver) ComplianceRecentRuns(
	ctx context.Context,
	args struct {
		ClusterID, StandardID *graphql.ID
		Since                 *graphql.Time
	}) ([]*complianceRunResolver, error) {
	if err := readCompliance(ctx); err != nil {
		return nil, err
	}
	req := &v1.GetRecentComplianceRunsRequest{}
	if args.ClusterID != nil {
		req.ClusterIdOpt = &v1.GetRecentComplianceRunsRequest_ClusterId{ClusterId: string(*args.ClusterID)}
	}
	if args.StandardID != nil {
		req.StandardIdOpt = &v1.GetRecentComplianceRunsRequest_StandardId{StandardId: string(*args.StandardID)}
	}
	if args.Since != nil {
		t, err := types.TimestampProto(args.Since.Time)
		if err != nil {
			return nil, err
		}
		req.Since = t
	}
	runs, err := resolver.ComplianceManager.GetRecentRuns(ctx, req)
	if err != nil {
		return nil, err
	}
	return resolver.wrapComplianceRuns(runs, nil)
}

// ComplianceRun returns a specific compliance run, if it exists
func (resolver *Resolver) ComplianceRun(ctx context.Context, args struct{ graphql.ID }) (*complianceRunResolver, error) {
	if err := readCompliance(ctx); err != nil {
		return nil, err
	}
	run, err := resolver.ComplianceManager.GetRecentRun(ctx, string(args.ID))
	return resolver.wrapComplianceRun(run, run != nil, err)
}
