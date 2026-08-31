package acm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	clusterviewv1alpha1 "github.com/stolostron/cluster-lifecycle-api/clusterview/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

func TestNewClientForConfig(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer server.Close()

	client, err := NewClientForConfig(&rest.Config{Host: server.URL})
	require.NoError(t, err)
	require.NotNil(t, client)
	assert.NotNil(t, client.clusterviewClient)
}

func TestClient_WithFakeServer(t *testing.T) {
	testPermissions := &clusterviewv1alpha1.UserPermissionList{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "clusterview.open-cluster-management.io/v1alpha1",
			Kind:       "UserPermissionList",
		},
		ListMeta: metav1.ListMeta{},
		Items: []clusterviewv1alpha1.UserPermission{
			{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "clusterview.open-cluster-management.io/v1alpha1",
					Kind:       "UserPermission",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "managedcluster:admin",
				},
				Status: clusterviewv1alpha1.UserPermissionStatus{
					Bindings: []clusterviewv1alpha1.ClusterBinding{
						{
							Cluster:    "cluster1",
							Scope:      clusterviewv1alpha1.BindingScopeCluster,
							Namespaces: []string{"*"},
						},
						{
							Cluster:    "cluster2",
							Scope:      clusterviewv1alpha1.BindingScopeNamespace,
							Namespaces: []string{"default", "kube-system"},
						},
					},
					ClusterRoleDefinition: clusterviewv1alpha1.ClusterRoleDefinition{
						Rules: []rbacv1.PolicyRule{
							{
								APIGroups: []string{"*"},
								Resources: []string{"*"},
								Verbs:     []string{"*"},
							},
						},
					},
				},
			},
			{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "clusterview.open-cluster-management.io/v1alpha1",
					Kind:       "UserPermission",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "managedcluster:view",
				},
				Status: clusterviewv1alpha1.UserPermissionStatus{
					Bindings: []clusterviewv1alpha1.ClusterBinding{
						{
							Cluster:    "cluster1",
							Scope:      clusterviewv1alpha1.BindingScopeCluster,
							Namespaces: []string{"*"},
						},
					},
					ClusterRoleDefinition: clusterviewv1alpha1.ClusterRoleDefinition{
						Rules: []rbacv1.PolicyRule{
							{
								APIGroups: []string{""},
								Resources: []string{"pods", "services"},
								Verbs:     []string{"get", "list", "watch"},
							},
						},
					},
				},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedBasePath := "/apis/clusterview.open-cluster-management.io/v1alpha1/userpermissions"

		switch {
		case r.Method == http.MethodGet && r.URL.Path == expectedBasePath:
			w.Header().Set("Content-Type", "application/json")
			err := json.NewEncoder(w).Encode(testPermissions)
			require.NoError(t, err)

		case r.Method == http.MethodGet && r.URL.Path == expectedBasePath+"/managedcluster:admin":
			w.Header().Set("Content-Type", "application/json")
			err := json.NewEncoder(w).Encode(&testPermissions.Items[0])
			require.NoError(t, err)

		case r.Method == http.MethodGet && r.URL.Path == expectedBasePath+"/managedcluster:view":
			w.Header().Set("Content-Type", "application/json")
			err := json.NewEncoder(w).Encode(&testPermissions.Items[1])
			require.NoError(t, err)

		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, err := NewClientForConfig(&rest.Config{Host: server.URL})
	require.NoError(t, err)
	require.NotNil(t, client)

	t.Run("ListUserPermissions", func(t *testing.T) {
		list, err := client.ListUserPermissions(context.Background(), metav1.ListOptions{})
		require.NoError(t, err)
		require.NotNil(t, list)

		assert.Len(t, list.Items, 2)

		assert.Equal(t, "managedcluster:admin", list.Items[0].Name)
		assert.Len(t, list.Items[0].Status.Bindings, 2)
		assert.Equal(t, "cluster1", list.Items[0].Status.Bindings[0].Cluster)
		assert.Equal(t, clusterviewv1alpha1.BindingScopeCluster, list.Items[0].Status.Bindings[0].Scope)
		assert.Equal(t, []string{"*"}, list.Items[0].Status.Bindings[0].Namespaces)

		assert.Equal(t, "cluster2", list.Items[0].Status.Bindings[1].Cluster)
		assert.Equal(t, clusterviewv1alpha1.BindingScopeNamespace, list.Items[0].Status.Bindings[1].Scope)
		assert.Equal(t, []string{"default", "kube-system"}, list.Items[0].Status.Bindings[1].Namespaces)

		assert.Len(t, list.Items[0].Status.ClusterRoleDefinition.Rules, 1)
		assert.Equal(t, []string{"*"}, list.Items[0].Status.ClusterRoleDefinition.Rules[0].APIGroups)
		assert.Equal(t, []string{"*"}, list.Items[0].Status.ClusterRoleDefinition.Rules[0].Resources)
		assert.Equal(t, []string{"*"}, list.Items[0].Status.ClusterRoleDefinition.Rules[0].Verbs)

		assert.Equal(t, "managedcluster:view", list.Items[1].Name)
		assert.Len(t, list.Items[1].Status.Bindings, 1)
		assert.Equal(t, "cluster1", list.Items[1].Status.Bindings[0].Cluster)
	})

	t.Run("GetUserPermission_Admin", func(t *testing.T) {
		permission, err := client.GetUserPermission(context.Background(), "managedcluster:admin", metav1.GetOptions{})
		require.NoError(t, err)
		require.NotNil(t, permission)

		assert.Equal(t, "managedcluster:admin", permission.Name)
		assert.Len(t, permission.Status.Bindings, 2)

		assert.Equal(t, "cluster1", permission.Status.Bindings[0].Cluster)
		assert.Equal(t, clusterviewv1alpha1.BindingScopeCluster, permission.Status.Bindings[0].Scope)
		assert.Equal(t, []string{"*"}, permission.Status.Bindings[0].Namespaces)

		assert.Equal(t, "cluster2", permission.Status.Bindings[1].Cluster)
		assert.Equal(t, clusterviewv1alpha1.BindingScopeNamespace, permission.Status.Bindings[1].Scope)
		assert.Equal(t, []string{"default", "kube-system"}, permission.Status.Bindings[1].Namespaces)

		assert.Len(t, permission.Status.ClusterRoleDefinition.Rules, 1)
		rule := permission.Status.ClusterRoleDefinition.Rules[0]
		assert.Equal(t, []string{"*"}, rule.APIGroups)
		assert.Equal(t, []string{"*"}, rule.Resources)
		assert.Equal(t, []string{"*"}, rule.Verbs)
	})

	t.Run("GetUserPermission_View", func(t *testing.T) {
		permission, err := client.GetUserPermission(context.Background(), "managedcluster:view", metav1.GetOptions{})
		require.NoError(t, err)
		require.NotNil(t, permission)

		assert.Equal(t, "managedcluster:view", permission.Name)
		assert.Len(t, permission.Status.Bindings, 1)

		assert.Equal(t, "cluster1", permission.Status.Bindings[0].Cluster)
		assert.Equal(t, clusterviewv1alpha1.BindingScopeCluster, permission.Status.Bindings[0].Scope)
		assert.Equal(t, []string{"*"}, permission.Status.Bindings[0].Namespaces)

		assert.Len(t, permission.Status.ClusterRoleDefinition.Rules, 1)
		rule := permission.Status.ClusterRoleDefinition.Rules[0]
		assert.Equal(t, []string{""}, rule.APIGroups)
		assert.Equal(t, []string{"pods", "services"}, rule.Resources)
		assert.Equal(t, []string{"get", "list", "watch"}, rule.Verbs)
	})
}

func TestClient_WithFakeServer_EmptyList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		emptyList := &clusterviewv1alpha1.UserPermissionList{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "clusterview.open-cluster-management.io/v1alpha1",
				Kind:       "UserPermissionList",
			},
			ListMeta: metav1.ListMeta{},
			Items:    []clusterviewv1alpha1.UserPermission{},
		}

		w.Header().Set("Content-Type", "application/json")
		err := json.NewEncoder(w).Encode(emptyList)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClientForConfig(&rest.Config{Host: server.URL})
	require.NoError(t, err)

	list, err := client.ListUserPermissions(context.Background(), metav1.ListOptions{})
	require.NoError(t, err)
	require.NotNil(t, list)
	assert.Len(t, list.Items, 0)
}

func TestClient_WithFakeServer_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer server.Close()

	client, err := NewClientForConfig(&rest.Config{Host: server.URL})
	require.NoError(t, err)

	_, err = client.GetUserPermission(context.Background(), "nonexistent", metav1.GetOptions{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get user permission")
}
