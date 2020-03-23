package resolvers

import (
	"context"
	"time"

	"github.com/graph-gophers/graphql-go"
	"github.com/stackrox/rox/central/metrics"
	pkgMetrics "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/utils"
)

func init() {
	schema := getBuilder()
	utils.Must(
		schema.AddMutation("addAlertTags(resourceId: ID!, tags: [String!]!): [String!]!"),
		schema.AddMutation("removeAlertTags(resourceId: ID!, tags: [String!]!): Boolean!"),
	)
}

//AddAlertTags adds tags to an alert
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

//RemoveAlertTags removes tags from an alert
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
