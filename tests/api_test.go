package tests

import (
	"context"
	"testing"
	"time"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/image/policies"
	"bitbucket.org/stack-rox/apollo/pkg/clientconn"
	"bitbucket.org/stack-rox/apollo/pkg/defaults"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestPing(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := clientconn.UnauthenticatedGRPCConnection(apiEndpoint)
	require.NoError(t, err)

	service := v1.NewPingServiceClient(conn)
	_, err = service.Ping(ctx, &empty.Empty{})
	assert.NoError(t, err)
}

func TestDefaultPolicies(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	conn, err := clientconn.UnauthenticatedGRPCConnection(apiEndpoint)
	require.NoError(t, err)

	defaults.PoliciesPath = policies.Directory()
	defaultPolicies, err := defaults.Policies()
	require.NoError(t, err)

	service := v1.NewPolicyServiceClient(conn)
	resp, err := service.GetPolicies(ctx, &v1.GetPoliciesRequest{})
	require.NoError(t, err)

	policiesMap := make(map[string]*v1.Policy)
	for _, p := range resp.GetPolicies() {
		p.Id = ""
		policiesMap[p.GetName()] = p
	}

	assert.Equal(t, len(defaultPolicies), len(resp.GetPolicies()))

	for _, p := range defaultPolicies {
		assert.Equal(t, p, policiesMap[p.GetName()])
	}
}

func TestPoliciesCRUD(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	policy := &v1.Policy{
		Name:        "test policy " + time.Now().String(),
		Description: "description",
		Severity:    v1.Severity_HIGH_SEVERITY,
		Categories:  []v1.Policy_Category{v1.Policy_Category_IMAGE_ASSURANCE, v1.Policy_Category_PRIVILEGES_CAPABILITIES},
		Disabled:    false,
		ImagePolicy: &v1.ImagePolicy{
			ImageName: &v1.ImageNamePolicy{
				Tag: "latest",
			},
		},
		PrivilegePolicy: &v1.PrivilegePolicy{
			SetPrivileged: &v1.PrivilegePolicy_Privileged{
				Privileged: true,
			},
		},
	}

	conn, err := clientconn.UnauthenticatedGRPCConnection(apiEndpoint)
	require.NoError(t, err)

	service := v1.NewPolicyServiceClient(conn)

	// Create
	postResp, err := service.PostPolicy(ctx, policy)
	require.NoError(t, err)

	policy.Id = postResp.GetId()
	assert.Equal(t, policy, postResp)

	// Read
	getResp, err := service.GetPolicy(ctx, &v1.ResourceByID{Id: policy.GetId()})
	require.NoError(t, err)
	assert.Equal(t, policy, getResp)

	getManyResp, err := service.GetPolicies(ctx, &v1.GetPoliciesRequest{Name: []string{policy.GetName()}})
	require.NoError(t, err)
	assert.Equal(t, 1, len(getManyResp.GetPolicies()))
	if len(getManyResp.GetPolicies()) > 0 {
		assert.Equal(t, policy, getManyResp.GetPolicies()[0])
	}

	// Update
	policy.Severity = v1.Severity_LOW_SEVERITY
	policy.Description = "updated description"
	policy.Disabled = true
	policy.ImagePolicy.SetScanAgeDays = &v1.ImagePolicy_ScanAgeDays{ScanAgeDays: 10}
	policy.PrivilegePolicy.AddCapabilities = []string{"CAP_SYS_MODULE"}

	_, err = service.PutPolicy(ctx, policy)
	require.NoError(t, err)

	getResp, err = service.GetPolicy(ctx, &v1.ResourceByID{Id: policy.GetId()})
	require.NoError(t, err)
	assert.Equal(t, policy, getResp)

	// Delete
	_, err = service.DeletePolicy(ctx, &v1.ResourceByID{Id: policy.GetId()})
	require.NoError(t, err)

	_, err = service.GetPolicy(ctx, &v1.ResourceByID{Id: policy.GetId()})
	s, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.NotFound, s.Code())
}
