package analystnotes

import (
	"context"

	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/grpc/authn"
	"github.com/stackrox/stackrox/pkg/stringutils"
)

// UserFromContext returns a comment_user from the given context.
func UserFromContext(ctx context.Context) *storage.Comment_User {
	var commentUser *storage.Comment_User
	identity := authn.IdentityFromContextOrNil(ctx)
	if identity != nil {
		commentUser = &storage.Comment_User{
			Id:   identity.UID(),
			Name: stringutils.FirstNonEmpty(identity.FullName(), identity.FriendlyName()),
		}
		if user := identity.User(); user != nil {
			commentUser.Email = user.Username
		}
	}
	return commentUser
}
