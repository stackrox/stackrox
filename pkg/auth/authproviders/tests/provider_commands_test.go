package authproviders_test

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/auth/authproviders/mocks"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestValidateName(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	ctx := context.Background()
	store := mocks.NewMockStore(mockCtrl)
	option := authproviders.ValidateName(ctx, store)

	providerWithNoName := authproviders.GetTestProvider(t)
	_ = authproviders.WithStorageView(&storage.AuthProvider{})(providerWithNoName)
	err := option(providerWithNoName)
	assert.Error(t, err)

	const testProvider1Name = "Test Provider 1"
	testProvider1 := authproviders.GetTestProvider(t)
	_ = authproviders.WithStorageView(&storage.AuthProvider{Name: testProvider1Name})(testProvider1)
	fakeErr := errors.New("fake error")
	store.EXPECT().
		AuthProviderExistsWithName(gomock.Any(), testProvider1Name).
		Times(1).
		Return(false, fakeErr)
	err = option(testProvider1)
	assert.ErrorIs(t, err, fakeErr)

	const testProvider2Name = "Test Provider 2"
	testProvider2 := authproviders.GetTestProvider(t)
	_ = authproviders.WithStorageView(&storage.AuthProvider{Name: testProvider2Name})(testProvider2)
	store.EXPECT().
		AuthProviderExistsWithName(gomock.Any(), testProvider2Name).
		Times(1).
		Return(true, nil)
	err = option(testProvider2)
	assert.ErrorIs(t, err, errox.InvalidArgs)

	const testProvider3Name = "Test Provider 3"
	testProvider3 := authproviders.GetTestProvider(t)
	_ = authproviders.WithStorageView(&storage.AuthProvider{Name: testProvider3Name})(testProvider3)
	store.EXPECT().
		AuthProviderExistsWithName(gomock.Any(), testProvider3Name).
		Times(1).
		Return(false, nil)
	err = option(testProvider3)
	assert.NoError(t, err)
}
