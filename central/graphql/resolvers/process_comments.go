package resolvers

import (
	"context"
	"time"

	"github.com/graph-gophers/graphql-go"
	"github.com/stackrox/rox/central/comments"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/storage"
	pkgMetrics "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/utils"
)

func init() {
	schema := getBuilder()
	utils.Must(
		schema.AddQuery("processComments(deploymentID: ID!, containerName: String!, execFilePath: String!, args: String!): [Comment!]!"),
		schema.AddMutation("addProcessComment(deploymentID: ID!, containerName: String!, execFilePath: String!, args: String!, commentMessage: String!): String!"),
		schema.AddMutation("updateProcessComment(deploymentID: ID!, containerName: String!, execFilePath: String!, args: String!, commentId: ID!, commentMessage: String!): Boolean!"),
		schema.AddMutation("removeProcessComment(deploymentID: ID!, containerName: String!, execFilePath: String!, args: String!, commentId: ID!): Boolean!"),
	)
}

// ProcessComments returns a list of comments for a process.
func (resolver *Resolver) ProcessComments(ctx context.Context, args struct {
	DeploymentID  graphql.ID
	ContainerName string
	ExecFilePath  string
	Args          string
}) ([]*commentResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "ProcessComments")
	if err := readIndicators(ctx); err != nil {
		return nil, err
	}
	return resolver.wrapComments(
		resolver.ProcessIndicatorStore.GetCommentsForProcess(ctx, &comments.ProcessCommentKey{
			DeploymentID:  string(args.DeploymentID),
			ContainerName: args.ContainerName,
			ExecFilePath:  args.ExecFilePath,
			Args:          args.Args,
		}))
}

// AddProcessComment adds a process comment.
func (resolver *Resolver) AddProcessComment(ctx context.Context, args struct {
	DeploymentID   graphql.ID
	ContainerName  string
	ExecFilePath   string
	Args           string
	CommentMessage string
}) (string, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "AddProcessComment")
	if err := writeIndicators(ctx); err != nil {
		return "", err
	}

	comment := &storage.Comment{
		CommentMessage: args.CommentMessage,
	}
	commentID, err := resolver.ProcessIndicatorStore.AddProcessComment(ctx, &comments.ProcessCommentKey{
		DeploymentID:  string(args.DeploymentID),
		ContainerName: args.ContainerName,
		ExecFilePath:  args.ExecFilePath,
		Args:          args.Args,
	}, comment)
	if err != nil {
		return "", err
	}
	return commentID, nil
}

// UpdateProcessComment updates a process comment.
func (resolver *Resolver) UpdateProcessComment(ctx context.Context, args struct {
	DeploymentID   graphql.ID
	ContainerName  string
	ExecFilePath   string
	Args           string
	CommentID      graphql.ID
	CommentMessage string
}) (bool, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "UpdateProcessComment")
	if err := writeIndicators(ctx); err != nil {
		return false, err
	}
	request := &storage.Comment{
		CommentId:      string(args.CommentID),
		CommentMessage: args.CommentMessage,
	}

	err := resolver.ProcessIndicatorStore.UpdateProcessComment(ctx, &comments.ProcessCommentKey{
		DeploymentID:  string(args.DeploymentID),
		ContainerName: args.ContainerName,
		ExecFilePath:  args.ExecFilePath,
		Args:          args.Args,
	}, request)
	if err != nil {
		return false, err
	}

	return true, nil
}

// RemoveProcessComment removes a process comment.
func (resolver *Resolver) RemoveProcessComment(ctx context.Context, args struct {
	DeploymentID  graphql.ID
	ContainerName string
	ExecFilePath  string
	Args          string
	CommentID     graphql.ID
}) (bool, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "RemoveProcessComment")
	if err := writeIndicators(ctx); err != nil {
		return false, err
	}

	err := resolver.ProcessIndicatorStore.RemoveProcessComment(ctx, &comments.ProcessCommentKey{
		DeploymentID:  string(args.DeploymentID),
		ContainerName: args.ContainerName,
		ExecFilePath:  args.ExecFilePath,
		Args:          args.Args,
	}, string(args.CommentID))
	if err != nil {
		return false, err
	}

	return true, nil
}
