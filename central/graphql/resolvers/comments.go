package resolvers

import (
	"context"
	"time"

	"github.com/stackrox/rox/central/analystnotes"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	pkgMetrics "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	deleteNonOwnedCommentsAuthorizer = user.With(permissions.Modify(resources.AllComments))
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
	curUser := analystnotes.UserFromContext(ctx)
	return curUser.GetId() == resolver.data.GetUser().GetId() || deleteNonOwnedCommentsAuthorizer.Authorized(ctx, "graphql") == nil, nil
}
