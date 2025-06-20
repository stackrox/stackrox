package authn

import (
	"context"
	"testing"

	"github.com/stackrox/rox/pkg/grpc/authn/mocks"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestCopyContextIdentity(t *testing.T) {
	ctrl := gomock.NewController(t)
	mid := mocks.NewMockIdentity(ctrl)
	mid.EXPECT().UID().AnyTimes().Return("username")

	original := ContextWithIdentity(context.Background(), mid, t)
	copy := CopyContextIdentity(context.Background(), original)

	id, err := IdentityFromContext(copy)
	assert.NoError(t, err)
	if assert.NotNil(t, id) {
		assert.Equal(t, "username", id.UID())
	}
}
