package service

import (
	"context"
	"errors"
	"testing"
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	durationpb "google.golang.org/protobuf/types/known/durationpb"
)

// mockBackend is a mock implementation of the Backend interface for testing.
type mockBackend struct {
	issueEphemeralFunc func(ctx context.Context, name string, roleNames []string, dynamicScope *storage.DynamicAccessScope, ttl time.Duration) (string, *time.Time, error)
}

func (m *mockBackend) GetTokenOrNil(ctx context.Context, tokenID string) (*storage.TokenMetadata, error) {
	return nil, errors.New("not implemented")
}

func (m *mockBackend) GetTokens(ctx context.Context, req *v1.GetAPITokensRequest) ([]*storage.TokenMetadata, error) {
	return nil, errors.New("not implemented")
}

func (m *mockBackend) IssueRoleToken(ctx context.Context, name string, roleNames []string, expireAt *time.Time) (string, *storage.TokenMetadata, error) {
	return "", nil, errors.New("not implemented")
}

func (m *mockBackend) IssueEphemeralScopedToken(ctx context.Context, name string, roleNames []string, dynamicScope *storage.DynamicAccessScope, ttl time.Duration) (string, *time.Time, error) {
	if m.issueEphemeralFunc != nil {
		return m.issueEphemeralFunc(ctx, name, roleNames, dynamicScope, ttl)
	}
	expiresAt := time.Now().Add(ttl)
	return "mock-token-abc123", &expiresAt, nil
}

func (m *mockBackend) RevokeToken(ctx context.Context, tokenID string) (bool, error) {
	return false, errors.New("not implemented")
}

func TestServiceImpl_IssueToken_Success(t *testing.T) {
	ctx := context.Background()
	backend := &mockBackend{}

	service := &serviceImpl{
		tokenBackend: backend,
	}

	req := &v1.IssueScopedTokenRequest{
		UserIdentifier: "test-user@example.com",
		ClusterName:    "production-cluster",
		Namespace:      "default",
		Deployment:     "nginx",
		Ttl:            durationpb.New(3 * time.Minute),
	}

	resp, err := service.IssueToken(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "mock-token-abc123", resp.GetToken())
	assert.NotNil(t, resp.GetExpiresAt())
}

func TestServiceImpl_IssueToken_ClusterScopeOnly(t *testing.T) {
	ctx := context.Background()
	backend := &mockBackend{}

	service := &serviceImpl{
		tokenBackend: backend,
	}

	req := &v1.IssueScopedTokenRequest{
		UserIdentifier: "admin@example.com",
		ClusterName:    "prod-cluster",
		// No namespace or deployment = cluster-wide scope
		Ttl: durationpb.New(5 * time.Minute),
	}

	resp, err := service.IssueToken(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "mock-token-abc123", resp.GetToken())
}

func TestServiceImpl_IssueToken_NamespaceScopeOnly(t *testing.T) {
	ctx := context.Background()
	backend := &mockBackend{}

	service := &serviceImpl{
		tokenBackend: backend,
	}

	req := &v1.IssueScopedTokenRequest{
		UserIdentifier: "user@example.com",
		ClusterName:    "staging",
		Namespace:      "monitoring",
		// No deployment = namespace-wide scope
		Ttl: durationpb.New(2 * time.Minute),
	}

	resp, err := service.IssueToken(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "mock-token-abc123", resp.GetToken())
}

func TestServiceImpl_IssueToken_MissingUserIdentifier(t *testing.T) {
	ctx := context.Background()
	backend := &mockBackend{}

	service := &serviceImpl{
		tokenBackend: backend,
	}

	req := &v1.IssueScopedTokenRequest{
		UserIdentifier: "", // Empty user identifier
		ClusterName:    "production",
		Namespace:      "default",
		Deployment:     "nginx",
	}

	resp, err := service.IssueToken(ctx, req)

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "user_identifier is required")
}

func TestServiceImpl_IssueToken_MissingClusterName(t *testing.T) {
	ctx := context.Background()
	backend := &mockBackend{}

	service := &serviceImpl{
		tokenBackend: backend,
	}

	req := &v1.IssueScopedTokenRequest{
		UserIdentifier: "user@example.com",
		ClusterName:    "", // Empty cluster name
		Namespace:      "default",
		Deployment:     "nginx",
	}

	resp, err := service.IssueToken(ctx, req)

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "cluster_name is required")
}

func TestServiceImpl_IssueToken_NilRequest(t *testing.T) {
	ctx := context.Background()
	backend := &mockBackend{}

	service := &serviceImpl{
		tokenBackend: backend,
	}

	resp, err := service.IssueToken(ctx, nil)

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "request is nil")
}

func TestServiceImpl_IssueToken_InvalidNamespaceFormat(t *testing.T) {
	ctx := context.Background()
	backend := &mockBackend{}

	service := &serviceImpl{
		tokenBackend: backend,
	}

	req := &v1.IssueScopedTokenRequest{
		UserIdentifier: "user@example.com",
		ClusterName:    "production",
		Namespace:      "INVALID_NAMESPACE!", // Invalid Kubernetes namespace name
		Deployment:     "nginx",
	}

	resp, err := service.IssueToken(ctx, req)

	require.Error(t, err)
	assert.Nil(t, resp)
	// Error comes from dynamic.BuildDynamicScope validation
}

func TestServiceImpl_IssueToken_InvalidDeploymentFormat(t *testing.T) {
	ctx := context.Background()
	backend := &mockBackend{}

	service := &serviceImpl{
		tokenBackend: backend,
	}

	req := &v1.IssueScopedTokenRequest{
		UserIdentifier: "user@example.com",
		ClusterName:    "production",
		Namespace:      "default",
		Deployment:     "INVALID DEPLOYMENT NAME!", // Invalid Kubernetes deployment name
	}

	resp, err := service.IssueToken(ctx, req)

	require.Error(t, err)
	assert.Nil(t, resp)
	// Error comes from dynamic.BuildDynamicScope validation
}

func TestServiceImpl_IssueToken_DefaultTTL(t *testing.T) {
	ctx := context.Background()

	var capturedTTL time.Duration
	backend := &mockBackend{
		issueEphemeralFunc: func(ctx context.Context, name string, roleNames []string, dynamicScope *storage.DynamicAccessScope, ttl time.Duration) (string, *time.Time, error) {
			capturedTTL = ttl
			expiresAt := time.Now().Add(ttl)
			return "token", &expiresAt, nil
		},
	}

	service := &serviceImpl{
		tokenBackend: backend,
	}

	req := &v1.IssueScopedTokenRequest{
		UserIdentifier: "user@example.com",
		ClusterName:    "prod",
		Ttl:            nil, // No TTL specified
	}

	_, err := service.IssueToken(ctx, req)

	require.NoError(t, err)
	assert.Equal(t, DefaultTokenTTL, capturedTTL)
}

func TestServiceImpl_IssueToken_ZeroTTL_UsesDefault(t *testing.T) {
	ctx := context.Background()

	var capturedTTL time.Duration
	backend := &mockBackend{
		issueEphemeralFunc: func(ctx context.Context, name string, roleNames []string, dynamicScope *storage.DynamicAccessScope, ttl time.Duration) (string, *time.Time, error) {
			capturedTTL = ttl
			expiresAt := time.Now().Add(ttl)
			return "token", &expiresAt, nil
		},
	}

	service := &serviceImpl{
		tokenBackend: backend,
	}

	req := &v1.IssueScopedTokenRequest{
		UserIdentifier: "user@example.com",
		ClusterName:    "prod",
		Ttl:            durationpb.New(0), // Zero duration
	}

	_, err := service.IssueToken(ctx, req)

	require.NoError(t, err)
	assert.Equal(t, DefaultTokenTTL, capturedTTL)
}

func TestServiceImpl_IssueToken_NegativeTTL_UsesDefault(t *testing.T) {
	ctx := context.Background()

	var capturedTTL time.Duration
	backend := &mockBackend{
		issueEphemeralFunc: func(ctx context.Context, name string, roleNames []string, dynamicScope *storage.DynamicAccessScope, ttl time.Duration) (string, *time.Time, error) {
			capturedTTL = ttl
			expiresAt := time.Now().Add(ttl)
			return "token", &expiresAt, nil
		},
	}

	service := &serviceImpl{
		tokenBackend: backend,
	}

	req := &v1.IssueScopedTokenRequest{
		UserIdentifier: "user@example.com",
		ClusterName:    "prod",
		Ttl:            durationpb.New(-1 * time.Minute), // Negative duration
	}

	_, err := service.IssueToken(ctx, req)

	require.NoError(t, err)
	assert.Equal(t, DefaultTokenTTL, capturedTTL)
}

func TestServiceImpl_IssueToken_ExcessiveTTL_Capped(t *testing.T) {
	ctx := context.Background()

	var capturedTTL time.Duration
	backend := &mockBackend{
		issueEphemeralFunc: func(ctx context.Context, name string, roleNames []string, dynamicScope *storage.DynamicAccessScope, ttl time.Duration) (string, *time.Time, error) {
			capturedTTL = ttl
			expiresAt := time.Now().Add(ttl)
			return "token", &expiresAt, nil
		},
	}

	service := &serviceImpl{
		tokenBackend: backend,
	}

	req := &v1.IssueScopedTokenRequest{
		UserIdentifier: "user@example.com",
		ClusterName:    "prod",
		Ttl:            durationpb.New(10 * time.Minute), // Exceeds MaxTokenTTL (5 min)
	}

	_, err := service.IssueToken(ctx, req)

	require.NoError(t, err)
	assert.Equal(t, MaxTokenTTL, capturedTTL)
}

func TestServiceImpl_IssueToken_ValidCustomTTL(t *testing.T) {
	ctx := context.Background()

	var capturedTTL time.Duration
	backend := &mockBackend{
		issueEphemeralFunc: func(ctx context.Context, name string, roleNames []string, dynamicScope *storage.DynamicAccessScope, ttl time.Duration) (string, *time.Time, error) {
			capturedTTL = ttl
			expiresAt := time.Now().Add(ttl)
			return "token", &expiresAt, nil
		},
	}

	service := &serviceImpl{
		tokenBackend: backend,
	}

	customTTL := 2 * time.Minute
	req := &v1.IssueScopedTokenRequest{
		UserIdentifier: "user@example.com",
		ClusterName:    "prod",
		Ttl:            durationpb.New(customTTL),
	}

	_, err := service.IssueToken(ctx, req)

	require.NoError(t, err)
	assert.Equal(t, customTTL, capturedTTL)
}

func TestServiceImpl_IssueToken_BackendError(t *testing.T) {
	ctx := context.Background()

	backend := &mockBackend{
		issueEphemeralFunc: func(ctx context.Context, name string, roleNames []string, dynamicScope *storage.DynamicAccessScope, ttl time.Duration) (string, *time.Time, error) {
			return "", nil, errors.New("database connection failed")
		},
	}

	service := &serviceImpl{
		tokenBackend: backend,
	}

	req := &v1.IssueScopedTokenRequest{
		UserIdentifier: "user@example.com",
		ClusterName:    "prod",
		Namespace:      "default",
	}

	resp, err := service.IssueToken(ctx, req)

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "failed to issue token")
}

func TestServiceImpl_IssueToken_VerifyAnalystRole(t *testing.T) {
	ctx := context.Background()

	var capturedRoles []string
	backend := &mockBackend{
		issueEphemeralFunc: func(ctx context.Context, name string, roleNames []string, dynamicScope *storage.DynamicAccessScope, ttl time.Duration) (string, *time.Time, error) {
			capturedRoles = roleNames
			expiresAt := time.Now().Add(ttl)
			return "token", &expiresAt, nil
		},
	}

	service := &serviceImpl{
		tokenBackend: backend,
	}

	req := &v1.IssueScopedTokenRequest{
		UserIdentifier: "user@example.com",
		ClusterName:    "prod",
	}

	_, err := service.IssueToken(ctx, req)

	require.NoError(t, err)
	assert.Contains(t, capturedRoles, "Analyst")
}

func TestServiceImpl_IssueToken_VerifyDynamicScope(t *testing.T) {
	ctx := context.Background()

	var capturedScope *storage.DynamicAccessScope
	backend := &mockBackend{
		issueEphemeralFunc: func(ctx context.Context, name string, roleNames []string, dynamicScope *storage.DynamicAccessScope, ttl time.Duration) (string, *time.Time, error) {
			capturedScope = dynamicScope
			expiresAt := time.Now().Add(ttl)
			return "token", &expiresAt, nil
		},
	}

	service := &serviceImpl{
		tokenBackend: backend,
	}

	req := &v1.IssueScopedTokenRequest{
		UserIdentifier: "user@example.com",
		ClusterName:    "production-cluster",
		Namespace:      "kube-system",
		Deployment:     "coredns",
	}

	_, err := service.IssueToken(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, capturedScope)
	assert.Equal(t, "production-cluster", capturedScope.GetClusterName())
	assert.Equal(t, "kube-system", capturedScope.GetNamespace())
	assert.Equal(t, "coredns", capturedScope.GetDeployment())
}

func TestGenerateTokenName(t *testing.T) {
	tests := []struct {
		name     string
		req      *v1.IssueScopedTokenRequest
		expected string
	}{
		{
			name: "basic token name",
			req: &v1.IssueScopedTokenRequest{
				UserIdentifier: "user@example.com",
				ClusterName:    "prod",
			},
			expected: "ocp-console:user@example.com@prod",
		},
		{
			name: "token name with special characters",
			req: &v1.IssueScopedTokenRequest{
				UserIdentifier: "system:serviceaccount:default:test",
				ClusterName:    "test-cluster-123",
			},
			expected: "ocp-console:system:serviceaccount:default:test@test-cluster-123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateTokenName(tt.req)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDetermineTTL(t *testing.T) {
	tests := []struct {
		name     string
		input    *durationpb.Duration
		expected time.Duration
	}{
		{
			name:     "nil TTL returns default",
			input:    nil,
			expected: DefaultTokenTTL,
		},
		{
			name:     "zero TTL returns default",
			input:    durationpb.New(0),
			expected: DefaultTokenTTL,
		},
		{
			name:     "negative TTL returns default",
			input:    durationpb.New(-5 * time.Minute),
			expected: DefaultTokenTTL,
		},
		{
			name:     "excessive TTL is capped at max",
			input:    durationpb.New(30 * time.Minute),
			expected: MaxTokenTTL,
		},
		{
			name:     "valid TTL is returned as-is",
			input:    durationpb.New(3 * time.Minute),
			expected: 3 * time.Minute,
		},
		{
			name:     "max TTL is allowed",
			input:    durationpb.New(MaxTokenTTL),
			expected: MaxTokenTTL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := determineTTL(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateRequest(t *testing.T) {
	tests := []struct {
		name          string
		req           *v1.IssueScopedTokenRequest
		expectError   bool
		errorContains string
	}{
		{
			name: "valid request with all fields",
			req: &v1.IssueScopedTokenRequest{
				UserIdentifier: "user@example.com",
				ClusterName:    "prod",
				Namespace:      "default",
				Deployment:     "nginx",
			},
			expectError: false,
		},
		{
			name: "valid request without namespace/deployment",
			req: &v1.IssueScopedTokenRequest{
				UserIdentifier: "user@example.com",
				ClusterName:    "prod",
			},
			expectError: false,
		},
		{
			name:          "nil request",
			req:           nil,
			expectError:   true,
			errorContains: "request is nil",
		},
		{
			name: "missing user identifier",
			req: &v1.IssueScopedTokenRequest{
				UserIdentifier: "",
				ClusterName:    "prod",
			},
			expectError:   true,
			errorContains: "user_identifier is required",
		},
		{
			name: "missing cluster name",
			req: &v1.IssueScopedTokenRequest{
				UserIdentifier: "user@example.com",
				ClusterName:    "",
			},
			expectError:   true,
			errorContains: "cluster_name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRequest(tt.req)
			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
