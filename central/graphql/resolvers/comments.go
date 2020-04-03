package resolvers

import (
	"context"
	"time"

	"github.com/stackrox/rox/central/analystnotes"
	"github.com/stackrox/rox/central/metrics"
	pkgMetrics "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/utils"
)

func init() {
	schema := getBuilder()
	utils.Must(
		schema.AddExtraResolver("Comment", `modifiable: Boolean!`),
		schema.AddExtraResolver("Comment", `deletable: Boolean!`),
	)
}

// Modifiable represents whether current user could modify the comment
func (resolver *commentResolver) Modifiable(ctx context.Context) (bool, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "Modifiable")

	return analystnotes.CommentIsModifiable(ctx, resolver.data), nil
}

// Deletable represents whether the current user can delete the comment.
func (resolver *commentResolver) Deletable(ctx context.Context) (bool, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "Deletable")

	return analystnotes.CommentIsDeletable(ctx, resolver.data), nil
}
