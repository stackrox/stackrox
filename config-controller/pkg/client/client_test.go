package client

import (
	"context"
	"embed"
	"testing"
	"time"

	"github.com/stackrox/rox/config-controller/pkg/client/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/encoding/protojson"
)

//go:embed test-policy-1.json test-policy-2.json
var AssetFS embed.FS

func loadPolicies() []*storage.Policy {
	policies := make([]*storage.Policy, 2)
	unmashaller := protojson.UnmarshalOptions{}

	for i, name := range []string{"test-policy-1.json", "test-policy-2.json"} {
		bytes, err := AssetFS.ReadFile(name)
		if err != nil {
			panic(err)
		}

		policyProto := &storage.Policy{}
		if err = unmashaller.Unmarshal(bytes, policyProto); err != nil {
			panic(err)
		}

		policies[i] = policyProto
	}
	return policies
}

type applyMockPolicyClient struct {
	mockClient PolicyClient
}

func (a *applyMockPolicyClient) Apply(c CachedPolicyClient) {
	cl := c.(*client)
	cl.svc = a.mockClient
}

func createListPolicies(policies []*storage.Policy) []*storage.ListPolicy {
	ret := make([]*storage.ListPolicy, len(policies))

	for i, policy := range policies {
		lp := storage.ListPolicy{Id: policy.Id}
		ret[i] = &lp
	}
	return ret
}

type clientTest struct {
	ctx        context.Context
	policies   []*storage.Policy
	client     CachedPolicyClient
	controller *gomock.Controller
	mockClient *mocks.MockPolicyClient
}

// setUp loads the policies from embed.FS, creates the mock and cached client
// Every test in this file calls setUp
func setUp(t *testing.T, fn func(*mocks.MockPolicyClient, []*storage.Policy)) clientTest {
	policies := loadPolicies()

	controller := gomock.NewController(t)
	mockClient := mocks.NewMockPolicyClient(controller)
	o := applyMockPolicyClient{
		mockClient: mockClient,
	}

	fn(mockClient, policies)
	ctx := context.Background()
	client, err := New(ctx, &o)
	assert.NoError(t, err, "Unexpected error creating CachedPolicyClient")
	return clientTest{
		ctx:        ctx,
		policies:   policies,
		client:     client,
		controller: controller,
		mockClient: mockClient,
	}
}

// TestCachedClientList validates that the cached client lists policies as expected
func TestCachedClientList(t *testing.T) {

	clientTest := setUp(t, func(mockClient *mocks.MockPolicyClient, policies []*storage.Policy) {
		mockClient.EXPECT().ListPolicies(gomock.Any()).Return(createListPolicies(policies), nil).Times(1)
		mockClient.EXPECT().GetPolicy(gomock.Any(), policies[0].Id).Return(policies[0], nil).Times(1)
		mockClient.EXPECT().GetPolicy(gomock.Any(), policies[1].Id).Return(policies[1], nil).Times(1)
		mockClient.EXPECT().TokenExchange(gomock.Any()).Return(nil).Times(1)
	})
	defer clientTest.controller.Finish()

	returnedPolicies, err := clientTest.client.ListPolicies(clientTest.ctx)

	assert.NoError(t, err, "Unexpected error listing policies")
	assert.Equal(t, 2, len(returnedPolicies), "Wrong size of returned policy list")
	protoassert.ElementsMatch(t, clientTest.policies, returnedPolicies)
}

// TestCachedClientGet validates that the cached client fetches policies as expected
func TestCachedClientGet(t *testing.T) {
	clientTest := setUp(t, func(mockClient *mocks.MockPolicyClient, policies []*storage.Policy) {
		mockClient.EXPECT().ListPolicies(gomock.Any()).Return(createListPolicies(policies), nil).Times(1)
		mockClient.EXPECT().GetPolicy(gomock.Any(), policies[0].Id).Return(policies[0], nil).Times(1)
		mockClient.EXPECT().GetPolicy(gomock.Any(), policies[1].Id).Return(policies[1], nil).Times(1)
		mockClient.EXPECT().TokenExchange(gomock.Any()).Return(nil).Times(1)
	})
	defer clientTest.controller.Finish()

	returnedPolicy, exists, err := clientTest.client.GetPolicy(clientTest.ctx, clientTest.policies[0].Name)

	assert.NoError(t, err, "Unexpected error GETting a policy")
	assert.True(t, exists, "Policy doesn't exist when it should")
	assert.Equal(t, clientTest.policies[0].Id, returnedPolicy.Id)

	_, exists, err = clientTest.client.GetPolicy(clientTest.ctx, "Policy name that doesn't exist")

	assert.NoError(t, err, "Unexpected error GETting a policy")
	assert.False(t, exists, "Policy exists when it should not")
}

// TestCachedClientCreate validates that the cached client creates policies as expected
func TestCachedClientCreate(t *testing.T) {
	newPolicy := storage.Policy{
		Name:            "This is a new policy",
		Description:     "A really good description",
		LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_BUILD},
		Severity:        storage.Severity_CRITICAL_SEVERITY,
		PolicySections: []*storage.PolicySection{{
			SectionName: "Section A",
			PolicyGroups: []*storage.PolicyGroup{{
				FieldName: "Image",
				Values: []*storage.PolicyValue{{
					Value: "hello",
				}},
			}},
		}},
	}

	mockPolicyToReturn := newPolicy.CloneVT()
	mockPolicyToReturn.Id = "abc123"

	clientTest := setUp(t, func(mockClient *mocks.MockPolicyClient, policies []*storage.Policy) {
		mockClient.EXPECT().ListPolicies(gomock.Any()).Return(createListPolicies(policies), nil).Times(1)
		mockClient.EXPECT().GetPolicy(gomock.Any(), policies[0].Id).Return(policies[0], nil).Times(1)
		mockClient.EXPECT().GetPolicy(gomock.Any(), policies[1].Id).Return(policies[1], nil).Times(1)
		mockClient.EXPECT().PostPolicy(gomock.Any(), &newPolicy).Return(mockPolicyToReturn, nil).Times(1)
		mockClient.EXPECT().TokenExchange(gomock.Any()).Return(nil).Times(1)
	})
	defer clientTest.controller.Finish()

	createdPolicy, err := clientTest.client.CreatePolicy(clientTest.ctx, &newPolicy)

	assert.NoError(t, err, "Unexpected error creating a policy")
	assert.Equal(t, "abc123", createdPolicy.Id)
}

// TestCachedClientUpdate validates that the cached client updates policies as expected
func TestCachedClientUpdate(t *testing.T) {
	clientTest := setUp(t, func(mockClient *mocks.MockPolicyClient, policies []*storage.Policy) {
		mockClient.EXPECT().ListPolicies(gomock.Any()).Return(createListPolicies(policies), nil).Times(1)
		mockClient.EXPECT().GetPolicy(gomock.Any(), policies[0].Id).Return(policies[0], nil).Times(1)
		mockClient.EXPECT().GetPolicy(gomock.Any(), policies[1].Id).Return(policies[1], nil).Times(1)
		mockClient.EXPECT().TokenExchange(gomock.Any()).Return(nil).Times(1)
	})
	defer clientTest.controller.Finish()

	policyToUpdate := clientTest.policies[0]
	policyToUpdate.Description = "Update this description"
	clientTest.mockClient.EXPECT().PutPolicy(gomock.Any(), policyToUpdate).Return(nil).Times(1)

	err := clientTest.client.UpdatePolicy(clientTest.ctx, policyToUpdate)

	assert.NoError(t, err, "Unexpected error updating a policy")
}

// TestCachedClientFlushCacheNoUpdate validates that the cached client flushes the cache as expected
// In this test, FlushCache is a no-op since the last updated timestamp is too recent.
func TestCachedClientFlushCacheNoUpdate(t *testing.T) {
	clientTest := setUp(t, func(mockClient *mocks.MockPolicyClient, policies []*storage.Policy) {
		mockClient.EXPECT().ListPolicies(gomock.Any()).Return(createListPolicies(policies), nil).Times(1)
		mockClient.EXPECT().GetPolicy(gomock.Any(), policies[0].Id).Return(policies[0], nil).Times(1)
		mockClient.EXPECT().GetPolicy(gomock.Any(), policies[1].Id).Return(policies[1], nil).Times(1)
		mockClient.EXPECT().TokenExchange(gomock.Any()).Return(nil).Times(1)
	})
	defer clientTest.controller.Finish()

	err := clientTest.client.FlushCache(clientTest.ctx)

	assert.NoError(t, err, "Unexpected error flushing cache")
}

// TestCachedClientFlushCacheWithUpdate validates that the cached client flushes the cache as expected
// In this test, the last updated timestamp is "hacked" to make it appear older so as to trigger a real flush.
func TestCachedClientFlushCacheWithUpdate(t *testing.T) {
	clientTest := setUp(t, func(mockClient *mocks.MockPolicyClient, policies []*storage.Policy) {
		mockClient.EXPECT().ListPolicies(gomock.Any()).Return(createListPolicies(policies), nil).Times(2)
		mockClient.EXPECT().GetPolicy(gomock.Any(), policies[0].Id).Return(policies[0], nil).Times(2)
		mockClient.EXPECT().GetPolicy(gomock.Any(), policies[1].Id).Return(policies[1], nil).Times(2)
		mockClient.EXPECT().TokenExchange(gomock.Any()).Return(nil).Times(1)
	})
	defer clientTest.controller.Finish()

	clientImpl := clientTest.client.(*client)
	clientImpl.lastUpdated = time.Now().Add(time.Second * -11)

	err := clientTest.client.FlushCache(clientTest.ctx)

	assert.NoError(t, err, "Unexpected error flushing cache")
}

// TestCachedClientEnsureFreshNoUpdate validates that EnsureFresh works as expected
// In this test, EnsureFresh is a no-op since the last updated timestamp is too recent.
func TestCachedClientEnsureFreshNoUpdate(t *testing.T) {
	clientTest := setUp(t, func(mockClient *mocks.MockPolicyClient, policies []*storage.Policy) {
		mockClient.EXPECT().ListPolicies(gomock.Any()).Return(createListPolicies(policies), nil).Times(1)
		mockClient.EXPECT().GetPolicy(gomock.Any(), policies[0].Id).Return(policies[0], nil).Times(1)
		mockClient.EXPECT().GetPolicy(gomock.Any(), policies[1].Id).Return(policies[1], nil).Times(1)
		mockClient.EXPECT().TokenExchange(gomock.Any()).Return(nil).Times(2)
	})
	defer clientTest.controller.Finish()

	err := clientTest.client.EnsureFresh(clientTest.ctx)

	assert.NoError(t, err, "Unexpected error flushing cache")
}

// TestCachedClientEnsureFreshWithUpdate validates that EnsureFresh works as expected
// In this test, the last updated timestamp is "hacked" to make it appear older so as to trigger a real flush.
func TestCachedClientEnsureFreshWithUpdate(t *testing.T) {
	clientTest := setUp(t, func(mockClient *mocks.MockPolicyClient, policies []*storage.Policy) {
		mockClient.EXPECT().ListPolicies(gomock.Any()).Return(createListPolicies(policies), nil).Times(2)
		mockClient.EXPECT().GetPolicy(gomock.Any(), policies[0].Id).Return(policies[0], nil).Times(2)
		mockClient.EXPECT().GetPolicy(gomock.Any(), policies[1].Id).Return(policies[1], nil).Times(2)
		mockClient.EXPECT().TokenExchange(gomock.Any()).Return(nil).Times(2)
	})
	defer clientTest.controller.Finish()

	clientImpl := clientTest.client.(*client)
	clientImpl.lastUpdated = time.Now().Add(time.Minute * -6)

	err := clientTest.client.EnsureFresh(clientTest.ctx)

	assert.NoError(t, err, "Unexpected error flushing cache")
}
