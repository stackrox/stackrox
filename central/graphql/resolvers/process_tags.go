package resolvers

import (
	"context"
	"time"

	"github.com/stackrox/stackrox/central/analystnotes"
	"github.com/stackrox/stackrox/central/metrics"
	pkgMetrics "github.com/stackrox/stackrox/pkg/metrics"
	"github.com/stackrox/stackrox/pkg/utils"
)

func init() {
	schema := getBuilder()
	utils.Must(
		schema.AddQuery("processTags(key: ProcessNoteKey!): [String!]!"),
		schema.AddQuery("processTagsCount(key: ProcessNoteKey!): Int!"),
		schema.AddMutation("addProcessTags(key: ProcessNoteKey!, tags: [String!]!): Boolean!"),
		schema.AddMutation("removeProcessTags(key: ProcessNoteKey!, tags: [String!]!): Boolean!"),
	)
}

// ProcessTags retrieves process tags.
func (resolver *Resolver) ProcessTags(ctx context.Context, args struct {
	Key analystnotes.ProcessNoteKey
}) ([]string, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "ProcessTags")
	if err := readIndicators(ctx); err != nil {
		return nil, err
	}
	tags, err := resolver.DeploymentDataStore.GetTagsForProcessKey(ctx, &args.Key)
	if err != nil {
		return nil, err
	}
	return tags, nil
}

// ProcessTagsCount counts process tags.
func (resolver *Resolver) ProcessTagsCount(ctx context.Context, args struct {
	Key analystnotes.ProcessNoteKey
}) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "ProcessTagsCount")
	if err := readIndicators(ctx); err != nil {
		return 0, err
	}
	tags, err := resolver.DeploymentDataStore.GetTagsForProcessKey(ctx, &args.Key)
	if err != nil {
		return 0, err
	}
	return int32(len(tags)), nil
}

// AddProcessTags adds process tags.
func (resolver *Resolver) AddProcessTags(ctx context.Context, args struct {
	Key  analystnotes.ProcessNoteKey
	Tags []string
}) (bool, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "AddProcessTags")
	if err := writeIndicators(ctx); err != nil {
		return false, err
	}
	err := resolver.DeploymentDataStore.AddTagsToProcessKey(ctx, &args.Key, args.Tags)
	if err != nil {
		return false, err
	}
	return true, nil
}

// RemoveProcessTags removes process tags.
func (resolver *Resolver) RemoveProcessTags(ctx context.Context, args struct {
	Key  analystnotes.ProcessNoteKey
	Tags []string
}) (bool, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "RemoveProcessTags")
	if err := writeIndicators(ctx); err != nil {
		return false, err
	}
	err := resolver.DeploymentDataStore.RemoveTagsFromProcessKey(ctx, &args.Key, args.Tags)
	if err != nil {
		return false, err
	}
	return true, nil
}
