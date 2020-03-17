package resolvers

import (
	"context"
	"time"

	"github.com/stackrox/rox/central/comments"
	"github.com/stackrox/rox/central/metrics"
	pkgMetrics "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/utils"
)

func init() {
	schema := getBuilder()
	utils.Must(
		schema.AddExtraResolver("Comment", `modifiable: Boolean!`),
	)
}

// Modifiable represents whether current user could modify the comment
func (resolver *commentResolver) Modifiable(ctx context.Context) (bool, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "Modifiable")

	// TODO: update this after the access control changes go in.
	user := comments.UserFromContext(ctx)
	return user.GetId() == resolver.data.GetUser().GetId(), nil
}
