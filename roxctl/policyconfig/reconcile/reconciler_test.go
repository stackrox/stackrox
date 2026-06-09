package reconcile

import (
	"context"
	"net"
	"strings"
	"testing"

	v1alpha1 "github.com/stackrox/rox/config-controller/api/v1alpha1"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/roxctl/common/environment/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

const testScope = "test-scope"

type ReconcilerTestSuite struct {
	suite.Suite
	server *mockServer
	conn   *grpc.ClientConn
	rec    *reconciler
}

func TestReconciler(t *testing.T) {
	suite.Run(t, new(ReconcilerTestSuite))
}

func (s *ReconcilerTestSuite) SetupTest() {
	s.server = &mockServer{
		policies:  make(map[string]*storage.Policy),
		notifiers: []*storage.Notifier{},
		clusters:  []*storage.Cluster{},
	}

	lis := bufconn.Listen(1024 * 1024)
	srv := grpc.NewServer()
	v1.RegisterPolicyServiceServer(srv, s.server)
	v1.RegisterNotifierServiceServer(srv, s.server)
	v1.RegisterClustersServiceServer(srv, s.server)

	go func() { _ = srv.Serve(lis) }()
	s.T().Cleanup(func() { srv.Stop() })

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(s.T(), err)
	s.conn = conn

	env, _, _ := mocks.NewEnvWithConn(conn, s.T())
	s.rec = &reconciler{
		env:         env,
		policySvc:   v1.NewPolicyServiceClient(conn),
		notifierSvc: v1.NewNotifierServiceClient(conn),
		clusterSvc:  v1.NewClustersServiceClient(conn),
		configScope: testScope,
	}
}

func (s *ReconcilerTestSuite) TestCreateNewPolicies() {
	specs := []v1alpha1.SecurityPolicySpec{
		{
			PolicyName:      "New Policy",
			Categories:      []string{"cat1"},
			LifecycleStages: []v1alpha1.LifecycleStage{"DEPLOY"},
			Severity:        "HIGH_SEVERITY",
			PolicySections: []v1alpha1.PolicySection{
				{PolicyGroups: []v1alpha1.PolicyGroup{
					{FieldName: "Process UID", Values: []v1alpha1.PolicyValue{{Value: "0"}}},
				}},
			},
		},
	}

	result, err := s.rec.reconcile(context.Background(), specs)
	require.NoError(s.T(), err)
	assert.Len(s.T(), result.created, 1)
	assert.Contains(s.T(), result.created, "New Policy")
	assert.Empty(s.T(), result.applied)
	assert.Empty(s.T(), result.deleted)

	assert.Len(s.T(), s.server.policies, 1)
	for _, p := range s.server.policies {
		assert.Equal(s.T(), storage.PolicySource_DECLARATIVE, p.GetSource())
		assert.Equal(s.T(), testScope, p.GetConfigScope())
	}
}

func (s *ReconcilerTestSuite) TestUpdateExistingPolicy() {
	s.server.policies["id-1"] = &storage.Policy{
		Id:          "id-1",
		Name:        "Existing",
		Source:      storage.PolicySource_DECLARATIVE,
		ConfigScope: testScope,
		Description: "old description",
	}

	specs := []v1alpha1.SecurityPolicySpec{
		{
			PolicyName:      "Existing",
			Description:     "new description",
			Categories:      []string{"cat1"},
			LifecycleStages: []v1alpha1.LifecycleStage{"DEPLOY"},
			Severity:        "HIGH_SEVERITY",
			PolicySections: []v1alpha1.PolicySection{
				{PolicyGroups: []v1alpha1.PolicyGroup{
					{FieldName: "Process UID", Values: []v1alpha1.PolicyValue{{Value: "0"}}},
				}},
			},
		},
	}

	result, err := s.rec.reconcile(context.Background(), specs)
	require.NoError(s.T(), err)
	assert.Empty(s.T(), result.created)
	assert.Len(s.T(), result.applied, 1)
	assert.Empty(s.T(), result.deleted)

	assert.Equal(s.T(), "new description", s.server.policies["id-1"].GetDescription())
}

func (s *ReconcilerTestSuite) TestDeleteOrphanedPolicies() {
	s.server.policies["orphan-1"] = &storage.Policy{
		Id:          "orphan-1",
		Name:        "Orphaned Policy",
		Source:      storage.PolicySource_DECLARATIVE,
		ConfigScope: testScope,
	}

	result, err := s.rec.reconcile(context.Background(), nil)
	require.NoError(s.T(), err)
	assert.Empty(s.T(), result.created)
	assert.Empty(s.T(), result.applied)
	assert.Len(s.T(), result.deleted, 1)
	assert.Contains(s.T(), result.deleted, "Orphaned Policy")

	assert.Empty(s.T(), s.server.policies)
}

func (s *ReconcilerTestSuite) TestIgnoresPoliciesFromOtherScopes() {
	s.server.policies["other-1"] = &storage.Policy{
		Id:          "other-1",
		Name:        "Other Scope Policy",
		Source:      storage.PolicySource_DECLARATIVE,
		ConfigScope: "other-scope",
	}
	s.server.policies["imperative-1"] = &storage.Policy{
		Id:     "imperative-1",
		Name:   "Manual Policy",
		Source: storage.PolicySource_IMPERATIVE,
	}
	s.server.policies["declarative-1"] = &storage.Policy{
		Id:     "declarative-1",
		Name:   "Config Controller Policy",
		Source: storage.PolicySource_DECLARATIVE,
	}

	result, err := s.rec.reconcile(context.Background(), nil)
	require.NoError(s.T(), err)
	assert.Empty(s.T(), result.deleted)

	assert.Len(s.T(), s.server.policies, 3)
}

func (s *ReconcilerTestSuite) TestDryRun() {
	s.rec.dryRun = true

	s.server.policies["id-1"] = &storage.Policy{
		Id:          "id-1",
		Name:        "Existing",
		Source:      storage.PolicySource_DECLARATIVE,
		ConfigScope: testScope,
	}
	s.server.policies["orphan-1"] = &storage.Policy{
		Id:          "orphan-1",
		Name:        "Orphaned",
		Source:      storage.PolicySource_DECLARATIVE,
		ConfigScope: testScope,
	}

	specs := []v1alpha1.SecurityPolicySpec{
		{
			PolicyName:      "Existing",
			Categories:      []string{"cat1"},
			LifecycleStages: []v1alpha1.LifecycleStage{"DEPLOY"},
			Severity:        "HIGH_SEVERITY",
			PolicySections: []v1alpha1.PolicySection{
				{PolicyGroups: []v1alpha1.PolicyGroup{
					{FieldName: "f1", Values: []v1alpha1.PolicyValue{{Value: "v1"}}},
				}},
			},
		},
		{
			PolicyName:      "Brand New",
			Categories:      []string{"cat1"},
			LifecycleStages: []v1alpha1.LifecycleStage{"DEPLOY"},
			Severity:        "LOW_SEVERITY",
			PolicySections: []v1alpha1.PolicySection{
				{PolicyGroups: []v1alpha1.PolicyGroup{
					{FieldName: "f2", Values: []v1alpha1.PolicyValue{{Value: "v2"}}},
				}},
			},
		},
	}

	result, err := s.rec.reconcile(context.Background(), specs)
	require.NoError(s.T(), err)
	assert.Len(s.T(), result.dryCreate, 1)
	assert.Len(s.T(), result.dryUpdate, 1)
	assert.Len(s.T(), result.dryDelete, 1)

	// Verify nothing was actually changed
	assert.Len(s.T(), s.server.policies, 2)
}

// mockServer implements the gRPC services needed for testing.
type mockServer struct {
	v1.UnimplementedPolicyServiceServer
	v1.UnimplementedNotifierServiceServer
	v1.UnimplementedClustersServiceServer

	policies  map[string]*storage.Policy
	notifiers []*storage.Notifier
	clusters  []*storage.Cluster
	nextID    int
}

func (m *mockServer) ListPolicies(_ context.Context, req *v1.RawQuery) (*v1.ListPoliciesResponse, error) {
	var list []*storage.ListPolicy
	for _, p := range m.policies {
		if q := req.GetQuery(); q != "" {
			// Simple mock: extract value from "Config Scope:<value>" and match exactly.
			if after, ok := strings.CutPrefix(q, "Config Scope:"); ok {
				if p.GetConfigScope() != after {
					continue
				}
			}
		}
		list = append(list, &storage.ListPolicy{
			Id:          p.GetId(),
			Name:        p.GetName(),
			Source:      p.GetSource(),
			ConfigScope: p.GetConfigScope(),
		})
	}
	return &v1.ListPoliciesResponse{Policies: list}, nil
}

func (m *mockServer) GetPolicy(_ context.Context, req *v1.ResourceByID) (*storage.Policy, error) {
	p, ok := m.policies[req.GetId()]
	if !ok {
		return nil, nil
	}
	return p, nil
}

func (m *mockServer) PostPolicy(_ context.Context, req *v1.PostPolicyRequest) (*storage.Policy, error) {
	policy := req.GetPolicy()
	for _, existing := range m.policies {
		if existing.GetName() == policy.GetName() {
			return nil, alreadyExistsError(policy.GetName())
		}
	}
	m.nextID++
	id := "generated-id-" + string(rune('0'+m.nextID))
	policy.Id = id
	m.policies[id] = policy
	return policy, nil
}

func (m *mockServer) PutPolicy(_ context.Context, req *storage.Policy) (*v1.Empty, error) {
	m.policies[req.GetId()] = req
	return &v1.Empty{}, nil
}

func (m *mockServer) DeletePolicy(_ context.Context, req *v1.ResourceByID) (*v1.Empty, error) {
	delete(m.policies, req.GetId())
	return &v1.Empty{}, nil
}

func (m *mockServer) GetNotifiers(_ context.Context, _ *v1.GetNotifiersRequest) (*v1.GetNotifiersResponse, error) {
	return &v1.GetNotifiersResponse{Notifiers: m.notifiers}, nil
}

func (m *mockServer) GetClusters(_ context.Context, _ *v1.GetClustersRequest) (*v1.ClustersList, error) {
	return &v1.ClustersList{Clusters: m.clusters}, nil
}

func alreadyExistsError(name string) error {
	return grpc.Errorf(6, "policy with name %q already exists", name) //nolint:staticcheck // codes.AlreadyExists = 6
}
