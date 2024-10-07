package authproviders

import (
	"context"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders/mocks"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/protomock"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func getPreviousStoredView() *storage.AuthProvider {
	updateTime := time.Date(2023, time.December, 24, 23, 59, 59, 999999999, time.UTC)
	return &storage.AuthProvider{
		Id:          authProviderID,
		LastUpdated: protocompat.ConvertTimeToTimestampOrNil(&updateTime),
	}
}

func TestDefaultAddToStore(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	providerStore := mocks.NewMockStore(mockCtrl)

	lastUpdateTime := time.Date(2024, time.July, 4, 13, 30, 30, 123456789, time.UTC)
	storageView := &storage.AuthProvider{
		Id:          authProviderID,
		LastUpdated: protocompat.ConvertTimeToTimestampOrNil(&lastUpdateTime),
	}
	provider := &providerImpl{
		storedInfo: storageView,
	}

	option := DefaultAddToStore(context.Background(), providerStore)

	providerStore.EXPECT().
		AddAuthProvider(
			gomock.Any(),
			protomock.GoMockMatcherEqualMessage(storageView),
		).
		Times(1).
		Return(nil)

	revert, err := option(provider)
	assert.NoError(t, err)

	providerStore.EXPECT().
		RemoveAuthProvider(gomock.Any(), authProviderID, true).
		Times(1).
		Return(nil)

	err = revert(provider)
	assert.NoError(t, err)
}

func TestDefaultAddToStoreBreakRevert(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	providerStore := mocks.NewMockStore(mockCtrl)

	storageView := &storage.AuthProvider{
		Id: authProviderID,
	}
	provider := &providerImpl{
		storedInfo: storageView,
	}

	option := DefaultAddToStore(context.Background(), providerStore)

	providerStore.EXPECT().
		AddAuthProvider(
			gomock.Any(),
			gomock.Any(),
		).
		Times(1).
		Return(nil)

	revert, err := option(provider)
	assert.NoError(t, err)
	assert.NotNil(t, provider.storedInfo.GetLastUpdated())

	provider.storedInfo = nil

	providerStore.EXPECT().
		RemoveAuthProvider(gomock.Any(), authProviderID, true).
		Times(1).
		Return(nil)

	err = revert(provider)
	assert.ErrorIs(t, err, noStoredInfoErrox)

}

func TestUpdateStoreWithPreviousStoredValue(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	providerStore := mocks.NewMockStore(mockCtrl)

	storedView := getPreviousStoredView()
	providerStore.EXPECT().
		GetAuthProvider(gomock.Any(), authProviderID).
		Times(1).
		Return(storedView, true, nil)

	lastUpdateTime := time.Date(2024, time.July, 4, 13, 30, 30, 123456789, time.UTC)
	storageView := &storage.AuthProvider{
		Id:          authProviderID,
		LastUpdated: protocompat.ConvertTimeToTimestampOrNil(&lastUpdateTime),
	}
	provider := &providerImpl{
		storedInfo: storageView,
	}

	option := UpdateStore(context.Background(), providerStore)

	providerStore.EXPECT().
		UpdateAuthProvider(gomock.Any(), testRecentEnoughAuthProviderWithID(authProviderID)).
		Times(1).
		Return(nil)

	revert, err := option(provider)
	assert.NoError(t, err)

	providerStore.EXPECT().
		UpdateAuthProvider(gomock.Any(), protomock.GoMockMatcherEqualMessage(storedView)).
		Times(1).
		Return(nil)

	err = revert(provider)
	assert.NoError(t, err)
}

func TestUpdateStoreWithoutPreviousStoredValue(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	providerStore := mocks.NewMockStore(mockCtrl)

	providerStore.EXPECT().
		GetAuthProvider(gomock.Any(), authProviderID).
		Times(1).
		Return(nil, false, nil)

	lastUpdateTime := time.Date(2024, time.July, 4, 13, 30, 30, 123456789, time.UTC)
	storageView := &storage.AuthProvider{
		Id:          authProviderID,
		LastUpdated: protocompat.ConvertTimeToTimestampOrNil(&lastUpdateTime),
	}
	provider := &providerImpl{
		storedInfo: storageView,
	}

	option := UpdateStore(context.Background(), providerStore)

	providerStore.EXPECT().
		UpdateAuthProvider(gomock.Any(), testRecentEnoughAuthProviderWithID(authProviderID)).
		Times(1).
		Return(nil)

	revert, err := option(provider)
	assert.NoError(t, err)

	providerStore.EXPECT().
		RemoveAuthProvider(gomock.Any(), authProviderID, true).
		Times(1).
		Return(nil)

	err = revert(provider)
	assert.NoError(t, err)
}

func TestUpdateStoreWithLookupError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	providerStore := mocks.NewMockStore(mockCtrl)

	providerStore.EXPECT().
		GetAuthProvider(gomock.Any(), authProviderID).
		Times(1).
		Return(nil, false, errors.New("DB Lookup failure"))

	lastUpdateTime := time.Date(2024, time.July, 4, 13, 30, 30, 123456789, time.UTC)
	storageView := &storage.AuthProvider{
		Id:          authProviderID,
		LastUpdated: protocompat.ConvertTimeToTimestampOrNil(&lastUpdateTime),
	}
	provider := &providerImpl{
		storedInfo: storageView,
	}

	option := UpdateStore(context.Background(), providerStore)

	providerStore.EXPECT().
		UpdateAuthProvider(gomock.Any(), testRecentEnoughAuthProviderWithID(authProviderID)).
		Times(1).
		Return(nil)

	revert, err := option(provider)
	assert.NoError(t, err)

	// On lookup error, the revert action should be a no-op.
	err = revert(provider)
	assert.NoError(t, err)
}

func TestDeletePreExistingFromStore(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	providerStore := mocks.NewMockStore(mockCtrl)

	option := DeleteFromStore(context.Background(), providerStore, authProviderID, true)

	provider := &providerImpl{
		storedInfo: &storage.AuthProvider{
			Id: authProviderID,
		},
	}

	// The option application should fetch the stored state for revert generation,
	// then call remove on the store.
	storedView := getPreviousStoredView()
	providerStore.EXPECT().
		GetAuthProvider(gomock.Any(), authProviderID).
		Times(1).
		Return(storedView, true, nil)

	providerStore.EXPECT().
		RemoveAuthProvider(gomock.Any(), authProviderID, true).
		Times(1).
		Return(nil)

	revert, err := option(provider)
	assert.NoError(t, err)

	// Revert should add back the previous stored state.
	providerStore.EXPECT().
		AddAuthProvider(gomock.Any(), protomock.GoMockMatcherEqualMessage(storedView)).
		Times(1).
		Return(nil)

	err = revert(provider)
	assert.NoError(t, err)
}

func TestDeleteMissingFromStore(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	providerStore := mocks.NewMockStore(mockCtrl)

	option := DeleteFromStore(context.Background(), providerStore, authProviderID, true)

	provider := &providerImpl{
		storedInfo: &storage.AuthProvider{
			Id: authProviderID,
		},
	}

	// The option application should fetch the stored state for revert generation,
	// then call remove on the store.
	providerStore.EXPECT().
		GetAuthProvider(gomock.Any(), authProviderID).
		Times(1).
		Return(nil, false, nil)

	providerStore.EXPECT().
		RemoveAuthProvider(gomock.Any(), authProviderID, true).
		Times(1).
		Return(nil)

	revert, err := option(provider)
	assert.NoError(t, err)

	// Revert should do nothing as there is no previously stored state.
	err = revert(provider)
	assert.NoError(t, err)
}

func TestDeleteFromStoreWithLookupError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	providerStore := mocks.NewMockStore(mockCtrl)

	option := DeleteFromStore(context.Background(), providerStore, authProviderID, true)

	provider := &providerImpl{
		storedInfo: &storage.AuthProvider{
			Id: authProviderID,
		},
	}

	// The option application should fetch the stored state for revert generation,
	// then call remove on the store.
	providerStore.EXPECT().
		GetAuthProvider(gomock.Any(), authProviderID).
		Times(1).
		Return(nil, false, errors.New("DB Lookup failure"))

	providerStore.EXPECT().
		RemoveAuthProvider(gomock.Any(), authProviderID, true).
		Times(1).
		Return(nil)

	revert, err := option(provider)
	assert.NoError(t, err)

	// Revert should be a no-op on DB lookup failure when generating the revert.
	err = revert(provider)
	assert.NoError(t, err)
}

// region test helpers

func testRecentEnoughAuthProviderWithID(id string) gomock.Matcher {
	testProvider := func(msg any) bool {
		pr := msg.(*storage.AuthProvider)
		if pr == nil {
			return false
		}
		if pr.Id != id {
			return false
		}
		now := time.Now()
		lastUpdated := pr.GetLastUpdated().AsTime()
		delta := now.Sub(lastUpdated)
		lowBound := -1 * time.Second
		upperBound := time.Second
		return delta >= lowBound && delta <= upperBound
	}
	return gomock.Cond(testProvider)
}

// endregion test helpers
