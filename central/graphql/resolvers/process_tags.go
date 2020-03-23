package resolvers

import (
	"context"
	"time"

	"github.com/graph-gophers/graphql-go"
	"github.com/stackrox/rox/central/analystnotes"
	"github.com/stackrox/rox/central/metrics"
	pkgMetrics "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/utils"
)

func init() {
	schema := getBuilder()
	utils.Must(
		schema.AddQuery("processTags(deploymentID: ID!, containerName: String!, execFilePath: String!, args: String!): [String!]!"),
		schema.AddMutation("addProcessTags(deploymentID: ID!, containerName: String!, execFilePath: String!, args: String!, tags: [String!]!): Boolean!"),
		schema.AddMutation("removeProcessTags(deploymentID: ID!, containerName: String!, execFilePath: String!, args: String!, tags: [String!]!): Boolean!"),
	)
}

// ProcessTags retrieves process tags.
func (resolver *Resolver) ProcessTags(ctx context.Context, args struct {
	DeploymentID  graphql.ID
	ContainerName string
	ExecFilePath  string
	Args          string
}) ([]string, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "ProcessTags")
	if err := writeIndicators(ctx); err != nil {
		return nil, err
	}
	tags, err := resolver.DeploymentDataStore.GetTagsForProcessKey(ctx, &analystnotes.ProcessNoteKey{
		DeploymentID:  string(args.DeploymentID),
		ContainerName: args.ContainerName,
		ExecFilePath:  args.ExecFilePath,
		Args:          args.Args,
	})
	if err != nil {
		return nil, err
	}
	return tags, nil
}

// AddProcessTags adds process tags.
func (resolver *Resolver) AddProcessTags(ctx context.Context, args struct {
	DeploymentID  graphql.ID
	ContainerName string
	ExecFilePath  string
	Args          string
	Tags          []string
}) (bool, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "AddProcessTags")
	if err := writeIndicators(ctx); err != nil {
		return false, err
	}
	err := resolver.DeploymentDataStore.AddTagsToProcessKey(ctx, &analystnotes.ProcessNoteKey{
		DeploymentID:  string(args.DeploymentID),
		ContainerName: args.ContainerName,
		ExecFilePath:  args.ExecFilePath,
		Args:          args.Args,
	}, args.Tags)
	if err != nil {
		return false, err
	}
	return true, nil
}

// RemoveProcessTags removes process tags.
func (resolver *Resolver) RemoveProcessTags(ctx context.Context, args struct {
	DeploymentID  graphql.ID
	ContainerName string
	ExecFilePath  string
	Args          string
	Tags          []string
}) (bool, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "RemoveProcessTags")
	if err := writeIndicators(ctx); err != nil {
		return false, err
	}
	err := resolver.DeploymentDataStore.RemoveTagsFromProcessKey(ctx, &analystnotes.ProcessNoteKey{
		DeploymentID:  string(args.DeploymentID),
		ContainerName: args.ContainerName,
		ExecFilePath:  args.ExecFilePath,
		Args:          args.Args,
	}, args.Tags)
	if err != nil {
		return false, err
	}
	return true, nil
}
