// Package discover handles auto-discovery of ACS cluster IDs from Kubernetes
// clusters using multiple fallback methods.
package discover

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/stackrox/co-acs-importer/internal/models"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

// k8sResourceReader abstracts Kubernetes resource lookups for testing.
type k8sResourceReader interface {
	getAdmissionControlClusterID(ctx context.Context) (string, error)
	getOpenShiftClusterID(ctx context.Context) (string, error)
	getHelmSecretClusterName(ctx context.Context) (string, error)
}

// k8sDiscoveryClient is the production implementation using a dynamic k8s client.
type k8sDiscoveryClient struct {
	dynamic dynamic.Interface
}

// NewK8sDiscoveryClient creates a k8sResourceReader from a dynamic k8s client.
func NewK8sDiscoveryClient(dynClient dynamic.Interface) k8sResourceReader {
	return &k8sDiscoveryClient{dynamic: dynClient}
}

// IMP-MAP-016: admission-control ConfigMap in stackrox namespace.
func (c *k8sDiscoveryClient) getAdmissionControlClusterID(ctx context.Context) (string, error) {
	gvr := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "configmaps"}
	obj, err := c.dynamic.Resource(gvr).Namespace("stackrox").Get(ctx, "admission-control", metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("get admission-control ConfigMap: %w", err)
	}

	data, found, err := unstructured.NestedStringMap(obj.Object, "data")
	if err != nil || !found {
		return "", fmt.Errorf("parse ConfigMap data: %w", err)
	}

	clusterID, ok := data["cluster-id"]
	if !ok || clusterID == "" {
		return "", errors.New("cluster-id not found in admission-control ConfigMap")
	}
	return clusterID, nil
}

// IMP-MAP-017: OpenShift ClusterVersion resource.
func (c *k8sDiscoveryClient) getOpenShiftClusterID(ctx context.Context) (string, error) {
	gvr := schema.GroupVersionResource{
		Group:    "config.openshift.io",
		Version:  "v1",
		Resource: "clusterversions",
	}
	obj, err := c.dynamic.Resource(gvr).Get(ctx, "version", metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("get ClusterVersion: %w", err)
	}

	clusterID, found, err := unstructured.NestedString(obj.Object, "spec", "clusterID")
	if err != nil || !found {
		return "", fmt.Errorf("parse ClusterVersion.spec.clusterID: %w", err)
	}
	if clusterID == "" {
		return "", errors.New("ClusterVersion.spec.clusterID is empty")
	}
	return clusterID, nil
}

// IMP-MAP-018: helm-effective-cluster-name secret in stackrox namespace.
func (c *k8sDiscoveryClient) getHelmSecretClusterName(ctx context.Context) (string, error) {
	gvr := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "secrets"}
	obj, err := c.dynamic.Resource(gvr).Namespace("stackrox").Get(ctx, "helm-effective-cluster-name", metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("get helm-effective-cluster-name Secret: %w", err)
	}

	data, found, err := unstructured.NestedStringMap(obj.Object, "data")
	if err != nil || !found {
		return "", fmt.Errorf("parse Secret data: %w", err)
	}

	encodedName, ok := data["cluster-name"]
	if !ok || encodedName == "" {
		return "", errors.New("cluster-name not found in helm-effective-cluster-name Secret")
	}

	// Kubernetes secrets are base64-encoded.
	decoded, err := base64.StdEncoding.DecodeString(encodedName)
	if err != nil {
		return "", fmt.Errorf("decode cluster-name: %w", err)
	}
	return string(decoded), nil
}

// DiscoverClusterID attempts to resolve the ACS cluster ID for the given source cluster.
//
// Discovery chain (try in order, use first success):
//  1. admission-control ConfigMap: direct ACS cluster UUID (IMP-MAP-016).
//  2. OpenShift ClusterVersion: match providerMetadata.cluster.id (IMP-MAP-017).
//  3. helm-effective-cluster-name secret: match by cluster name (IMP-MAP-018).
//
// Returns error if all methods fail.
func DiscoverClusterID(
	ctx context.Context,
	k8s k8sResourceReader,
	acs models.ACSClient,
) (string, error) {
	// IMP-MAP-016: admission-control ConfigMap.
	if clusterID, err := k8s.getAdmissionControlClusterID(ctx); err == nil {
		return clusterID, nil
	}

	// IMP-MAP-017: OpenShift ClusterVersion.
	if ocpClusterID, err := k8s.getOpenShiftClusterID(ctx); err == nil {
		clusters, err := acs.ListClusters(ctx)
		if err != nil {
			return "", fmt.Errorf("list ACS clusters for OpenShift ID match: %w", err)
		}
		for _, c := range clusters {
			if c.ProviderClusterID == ocpClusterID {
				return c.ID, nil
			}
		}
		return "", fmt.Errorf("OpenShift cluster ID %q not found in ACS clusters", ocpClusterID)
	}

	// IMP-MAP-018: helm-effective-cluster-name secret.
	if clusterName, err := k8s.getHelmSecretClusterName(ctx); err == nil {
		clusters, err := acs.ListClusters(ctx)
		if err != nil {
			return "", fmt.Errorf("list ACS clusters for helm cluster name match: %w", err)
		}
		for _, c := range clusters {
			if c.Name == clusterName {
				return c.ID, nil
			}
		}
		return "", fmt.Errorf("helm cluster name %q not found in ACS clusters", clusterName)
	}

	return "", errors.New("all discovery methods failed to resolve ACS cluster ID")
}
