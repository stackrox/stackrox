package dynamic

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildDynamicScope(t *testing.T) {
	tests := []struct {
		name        string
		clusterName string
		namespace   string
		deployment  string
		wantErr     bool
		errType     error
	}{
		{
			name:        "valid cluster scope",
			clusterName: "prod-cluster",
			namespace:   "",
			deployment:  "",
			wantErr:     false,
		},
		{
			name:        "valid namespace scope",
			clusterName: "prod-cluster",
			namespace:   "default",
			deployment:  "",
			wantErr:     false,
		},
		{
			name:        "valid deployment scope",
			clusterName: "prod-cluster",
			namespace:   "default",
			deployment:  "nginx",
			wantErr:     false,
		},
		{
			name:        "namespace with hyphens",
			clusterName: "cluster-1",
			namespace:   "kube-system",
			deployment:  "",
			wantErr:     false,
		},
		{
			name:        "invalid namespace - dots not allowed",
			clusterName: "cluster-1",
			namespace:   "namespace.example",
			deployment:  "",
			wantErr:     true,
			errType:     errox.InvalidArgs,
		},
		{
			name:        "deployment with hyphens",
			clusterName: "cluster-1",
			namespace:   "default",
			deployment:  "my-deployment-v2",
			wantErr:     false,
		},
		{
			name:        "deployment with dots allowed",
			clusterName: "cluster-1",
			namespace:   "default",
			deployment:  "my-deployment.v2",
			wantErr:     false,
		},
		{
			name:        "empty cluster name",
			clusterName: "",
			namespace:   "default",
			deployment:  "",
			wantErr:     true,
			errType:     errox.InvalidArgs,
		},
		{
			name:        "deployment without namespace",
			clusterName: "cluster-1",
			namespace:   "",
			deployment:  "nginx",
			wantErr:     true,
			errType:     errox.InvalidArgs,
		},
		{
			name:        "namespace too long",
			clusterName: "cluster-1",
			namespace:   "this-is-a-very-long-namespace-name-that-exceeds-the-maximum-allowed-length-for-kubernetes-namespaces",
			deployment:  "",
			wantErr:     true,
			errType:     errox.InvalidArgs,
		},
		{
			name:        "invalid namespace - uppercase",
			clusterName: "cluster-1",
			namespace:   "Default",
			deployment:  "",
			wantErr:     true,
			errType:     errox.InvalidArgs,
		},
		{
			name:        "invalid namespace - starts with hyphen",
			clusterName: "cluster-1",
			namespace:   "-invalid",
			deployment:  "",
			wantErr:     true,
			errType:     errox.InvalidArgs,
		},
		{
			name:        "invalid namespace - ends with hyphen",
			clusterName: "cluster-1",
			namespace:   "invalid-",
			deployment:  "",
			wantErr:     true,
			errType:     errox.InvalidArgs,
		},
		{
			name:        "invalid namespace - special characters",
			clusterName: "cluster-1",
			namespace:   "invalid_namespace",
			deployment:  "",
			wantErr:     true,
			errType:     errox.InvalidArgs,
		},
		{
			name:        "invalid deployment - uppercase",
			clusterName: "cluster-1",
			namespace:   "default",
			deployment:  "Nginx",
			wantErr:     true,
			errType:     errox.InvalidArgs,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scope, err := BuildDynamicScope(tt.clusterName, tt.namespace, tt.deployment)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errType != nil {
					assert.ErrorIs(t, err, tt.errType)
				}
				assert.Nil(t, scope)
			} else {
				require.NoError(t, err)
				require.NotNil(t, scope)
				assert.Equal(t, tt.clusterName, scope.GetClusterName())
				assert.Equal(t, tt.namespace, scope.GetNamespace())
				assert.Equal(t, tt.deployment, scope.GetDeployment())
			}
		})
	}
}

func TestScopeDescription(t *testing.T) {
	tests := []struct {
		name        string
		scope       func() *storage.DynamicAccessScope
		wantContain string
	}{
		{
			name: "nil scope",
			scope: func() *storage.DynamicAccessScope {
				return nil
			},
			wantContain: "unrestricted",
		},
		{
			name: "cluster scope",
			scope: func() *storage.DynamicAccessScope {
				s, _ := BuildDynamicScope("prod-cluster", "", "")
				return s
			},
			wantContain: "all namespaces",
		},
		{
			name: "namespace scope",
			scope: func() *storage.DynamicAccessScope {
				s, _ := BuildDynamicScope("prod-cluster", "default", "")
				return s
			},
			wantContain: "all deployments",
		},
		{
			name: "deployment scope",
			scope: func() *storage.DynamicAccessScope {
				s, _ := BuildDynamicScope("prod-cluster", "default", "nginx")
				return s
			},
			wantContain: "deployment=nginx",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			desc := ScopeDescription(tt.scope())
			assert.Contains(t, desc, tt.wantContain)
		})
	}
}

func TestScopeLevelChecks(t *testing.T) {
	clusterScope, err := BuildDynamicScope("cluster-1", "", "")
	require.NoError(t, err)

	namespaceScope, err := BuildDynamicScope("cluster-1", "default", "")
	require.NoError(t, err)

	deploymentScope, err := BuildDynamicScope("cluster-1", "default", "nginx")
	require.NoError(t, err)

	t.Run("IsClusterScoped", func(t *testing.T) {
		assert.True(t, IsClusterScoped(clusterScope))
		assert.False(t, IsClusterScoped(namespaceScope))
		assert.False(t, IsClusterScoped(deploymentScope))
		assert.False(t, IsClusterScoped(nil))
	})

	t.Run("IsNamespaceScoped", func(t *testing.T) {
		assert.False(t, IsNamespaceScoped(clusterScope))
		assert.True(t, IsNamespaceScoped(namespaceScope))
		assert.False(t, IsNamespaceScoped(deploymentScope))
		assert.False(t, IsNamespaceScoped(nil))
	})

	t.Run("IsDeploymentScoped", func(t *testing.T) {
		assert.False(t, IsDeploymentScoped(clusterScope))
		assert.False(t, IsDeploymentScoped(namespaceScope))
		assert.True(t, IsDeploymentScoped(deploymentScope))
		assert.False(t, IsDeploymentScoped(nil))
	})
}
