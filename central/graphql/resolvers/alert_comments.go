package resolvers

import (
	"context"
	"time"

	"github.com/graph-gophers/graphql-go"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/storage"
	pkgMetrics "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/utils"
)

func init() {
	schema := getBuilder()
	utils.Must(
		schema.AddQuery("alertComments(resourceId: ID!): [Comment!]!"),
		schema.AddMutation("addAlertComment(resourceId: ID!, commentMessage: String!): String!"),
		schema.AddMutation("updateAlertComment(resourceId: ID!, commentId: ID!, commentMessage: String!): Boolean!"),
		schema.AddMutation("removeAlertComment(resourceId: ID!, commentId: ID!): Boolean!"),
	)
}

// AlertComments returns a list of comments for an alert
func (resolver *Resolver) AlertComments(ctx context.Context, args struct{ ResourceID graphql.ID }) ([]*commentResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "AlertComments")
	if err := readAlerts(ctx); err != nil {
		return nil, err
	}
	return resolver.wrapComments(
		resolver.ViolationsDataStore.GetAlertComments(ctx, string(args.ResourceID)))
}

// AddAlertComment adds a comment to an alert
func (resolver *Resolver) AddAlertComment(ctx context.Context, args struct {
	ResourceID     graphql.ID
	CommentMessage string
}) (string, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "AddAlertComment")
	if err := writeAlerts(ctx); err != nil {
		return "", err
	}
	request := &storage.Comment{
		ResourceId:     string(args.ResourceID),
		CommentMessage: args.CommentMessage,
	}
	commentID, err := resolver.ViolationsDataStore.AddAlertComment(ctx, request)
	if err != nil {
		return "", err
	}
	return commentID, nil
}

// UpdateAlertComment updates an existing alert comment
func (resolver *Resolver) UpdateAlertComment(ctx context.Context, args struct {
	ResourceID, CommentID graphql.ID
	CommentMessage        string
}) (bool, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "UpdateAlertComment")
	if err := writeAlerts(ctx); err != nil {
		return false, err
	}
	request := &storage.Comment{
		ResourceId:     string(args.ResourceID),
		CommentId:      string(args.CommentID),
		CommentMessage: args.CommentMessage,
	}

	err := resolver.ViolationsDataStore.UpdateAlertComment(ctx, request)
	if err != nil {
		return false, err
	}

	return true, nil
}

// RemoveAlertComment deletes an alert comment
func (resolver *Resolver) RemoveAlertComment(ctx context.Context, args struct{ ResourceID, CommentID graphql.ID }) (bool, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "RemoveAlertComment")
	if err := writeAlerts(ctx); err != nil {
		return false, err
	}

	err := resolver.ViolationsDataStore.RemoveAlertComment(ctx, string(args.ResourceID), string(args.CommentID))
	if err != nil {
		return false, err
	}

	return true, nil
}
