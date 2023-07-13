package resolvers

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/graph-gophers/graphql-go"
	"github.com/stackrox/rox/central/audit"
	clusterMockDS "github.com/stackrox/rox/central/cluster/datastore/mocks"
	deploymentMockDS "github.com/stackrox/rox/central/deployment/datastore/mocks"
	namespaceMockDS "github.com/stackrox/rox/central/namespace/datastore/mocks"
	netPolMockDS "github.com/stackrox/rox/central/networkpolicies/datastore/mocks"
	secretMocks "github.com/stackrox/rox/central/secret/datastore/mocks"
	serviceAccountMocks "github.com/stackrox/rox/central/serviceaccount/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/grpc/authn"
	mockIdentity "github.com/stackrox/rox/pkg/grpc/authn/mocks"
	notifierMocks "github.com/stackrox/rox/pkg/notifier/mocks"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

// While it would've been nice to use the proto object, because of the oneofs and enums json unmarshalling into that object is a struggle
type serviceAcctResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Namespace   string `json:"namespace"`
	ClusterName string `json:"clusterName"`
	ClusterID   string `json:"clusterId"`
	SANamespace *struct {
		Metadata struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"metadata"`
	} `json:"saNamespace,omitempty"`
}

func TestServiceAccountResolver(t *testing.T) {
	suite.Run(t, new(ServiceAccountResolverTestSuite))
}

type ServiceAccountResolverTestSuite struct {
	mockCtrl *gomock.Controller
	suite.Suite

	deployments             *deploymentMockDS.MockDataStore
	clusterDataStore        *clusterMockDS.MockDataStore
	serviceAccountDataStore *serviceAccountMocks.MockDataStore
	namespaceDataStore      *namespaceMockDS.MockDataStore
	secretsDataStore        *secretMocks.MockDataStore
	netPolDataStore         *netPolMockDS.MockDataStore

	resolver *Resolver
	schema   *graphql.Schema
}

func (s *ServiceAccountResolverTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *ServiceAccountResolverTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())

	s.deployments = deploymentMockDS.NewMockDataStore(s.mockCtrl)
	s.secretsDataStore = secretMocks.NewMockDataStore(s.mockCtrl)
	s.netPolDataStore = netPolMockDS.NewMockDataStore(s.mockCtrl)
	s.serviceAccountDataStore = serviceAccountMocks.NewMockDataStore(s.mockCtrl)
	s.clusterDataStore = clusterMockDS.NewMockDataStore(s.mockCtrl)
	s.namespaceDataStore = namespaceMockDS.NewMockDataStore(s.mockCtrl)
	notifierMock := notifierMocks.NewMockProcessor(s.mockCtrl)

	s.deployments.EXPECT().SearchDeployments(gomock.Any(), gomock.Any()).Return([]*v1.SearchResult{}, nil).AnyTimes()

	notifierMock.EXPECT().HasEnabledAuditNotifiers().Return(false).AnyTimes()
	s.resolver = &Resolver{
		ClusterDataStore:         s.clusterDataStore,
		ServiceAccountsDataStore: s.serviceAccountDataStore,
		NamespaceDataStore:       s.namespaceDataStore,
		DeploymentDataStore:      s.deployments,
		SecretsDataStore:         s.secretsDataStore,
		NetworkPoliciesStore:     s.netPolDataStore,
		AuditLogger:              audit.New(notifierMock),
	}

	var err error
	s.schema, err = graphql.ParseSchema(Schema(), s.resolver)
	s.NoError(err)
}

func (s *ServiceAccountResolverTestSuite) TestGetServiceAccounts() {
	sa := getServiceAcct("Valid SA", "Fake cluster", "Fake NS")
	s.serviceAccountDataStore.EXPECT().SearchRawServiceAccounts(gomock.Any(), gomock.Any()).Return([]*storage.ServiceAccount{sa}, nil).AnyTimes()

	query := `
		query serviceAccounts($query: String, $pagination: Pagination) {
			results: serviceAccounts(query: $query, pagination: $pagination) {
			id
			name
			namespace
			clusterName
			clusterId
		  }
	}
`
	response := s.schema.Exec(s.getMockContext(),
		query, "serviceAccounts", map[string]interface{}{"query": ""})

	var resp struct {
		Results []serviceAcctResponse `json:"results"`
	}

	s.Len(response.Errors, 0)
	s.NoError(json.Unmarshal(response.Data, &resp))

	s.Len(resp.Results, 1)
	s.Equal(sa.GetId(), resp.Results[0].ID)
	s.Equal(sa.GetNamespace(), resp.Results[0].Namespace)
	s.Equal(sa.GetClusterId(), resp.Results[0].ClusterID)
}

func (s *ServiceAccountResolverTestSuite) TestGetSaNamespace() {
	sa := getServiceAcct("Valid SA", "Fake cluster", "Fake NS")
	s.serviceAccountDataStore.EXPECT().SearchRawServiceAccounts(gomock.Any(), gomock.Any()).Return([]*storage.ServiceAccount{sa}, nil).AnyTimes()

	// Pulled by saNamespace. It's not necessary for this test so mock away
	s.deployments.EXPECT().Count(gomock.Any(), gomock.Any()).Return(0, nil).AnyTimes()
	s.secretsDataStore.EXPECT().Count(gomock.Any(), gomock.Any()).Return(0, nil).AnyTimes()
	// saNamesapce -> namespace resolver -> CountMatchingNetworkPolicies. Yikes
	s.netPolDataStore.EXPECT().CountMatchingNetworkPolicies(gomock.Any(), gomock.Any(), gomock.Any()).Return(0, nil).AnyTimes()
	s.namespaceDataStore.EXPECT().SearchNamespaces(gomock.Any(), gomock.Any()).AnyTimes().Return([]*storage.NamespaceMetadata{{
		Id:        "namespace-id",
		Name:      "Fake NS",
		ClusterId: sa.GetClusterId(),
	}}, nil)

	query := `
		query serviceAccounts($query: String, $pagination: Pagination) {
			results: serviceAccounts(query: $query, pagination: $pagination) {
			id
			name
			namespace
			saNamespace {
			  metadata {
				id
				name
			  }
			}
			clusterName
			clusterId
		  }
	}
`
	response := s.schema.Exec(s.getMockContext(resources.Cluster, resources.Namespace),
		query, "serviceAccounts", map[string]interface{}{"query": ""})

	var resp struct {
		Results []serviceAcctResponse `json:"results"`
	}

	s.Len(response.Errors, 0)
	s.NoError(json.Unmarshal(response.Data, &resp))

	s.Len(resp.Results, 1)
	s.Equal(sa.GetId(), resp.Results[0].ID)
	s.Equal(sa.GetNamespace(), resp.Results[0].Namespace)
	s.Equal(sa.GetClusterId(), resp.Results[0].ClusterID)
	s.NotNil(resp.Results[0].SANamespace)
	s.Equal(sa.GetNamespace(), resp.Results[0].SANamespace.Metadata.Name)
}

func (s *ServiceAccountResolverTestSuite) getMockContext(extraPerms ...permissions.ResourceMetadata) context.Context {
	id := mockIdentity.NewMockIdentity(s.mockCtrl)
	id.EXPECT().UID().Return("fakeUserID").AnyTimes()
	id.EXPECT().FullName().Return("First Last").AnyTimes()
	id.EXPECT().FriendlyName().Return("DefinitelyNotBob").AnyTimes()

	extraPerms = append(extraPerms, resources.ServiceAccount)
	perms := make(map[string]storage.Access)
	resKeys := make([]permissions.ResourceHandle, 0, len(extraPerms))
	for _, p := range extraPerms {
		perms[p.String()] = storage.Access_READ_WRITE_ACCESS
		resKeys = append(resKeys, p)
	}
	id.EXPECT().Permissions().Return(perms).AnyTimes()

	ctx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedResourceLevelScopes(
			sac.AccessModeScopeKeyList(storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resKeys...)))

	return authn.ContextWithIdentity(ctx, id, s.T())
}

func getServiceAcct(name, clusterName, namespace string) *storage.ServiceAccount {
	return &storage.ServiceAccount{
		Id:          uuid.NewV4().String(),
		Name:        name,
		Namespace:   namespace,
		ClusterName: clusterName,
		ClusterId:   uuid.NewV4().String(),
	}
}
