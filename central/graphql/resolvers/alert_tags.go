package resolvers

import (
	"context"
	"time"

	"github.com/graph-gophers/graphql-go"
	"github.com/stackrox/stackrox/central/metrics"
	pkgMetrics "github.com/stackrox/stackrox/pkg/metrics"
	"github.com/stackrox/stackrox/pkg/utils"
)

func init() {
	schema := getBuilder()
	utils.Must(
		schema.AddMutation("addAlertTags(resourceId: ID!, tags: [String!]!): [String!]!"),
		schema.AddMutation("removeAlertTags(resourceId: ID!, tags: [String!]!): Boolean!"),
		schema.AddMutation("bulkAddAlertTags(resourceIds: [ID!]!, tags: [String!]!): [String!]!"),
	)
}

// AddAlertTags adds tags to an alert
func (resolver *Resolver) AddAlertTags(ctx context.Context, args struct {
	ResourceID graphql.ID
	Tags       []string
}) ([]string, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "AddAlertTags")
	if err := writeAlerts(ctx); err != nil {
		return nil, err
	}
	tags, err := resolver.ViolationsDataStore.AddAlertTags(ctx, string(args.ResourceID), args.Tags)
	if err != nil {
		return nil, err
	}
	return tags, nil
}

// RemoveAlertTags removes tags from an alert
func (resolver *Resolver) RemoveAlertTags(ctx context.Context, args struct {
	ResourceID graphql.ID
	Tags       []string
}) (bool, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "RemoveAlertTags")
	if err := writeAlerts(ctx); err != nil {
		return false, err
	}
	err := resolver.ViolationsDataStore.RemoveAlertTags(ctx, string(args.ResourceID), args.Tags)
	if err != nil {
		return false, err
	}
	return true, nil
}

// BulkAddAlertTags adds tags to multi-alerts
func (resolver *Resolver) BulkAddAlertTags(ctx context.Context, args struct {
	ResourceIDs []graphql.ID
	Tags        []string
}) ([]string, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "BulkAddAlertTags")
	if err := writeAlerts(ctx); err != nil {
		return nil, err
	}
	var ids []string

	for _, id := range args.ResourceIDs {
		_, err := resolver.ViolationsDataStore.AddAlertTags(ctx, string(id), args.Tags)
		if err != nil {
			continue
		}
		ids = append(ids, string(id))
	}

	return ids, nil
}
