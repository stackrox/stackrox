package authn

import (
	"context"
	"errors"
	"testing"

	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authn/mocks"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

var errTest = errors.New("dummy test error")

func TestCopyContextIdentity(t *testing.T) {
	ctrl := gomock.NewController(t)
	mid := mocks.NewMockIdentity(ctrl)
	mid.EXPECT().UID().AnyTimes().Return("username")

	original := ContextWithIdentity(context.Background(), mid, t)

	type testKey string
	original = context.WithValue(original, testKey("key"), "value")
	original, cancelOriginal := context.WithCancel(original)

	copy := CopyContextIdentity(context.Background(), original)

	id, err := IdentityFromContext(copy)
	assert.NoError(t, err)
	if assert.NotNil(t, id) {
		assert.Equal(t, "username", id.UID())
	}

	cancelOriginal()
	assert.NotNil(t, original.Value(testKey("key")))
	assert.Nil(t, copy.Value(testKey("key")))
	assert.NotNil(t, original.Err())
	assert.Nil(t, copy.Err())
}

func TestIdentityFromContext(t *testing.T) {
	t.Run("no identity in context yields NoCredentials", func(it *testing.T) {
		id, err := IdentityFromContext(context.Background())
		assert.Nil(it, id)
		assert.ErrorIs(it, err, errox.NoCredentials)
	})

	t.Run("identity error in context takes precedence", func(it *testing.T) {
		ctx := context.WithValue(context.Background(), identityErrorContextKey{}, errTest)
		id, err := IdentityFromContext(ctx)
		assert.Nil(it, id)
		assert.ErrorIs(it, err, errTest)
	})

	t.Run("identity in context is returned", func(it *testing.T) {
		mockCtrl := gomock.NewController(it)
		defer mockCtrl.Finish()
		mid := mocks.NewMockIdentity(mockCtrl)
		mid.EXPECT().UID().AnyTimes().Return("username")
		ctx := ContextWithIdentity(context.Background(), mid, it)

		id, err := IdentityFromContext(ctx)
		assert.NoError(it, err)
		if assert.NotNil(it, id) {
			assert.Equal(it, "username", id.UID())
		}
	})
}

func TestIdentityFromContextOrNil(t *testing.T) {
	assert.Nil(t, IdentityFromContextOrNil(context.Background()))

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mid := mocks.NewMockIdentity(mockCtrl)
	mid.EXPECT().UID().AnyTimes().Return("username")
	ctx := ContextWithIdentity(context.Background(), mid, t)

	id := IdentityFromContextOrNil(ctx)
	if assert.NotNil(t, id) {
		assert.Equal(t, "username", id.UID())
	}
}
