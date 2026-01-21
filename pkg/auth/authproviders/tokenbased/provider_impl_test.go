package tokenbased

import (
	"errors"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders/tokenbased/mocks"
	permissionsMocks "github.com/stackrox/rox/pkg/auth/permissions/mocks"
	"github.com/stackrox/rox/pkg/auth/tokens"
	tokensMocks "github.com/stackrox/rox/pkg/auth/tokens/mocks"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestBasicTokenAuthProvider(t *testing.T) {
	sourceID := "basic token source"
	sourceName := "basic token source with no option applied"
	sourceType := "basic-test"
	source := NewTokenAuthProvider(sourceID, sourceName, sourceType)
	assert.Equal(t, sourceID, source.ID())
	assert.Equal(t, sourceName, source.Name())
	assert.Equal(t, sourceType, source.Type())
	assert.True(t, source.Enabled())
	assert.True(t, source.Active())
	assert.NoError(t, source.InitFromStore(t.Context(), nil))
	ctrl := gomock.NewController(t)
	mockStore := mocks.NewMockTokenStore(ctrl)
	mockStore.EXPECT().GetTokens(gomock.Any(), gomock.Any()).Times(1).Return(nil, nil)
	assert.NoError(t, source.InitFromStore(t.Context(), mockStore))
	assert.Nil(t, source.RoleMapper())
	// source without revocation layer does not validate claims
	assert.NoError(t, source.Validate(t.Context(), nil))
	assert.NotPanics(t, func() { source.Revoke("abcd", time.Now().Add(time.Second)) })
	assert.False(t, source.IsRevoked("abcd"))
	smokeTestUnimplementedFunctions(t, source, true)
}

func TestTokenSourceWithRoleMapperOnly(t *testing.T) {
	sourceID := "role-mapper only source"
	sourceName := "token source with role mapper option"
	sourceType := "role-mapper-source"

	controller := gomock.NewController(t)
	defer controller.Finish()
	mockRoleMapper := permissionsMocks.NewMockRoleMapper(controller)

	source := NewTokenAuthProvider(sourceID, sourceName, sourceType, WithRoleMapper(mockRoleMapper))

	assert.Equal(t, sourceID, source.ID())
	assert.Equal(t, sourceName, source.Name())
	assert.Equal(t, sourceType, source.Type())
	assert.True(t, source.Enabled())
	assert.True(t, source.Active())
	assert.NoError(t, source.InitFromStore(t.Context(), nil))
	assert.Equal(t, mockRoleMapper, source.RoleMapper())
	// source without revocation layer does not validate claims
	assert.NoError(t, source.Validate(t.Context(), nil))
	assert.NotPanics(t, func() { source.Revoke("abcd", time.Now().Add(time.Second)) })
	assert.False(t, source.IsRevoked("abcd"))
	smokeTestUnimplementedFunctions(t, source, true)
}

func TestTokenSourceWithRevocationLayerOnly(t *testing.T) {
	sourceID := "token source with revocation layer"
	sourceName := "token source with revocation layer option applied"
	sourceType := "revocation-test"

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockRevocationLayer := tokensMocks.NewMockRevocationLayer(ctrl)

	source := NewTokenAuthProvider(sourceID, sourceName, sourceType, WithRevocationLayer(mockRevocationLayer))
	assert.Equal(t, sourceID, source.ID())
	assert.Equal(t, sourceName, source.Name())
	assert.Equal(t, sourceType, source.Type())
	assert.True(t, source.Enabled())
	assert.True(t, source.Active())
	assert.Nil(t, source.RoleMapper())
	smokeTestUnimplementedFunctions(t, source, true)
	// Ensure InitFromStore does not feed the revocation layer when there are no tokens
	mockStore := mocks.NewMockTokenStore(ctrl)
	mockStore.EXPECT().GetTokens(gomock.Any(), gomock.Any()).Times(1).Return(nil, nil)
	assert.NoError(t, source.InitFromStore(t.Context(), mockStore))
	// Ensure InitFromStore propagates errors from the token store without calling the revocation layer
	expiration1 := time.Date(2025, time.July, 4, 22, 23, 24, 123456789, time.UTC)
	sampleToken1 := &storage.TokenMetadata{
		Id:         uuid.NewTestUUID(1).String(),
		Expiration: protoconv.ConvertTimeToTimestampOrNil(expiration1),
	}
	testErr := errors.New("test error")
	mockStore.EXPECT().GetTokens(gomock.Any(), gomock.Any()).Times(1).Return([]*storage.TokenMetadata{sampleToken1}, testErr)
	assert.ErrorIs(t, source.InitFromStore(t.Context(), mockStore), testErr)
	// Ensure InitFromStore feeds the revocation layer with data from the token store
	expiration2 := time.Date(2025, time.July, 14, 21, 22, 23, 987654321, time.UTC)
	tokenID2 := uuid.NewTestUUID(2).String()
	sampleToken2 := &storage.TokenMetadata{
		Id:         tokenID2,
		Expiration: protoconv.ConvertTimeToTimestampOrNil(expiration2),
	}
	mockStore.EXPECT().GetTokens(gomock.Any(), gomock.Any()).Times(1).Return([]*storage.TokenMetadata{sampleToken2}, nil)
	mockRevocationLayer.EXPECT().Revoke(tokenID2, expiration2).Times(1)
	assert.NoError(t, source.InitFromStore(t.Context(), mockStore))

	// Ensure validation is delegated to the revocation layer
	testClaims1 := &tokens.Claims{
		RoxClaims: tokens.RoxClaims{
			Name: "test claims 1",
		},
	}
	mockRevocationLayer.EXPECT().Validate(gomock.Any(), testClaims1).Times(1).Return(nil)
	assert.NoError(t, source.Validate(t.Context(), testClaims1))
	testClaims2 := &tokens.Claims{
		RoxClaims: tokens.RoxClaims{},
	}
	mockRevocationLayer.EXPECT().Validate(gomock.Any(), testClaims2).Times(1).Return(testErr)
	assert.ErrorIs(t, source.Validate(t.Context(), testClaims2), testErr)
	// ensure revocation is delegated to the revocation layer
	revokedTokenID := "abcd"
	expiration := time.Now().Add(time.Minute)
	mockRevocationLayer.EXPECT().Revoke(revokedTokenID, expiration).Times(1)
	assert.NotPanics(t, func() { source.Revoke(revokedTokenID, expiration) })
	// Ensure IsREvoked delegates the decision to the revocation layer
	mockRevocationLayer.EXPECT().IsRevoked(revokedTokenID).Times(1).Return(true)
	mockRevocationLayer.EXPECT().IsRevoked("efgh").Times(1).Return(false)
	assert.True(t, source.IsRevoked(revokedTokenID))
	assert.False(t, source.IsRevoked("efgh"))
}

// There is currently token source that relies on a combination of
// revocation layer and role mapper. The combination will therefore
// not be tested.
