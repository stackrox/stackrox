package analystnotes

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/authn/mocks"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/assert"
)

func TestCommentIsDeletable(t *testing.T) {
	comment := &storage.Comment{
		User: &storage.Comment_User{Id: "1"},
	}
	identity := mocks.NewMockIdentity(gomock.NewController(t))

	// 1. Comment is not deletable with no access to any resource.
	noAccessCtx := sac.WithNoAccess(context.Background())
	assert.False(t, CommentIsDeletable(noAccessCtx, comment))

	// 2. Comment is deletable when it's the user's own comment.
	identity.EXPECT().User().Return(nil)
	identity.EXPECT().UID().Return("1")
	identity.EXPECT().FullName().Return("name")
	identity.EXPECT().FriendlyName().Return("name")

	commentOwnerCtx := authn.ContextWithIdentity(context.Background(), identity, t)
	assert.True(t, CommentIsDeletable(commentOwnerCtx, comment))

	// 3. Comment is deletable when user has access to the AllComments resource.
	identity.EXPECT().User().Return(nil)
	identity.EXPECT().UID().Return("2")
	identity.EXPECT().FullName().Return("name")
	identity.EXPECT().FriendlyName().Return("name")
	identity.EXPECT().Permissions().Return(map[string]storage.Access{
		resources.AllComments.String(): storage.Access_READ_WRITE_ACCESS,
	})

	allCommentsAccessCtx := authn.ContextWithIdentity(context.Background(), identity, t)
	assert.True(t, CommentIsDeletable(allCommentsAccessCtx, comment))

	// 4. Comment is deletable when user has access to the Administration resource.
	identity.EXPECT().User().Return(nil)
	identity.EXPECT().UID().Return("2")
	identity.EXPECT().FullName().Return("name")
	identity.EXPECT().FriendlyName().Return("name")
	identity.EXPECT().Permissions().Return(map[string]storage.Access{
		resources.Administration.String(): storage.Access_READ_WRITE_ACCESS,
	})

	administrationAccessCtx := authn.ContextWithIdentity(context.Background(), identity, t)
	assert.True(t, CommentIsDeletable(administrationAccessCtx, comment))
}
