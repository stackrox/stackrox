//go:build sql_integration

package service

import (
	"context"
	"testing"

	authProviderDatastore "github.com/stackrox/rox/central/authprovider/datastore"
	groupDatastore "github.com/stackrox/rox/central/group/datastore"
	roleDatastore "github.com/stackrox/rox/central/role/datastore"
	roleMapper "github.com/stackrox/rox/central/role/mapper"
	userDatastore "github.com/stackrox/rox/central/user/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	openshiftAuth "github.com/stackrox/rox/pkg/auth/authproviders/openshift"
	authTokenMocks "github.com/stackrox/rox/pkg/auth/tokens/mocks"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestAuthProviderService(t *testing.T) {
	suite.Run(t, new(authProviderServiceTestSuite))
}

type authProviderServiceTestSuite struct {
	suite.Suite

	db *pgtest.TestPostgres

	service Service

	mockCtrl *gomock.Controller
}

func (s *authProviderServiceTestSuite) SetupSuite() {
	s.db = pgtest.ForT(s.T())

	s.mockCtrl = gomock.NewController(s.T())

	tokenIssuerFactory := authTokenMocks.NewMockIssuerFactory(s.mockCtrl)
	tokenIssuerFactory.EXPECT().CreateIssuer(gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)

	roleDS, err := roleDatastore.GetTestPostgresDataStore(s.T(), s.db)
	require.NoError(s.T(), err)
	authProviderDS := authProviderDatastore.GetTestPostgresDataStore(s.T(), s.db)
	groupDS := groupDatastore.GetTestPostgresDataStore(s.T(), s.db, roleDS, authProviderDS)
	userDS := userDatastore.GetTestDatastore(s.T())
	mapperFactory := roleMapper.NewStoreBasedMapperFactory(groupDS, roleDS, userDS)
	providerRegistry := authproviders.NewStoreBackedRegistry(
		"/sso/",
		"/auth/response/generic",
		authProviderDS,
		tokenIssuerFactory,
		mapperFactory,
	)

	ctx := sac.WithAllAccess(context.Background())
	providerRegistry.RegisterBackendFactory(ctx, openshiftAuth.TypeName, openshiftAuth.NewTestFactoryFunc(s.T()))

	s.service = New(providerRegistry, groupDS)
}

func (s *authProviderServiceTestSuite) TearDownSuite() {
	s.db.Teardown(s.T())
}

func (s *authProviderServiceTestSuite) TestPostDuplicateAuthProvider() {
	ctx := sac.WithAllAccess(context.Background())
	svc := s.service

	assert.Zero(s.T(), openshiftAuth.GetRegisteredBackendCount())

	postRequest := &v1.PostAuthProviderRequest{
		Provider: &storage.AuthProvider{
			Name:       "OpenShift",
			Type:       openshiftAuth.TypeName,
			Config:     map[string]string{},
			UiEndpoint: "central.svc",
			Enabled:    true,
			Traits:     &storage.Traits{MutabilityMode: storage.Traits_ALLOW_MUTATE},
		},
	}

	otherPostRequest := postRequest.CloneVT()
	otherPostRequest.Provider.Name = "OpenShift 2"

	// First call should succeed and register an auth provider backend
	// in the openshiftAuth certificate watch loop.
	_, err := svc.PostAuthProvider(ctx, postRequest)
	s.NoError(err)
	s.Equal(1, openshiftAuth.GetRegisteredBackendCount())

	// Second call should register a backend, hit an error when storing
	// the auth provider in DB and deregister its backend.
	_, err = svc.PostAuthProvider(ctx, postRequest)
	s.Error(err)
	s.Equal(1, openshiftAuth.GetRegisteredBackendCount())

	// Call with another provider name should succeed and register another
	// auth provider backend in the openshiftAuth certificate watch loop.
	_, err = svc.PostAuthProvider(ctx, otherPostRequest)
	s.NoError(err)
	s.Equal(2, openshiftAuth.GetRegisteredBackendCount())
}
