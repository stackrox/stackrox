package authn

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/grpc/authn/mocks"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/metadata"
)

func TestExtractBearerToken(t *testing.T) {
	md := metadata.MD{
		"authorization": []string{"Bearer foobar"},
	}

	token := ExtractToken(md, "bearer")
	assert.Equal(t, "foobar", token)
}

func TestExtractInvalidType(t *testing.T) {
	md := metadata.MD{
		"authorization": []string{"token foobar"},
	}

	token := ExtractToken(md, "Bearer")
	assert.Empty(t, token)
}

func TestExtractTokenWithMultipleAuthorizationHeaders(t *testing.T) {
	md := metadata.MD{
		"authorization": []string{"Bearer foo", "Bearer bar"},
	}

	token := ExtractToken(md, "Bearer")
	assert.Empty(t, token)
}

func TestExtractTokenPrefixIsCaseInsensitive(t *testing.T) {
	md := metadata.MD{
		"authorization": []string{"BEARER foobar"},
	}

	token := ExtractToken(md, "bearer")
	assert.Equal(t, "foobar", token)
}

func TestExtractTokenWithEmptyType(t *testing.T) {
	md := metadata.MD{
		"authorization": []string{"foobar"},
	}

	token := ExtractToken(md, "")
	assert.Equal(t, "foobar", token)
}

func TestUserFromContext(t *testing.T) {
	for name, tc := range map[string]struct {
		setupIdentity func(*gomock.Controller) Identity
		expectedUser  *storage.SlimUser
	}{
		"no identity in context": {
			setupIdentity: func(_ *gomock.Controller) Identity { return nil },
			expectedUser:  nil,
		},
		"identity with full name": {
			setupIdentity: func(mockCtrl *gomock.Controller) Identity {
				id := mocks.NewMockIdentity(mockCtrl)
				id.EXPECT().UID().AnyTimes().Return("user-id")
				id.EXPECT().FullName().AnyTimes().Return("full-name")
				// FriendlyName is always evaluated by FirstNonEmpty.
				id.EXPECT().FriendlyName().AnyTimes().Return("friendly-name")
				return id
			},
			expectedUser: &storage.SlimUser{Id: "user-id", Name: "full-name"},
		},
		"identity without full name falls back to friendly name": {
			setupIdentity: func(mockCtrl *gomock.Controller) Identity {
				id := mocks.NewMockIdentity(mockCtrl)
				id.EXPECT().UID().AnyTimes().Return("user-id")
				id.EXPECT().FullName().AnyTimes().Return("")
				id.EXPECT().FriendlyName().AnyTimes().Return("friendly-name")
				return id
			},
			expectedUser: &storage.SlimUser{Id: "user-id", Name: "friendly-name"},
		},
	} {
		t.Run(name, func(it *testing.T) {
			mockCtrl := gomock.NewController(it)
			defer mockCtrl.Finish()
			ctx := context.Background()
			if id := tc.setupIdentity(mockCtrl); id != nil {
				ctx = ContextWithIdentity(ctx, id, it)
			}

			protoassert.Equal(it, tc.expectedUser, UserFromContext(ctx))
		})
	}
}
