package comments

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/stringutils"
)

// UserFromContext returns a comment_user from the given context.
func UserFromContext(ctx context.Context) *storage.Comment_User {
	var commentUser *storage.Comment_User
	identity := authn.IdentityFromContext(ctx)
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
