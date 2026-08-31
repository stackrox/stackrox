package acm

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/retryablehttp"
	clusterviewclient "github.com/stolostron/cluster-lifecycle-api/client/clusterview/clientset/versioned/typed/clusterview/v1alpha1"
	clusterviewv1alpha1 "github.com/stolostron/cluster-lifecycle-api/clusterview/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

// Client provides access to the ACM clusterview aggregate API.
type Client struct {
	clusterviewClient clusterviewclient.ClusterviewV1alpha1Interface
}

// NewClientForConfig creates a new client for the ACM clusterview API using the provided config.
// The config's transport is wrapped with retry logic for transient errors.
func NewClientForConfig(config *rest.Config) (*Client, error) {
	cfg := rest.CopyConfig(config)
	retryablehttp.ConfigureRESTConfig(cfg)
	clusterviewClient, err := clusterviewclient.NewForConfig(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create clusterview client")
	}
	return &Client{
		clusterviewClient: clusterviewClient,
	}, nil
}

// ListUserPermissions retrieves the list of user permissions from the ACM clusterview API.
// This calls the aggregate API at /apis/clusterview.open-cluster-management.io/v1alpha1/userpermissions.
func (c *Client) ListUserPermissions(
	ctx context.Context,
	opts metav1.ListOptions,
) (*clusterviewv1alpha1.UserPermissionList, error) {
	list, err := c.clusterviewClient.UserPermissions().List(ctx, opts)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list user permissions")
	}
	return list, nil
}

// GetUserPermission retrieves a specific user permission by name from the ACM clusterview API.
func (c *Client) GetUserPermission(ctx context.Context, name string, opts metav1.GetOptions) (*clusterviewv1alpha1.UserPermission, error) {
	permission, err := c.clusterviewClient.UserPermissions().Get(ctx, name, opts)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get user permission %q", name)
	}
	return permission, nil
}
