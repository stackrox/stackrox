//go:build sql_integration

package service

import (
	"context"
	"testing"

	authProviderDataStore "github.com/stackrox/rox/central/authprovider/datastore"
	groupDataStore "github.com/stackrox/rox/central/group/datastore"
	roleDataStore "github.com/stackrox/rox/central/role/datastore"
	roleMapper "github.com/stackrox/rox/central/role/mapper"
	userDataStore "github.com/stackrox/rox/central/user/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	openshiftAuth "github.com/stackrox/rox/pkg/auth/authproviders/openshift"
	authTokenMocks "github.com/stackrox/rox/pkg/auth/tokens/mocks"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestAuthProviderService(t *testing.T) {
	suite.Run(t, new(authProviderServiceTestSuite))
}

type authProviderServiceTestSuite struct {
	suite.Suite

	db       *pgtest.TestPostgres
	mockCtrl *gomock.Controller

	service Service
}

func (s *authProviderServiceTestSuite) SetupSuite() {
	t := s.T()
	db := pgtest.ForT(t)
	s.db = db
	s.mockCtrl = gomock.NewController(t)

	tokenIssuerFactory := authTokenMocks.NewMockIssuerFactory(s.mockCtrl)
	tokenIssuerFactory.EXPECT().CreateIssuer(gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)

	roleDS := roleDataStore.GetTestPostgresDataStore(t, db)
	authProviderDS := authProviderDataStore.GetTestPostgresDataStore(t, db)
	groupDS := groupDataStore.GetTestPostgresDataStore(t, db, roleDS, authProviderDS)
	userDS := userDataStore.GetTestDataStore(t)
	mapperFactory := roleMapper.NewStoreBasedMapperFactory(groupDS, roleDS, userDS)
	providerRegistry := authproviders.NewStoreBackedRegistry(
		urlPathPrefix,
		redirectURL,
		authProviderDS,
		tokenIssuerFactory,
		mapperFactory,
	)

	ctx := sac.WithAllAccess(context.Background())
	err := providerRegistry.RegisterBackendFactory(
		ctx,
		openshiftAuth.TypeName,
		openshiftAuth.NewTestFactoryCreator(t),
	)
	s.Require().NoError(err)

	s.service = New(providerRegistry, groupDS)
}

func (s *authProviderServiceTestSuite) TearDownSuite() {
	s.mockCtrl.Finish()
}

func (s *authProviderServiceTestSuite) TestPostDuplicateAuthProvider() {
	ctx := sac.WithAllAccess(context.Background())

	s.Zero(openshiftAuth.GetRegisteredBackendCount())

	postRequest := &v1.PostAuthProviderRequest{
		Provider: &storage.AuthProvider{
			Name:       "OpenShift",
			Type:       openshiftAuth.TypeName,
			Config:     map[string]string{},
			UiEndpoint: "central.svc",
			Enabled:    true,
			Traits: &storage.Traits{
				MutabilityMode: storage.Traits_ALLOW_MUTATE,
			},
		},
	}

	otherPostRequest := postRequest.CloneVT()
	otherPostRequest.Provider.Name = "OpenShift 2"

	// First call should succeed and register an auth provider backend
	// in the openshiftAuth certificate watch loop.
	_, err := s.service.PostAuthProvider(ctx, postRequest)
	s.NoError(err)
	s.Equal(1, openshiftAuth.GetRegisteredBackendCount())

	// First call with another auth provider name should succeed
	// and register another auth provider backend in the openshiftAuth
	// certificate watch loop.
	_, err = s.service.PostAuthProvider(ctx, otherPostRequest)
	s.NoError(err)
	s.Equal(2, openshiftAuth.GetRegisteredBackendCount())

	// Second call with an already registered auth provider should
	// fail the duplicate name check and not be added
	// to the openshiftAuth certificate watch loop.
	_, err = s.service.PostAuthProvider(ctx, postRequest)
	s.Error(err)
	s.Equal(2, openshiftAuth.GetRegisteredBackendCount())
}
