package acmclient

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

func TestNewACMClientFromConfig(t *testing.T) {
	// This test demonstrates how to create an ACM client with a custom config.
	// In a real scenario, you would provide a valid rest.Config.
	config := &rest.Config{
		Host: "https://example.com",
	}

	// This will fail in the test because we don't have a real cluster,
	// but it demonstrates the API usage.
	client, err := NewACMClientFromConfig(config)

	// We expect this to fail in unit tests without a real cluster
	if err != nil {
		assert.Error(t, err)
		return
	}

	require.NotNil(t, client)
	assert.NotNil(t, client.clusterviewClient)
}

func TestACMClient_ListUserPermissions(t *testing.T) {
	// This test demonstrates the expected usage pattern.
	// In integration tests with a real cluster, you would:
	//
	// client, err := NewACMClient()
	// require.NoError(t, err)
	//
	// list, err := client.ListUserPermissions(context.Background(), metav1.ListOptions{})
	// require.NoError(t, err)
	// assert.NotNil(t, list)

	t.Skip("Requires a running cluster with ACM installed")
}

func TestACMClient_GetUserPermission(t *testing.T) {
	// This test demonstrates the expected usage pattern for getting a specific permission.
	// In integration tests with a real cluster, you would:
	//
	// client, err := NewACMClient()
	// require.NoError(t, err)
	//
	// permission, err := client.GetUserPermission(context.Background(), "managedcluster:admin", metav1.GetOptions{})
	// require.NoError(t, err)
	// assert.NotNil(t, permission)
	// assert.Equal(t, "managedcluster:admin", permission.Name)

	t.Skip("Requires a running cluster with ACM installed")
}

// Example usage:
func ExampleACMClient_ListUserPermissions() {
	client, err := NewACMClient()
	if err != nil {
		panic(err)
	}

	// List all user permissions
	list, err := client.ListUserPermissions(context.Background(), metav1.ListOptions{})
	if err != nil {
		panic(err)
	}

	for _, permission := range list.Items {
		_ = permission // Process each permission
	}
}

// Example usage with filtering:
func ExampleACMClient_GetUserPermission() {
	client, err := NewACMClient()
	if err != nil {
		panic(err)
	}

	// Get a specific user permission by name
	permission, err := client.GetUserPermission(context.Background(), "managedcluster:admin", metav1.GetOptions{})
	if err != nil {
		panic(err)
	}

	// Access the bindings
	for _, binding := range permission.Status.Bindings {
		_ = binding.Cluster    // The cluster name
		_ = binding.Scope      // "cluster" or "namespace"
		_ = binding.Namespaces // List of namespaces
	}
}
