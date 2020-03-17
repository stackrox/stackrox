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
		schema.AddQuery("processComments(processId: ID!): [Comment!]!"),
		schema.AddMutation("addProcessComment(processId: ID!, commentMessage: String!): String!"),
		schema.AddMutation("updateProcessComment(processId: ID!, commentId: ID!, commentMessage: String!): Boolean!"),
		schema.AddMutation("removeProcessComment(processId: ID!, commentId: ID!): Boolean!"),
	)
}

// ProcessComments returns a list of comments for a process.
func (resolver *Resolver) ProcessComments(ctx context.Context, args struct{ ProcessID graphql.ID }) ([]*commentResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "ProcessComments")
	if err := readIndicators(ctx); err != nil {
		return nil, err
	}
	return resolver.wrapComments(
		resolver.ProcessIndicatorStore.GetCommentsForProcess(ctx, string(args.ProcessID)))
}

// AddProcessComment adds a process comment.
func (resolver *Resolver) AddProcessComment(ctx context.Context, args struct {
	ProcessID      graphql.ID
	CommentMessage string
}) (string, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "AddProcessComment")
	if err := writeIndicators(ctx); err != nil {
		return "", err
	}

	comment := &storage.Comment{
		CommentMessage: args.CommentMessage,
	}
	commentID, err := resolver.ProcessIndicatorStore.AddProcessComment(ctx, string(args.ProcessID), comment)
	if err != nil {
		return "", err
	}
	return commentID, nil
}

// UpdateProcessComment updates a process comment.
func (resolver *Resolver) UpdateProcessComment(ctx context.Context, args struct {
	ProcessID, CommentID graphql.ID
	CommentMessage       string
}) (bool, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "UpdateProcessComment")
	if err := writeIndicators(ctx); err != nil {
		return false, err
	}
	request := &storage.Comment{
		CommentId:      string(args.CommentID),
		CommentMessage: args.CommentMessage,
	}

	err := resolver.ProcessIndicatorStore.UpdateProcessComment(ctx, string(args.ProcessID), request)
	if err != nil {
		return false, err
	}

	return true, nil
}

// RemoveProcessComment removes a process comment.
func (resolver *Resolver) RemoveProcessComment(ctx context.Context, args struct{ ProcessID, CommentID graphql.ID }) (bool, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "RemoveProcessComment")
	if err := writeIndicators(ctx); err != nil {
		return false, err
	}

	err := resolver.ProcessIndicatorStore.RemoveProcessComment(ctx, string(args.ProcessID), string(args.CommentID))
	if err != nil {
		return false, err
	}

	return true, nil
}
