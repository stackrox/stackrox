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
