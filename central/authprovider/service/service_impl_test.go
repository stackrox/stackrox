package service

import (
	"context"
	"testing"

	groupStoreMocks "github.com/stackrox/rox/central/group/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	authProviderMocks "github.com/stackrox/rox/pkg/auth/authproviders/mocks"
	permissionsMocks "github.com/stackrox/rox/pkg/auth/permissions/mocks"
	authTokenMocks "github.com/stackrox/rox/pkg/auth/tokens/mocks"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

const (
	testProviderType = "test"

	urlPathPrefix = "/sso/"
	redirectURL   = "/auth/response/generic"
)

func TestMockedAuthProviderService(t *testing.T) {
	suite.Run(t, new(mockedAuthProviderServiceTestSuite))
}

type mockedAuthProviderServiceTestSuite struct {
	suite.Suite

	mockCtrl *gomock.Controller

	providerMockBEFactory *authProviderMocks.MockBackendFactory
	providerMockStore     *authProviderMocks.MockStore

	service Service
}

func (s *mockedAuthProviderServiceTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())

	tokenIssuerFactory := authTokenMocks.NewMockIssuerFactory(s.mockCtrl)
	tokenIssuerFactory.EXPECT().CreateIssuer(gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)

	s.providerMockStore = authProviderMocks.NewMockStore(s.mockCtrl)

	mockRoleMapper := permissionsMocks.NewMockRoleMapper(s.mockCtrl)

	mapperFactory := permissionsMocks.NewMockRoleMapperFactory(s.mockCtrl)
	mapperFactory.EXPECT().GetRoleMapper(gomock.Any()).AnyTimes().Return(mockRoleMapper)

	registry := authproviders.NewStoreBackedRegistry(
		urlPathPrefix,
		redirectURL,
		s.providerMockStore,
		tokenIssuerFactory,
		mapperFactory,
	)

	s.providerMockBEFactory = authProviderMocks.NewMockBackendFactory(s.mockCtrl)

	backendFactoryCreator := func(_ string) authproviders.BackendFactory {
		return s.providerMockBEFactory
	}

	err := registry.RegisterBackendFactory(
		context.Background(),
		testProviderType,
		backendFactoryCreator,
	)
	s.Require().NoError(err)

	groupStore := groupStoreMocks.NewMockDataStore(s.mockCtrl)
	s.service = New(registry, groupStore)
}

func (s *mockedAuthProviderServiceTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *mockedAuthProviderServiceTestSuite) TestPostDuplicateAuthProvider() {
	traits := &storage.Traits{}
	traits.SetMutabilityMode(storage.Traits_ALLOW_MUTATE)
	ap := &storage.AuthProvider{}
	ap.SetName("Test Provider")
	ap.SetType(testProviderType)
	ap.SetConfig(map[string]string{})
	ap.SetUiEndpoint("central.svc")
	ap.SetEnabled(true)
	ap.SetTraits(traits)
	postRequest := &v1.PostAuthProviderRequest{}
	postRequest.SetProvider(ap)

	otherPostRequest := postRequest.CloneVT()
	otherPostRequest.GetProvider().SetName("Test Provider 2")

	ctx := context.Background()

	s.expectWorkingPostAuthProvider()
	_, err := s.service.PostAuthProvider(ctx, postRequest)
	s.NoError(err)

	s.expectWorkingPostAuthProvider()
	_, err = s.service.PostAuthProvider(ctx, otherPostRequest)
	s.NoError(err)

	// The AuthProvider creation flow should fail on the duplicate name check
	// and not proceed with the provider creation (hence no backend creation,
	// no store addition and no provider config redaction),
	s.providerMockStore.EXPECT().
		AuthProviderExistsWithName(gomock.Any(), gomock.Any()).
		Times(1).
		Return(true, nil)
	_, err = s.service.PostAuthProvider(ctx, postRequest)
	s.ErrorIs(err, errox.InvalidArgs)
}

func (s *mockedAuthProviderServiceTestSuite) expectWorkingPostAuthProvider() {
	s.providerMockStore.EXPECT().
		AuthProviderExistsWithName(gomock.Any(), gomock.Any()).
		Times(1).
		Return(false, nil)
	s.expectBackendCreation()
	s.expectProviderAdditionToStore()
	s.providerMockBEFactory.EXPECT().
		RedactConfig(gomock.Any()).
		Times(1).
		Return(map[string]string{})
}

func (s *mockedAuthProviderServiceTestSuite) expectBackendCreation() {
	mockBackend := authProviderMocks.NewMockBackend(s.mockCtrl)
	mockBackend.EXPECT().Config().AnyTimes().Return(map[string]string{})
	mockBackend.EXPECT().OnEnable(gomock.Any()).AnyTimes()

	s.providerMockBEFactory.EXPECT().
		CreateBackend(
			gomock.Any(), // ctx
			gomock.Any(), // id
			gomock.Any(), // uiEndpoints
			gomock.Any(), // config
			gomock.Any(), // mappings
		).Times(1).Return(mockBackend, nil)
}

func (s *mockedAuthProviderServiceTestSuite) expectProviderAdditionToStore() {
	s.providerMockStore.EXPECT().
		AddAuthProvider(gomock.Any(), gomock.Any()).
		Times(1).
		Return(nil)
}
