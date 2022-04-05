package analystnotes

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/auth/role/resources"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
)

var (
	// This authorizer checks whether the user is one that can delete comments they don't own.
	deleteNonOwnedCommentsAuthorizer = user.With(permissions.Modify(resources.AllComments))
)

// CommentIsModifiable returns whether the identity in the given context can modify the given comment.
// Note that this is in addition to SAC checks, and does NOT replace them.
func CommentIsModifiable(ctx context.Context, comment *storage.Comment) bool {
	return CommentIsModifiableUser(UserFromContext(ctx), comment)
}

// CommentIsModifiableUser returns whether the given user can modify the given comment.
func CommentIsModifiableUser(user *storage.Comment_User, comment *storage.Comment) bool {
	return user.GetId() == comment.GetUser().GetId()
}

// CommentIsDeletable returns whether the identity in the given context can delete the given comment.
// Note that this is in addition to SAC checks, and does NOT replace them.
func CommentIsDeletable(ctx context.Context, comment *storage.Comment) bool {
	return UserFromContext(ctx).GetId() == comment.GetUser().GetId() || deleteNonOwnedCommentsAuthorizer.Authorized(ctx, "") == nil
}
