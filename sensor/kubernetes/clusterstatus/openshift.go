package clusterstatus

import (
	"context"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

const (
	// Each resource tag for different cloud providers will have this set in case special "flavors" for installing
	// OpenShift were used (e.g. OSD, ARO, ROSA).
	redHatClusterTypeTagKey = "red-hat-clustertype"
)

var (
	infrastructureGVR = schema.GroupVersionResource{Group: "config.openshift.io", Version: "v1", Resource: "infrastructures"}
	clusterVersionGVR = schema.GroupVersionResource{Group: "config.openshift.io", Version: "v1", Resource: "clusterversions"}
)

type providerMetadataFromOpenShift = func(ctx context.Context, p dynamic.Interface) (*storage.ProviderMetadata, error)

// getProviderMetadataFromOpenShiftConfig reads Infrastructure and ClusterVersion CRs
// via the dynamic client to determine cloud provider metadata.
func getProviderMetadataFromOpenShiftConfig(ctx context.Context,
	client dynamic.Interface) (*storage.ProviderMetadata, error) {
	infraObj, err := client.Resource(infrastructureGVR).Get(ctx, "cluster", metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "retrieving cluster infrastructure CR")
	}

	versionObj, err := client.Resource(clusterVersionGVR).Get(ctx, "version", metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "retrieving cluster version CR")
	}

	return unstructuredToProviderMetadata(infraObj.Object, versionObj.Object), nil
}

func unstructuredToProviderMetadata(infra, clusterVersion map[string]interface{}) *storage.ProviderMetadata {
	platformType, _, _ := nestedString(infra, "status", "platformStatus", "type")
	if platformType == "" {
		return nil
	}

	infraName, _, _ := nestedString(infra, "status", "infrastructureName")
	clusterID, _, _ := nestedString(clusterVersion, "spec", "clusterID")

	clusterMeta := &storage.ClusterMetadata{
		Type: storage.ClusterMetadata_OCP,
		Name: infraName,
		Id:   clusterID,
	}

	// Check resource tags for managed cluster types (OSD, ROSA, ARO)
	clusterMeta.Type = clusterTypeFromResourceTags(infra, platformType)

	switch strings.ToLower(platformType) {
	case "aws":
		region, _, _ := nestedString(infra, "status", "platformStatus", "aws", "region")
		return &storage.ProviderMetadata{
			Region:   region,
			Provider: &storage.ProviderMetadata_Aws{Aws: &storage.AWSProviderMetadata{}},
			Verified: true,
			Cluster:  clusterMeta,
		}
	case "gcp":
		region, _, _ := nestedString(infra, "status", "platformStatus", "gcp", "region")
		projectID, _, _ := nestedString(infra, "status", "platformStatus", "gcp", "projectID")
		return &storage.ProviderMetadata{
			Region: region,
			Provider: &storage.ProviderMetadata_Google{Google: &storage.GoogleProviderMetadata{
				Project: projectID,
			}},
			Verified: true,
			Cluster:  clusterMeta,
		}
	case "azure":
		return &storage.ProviderMetadata{
			Provider: &storage.ProviderMetadata_Azure{Azure: &storage.AzureProviderMetadata{}},
			Verified: true,
			Cluster:  clusterMeta,
		}
	default:
		return &storage.ProviderMetadata{Cluster: clusterMeta}
	}
}

// clusterTypeFromResourceTags extracts the cluster type from resource tags
// for the given platform type in an unstructured Infrastructure object.
func clusterTypeFromResourceTags(infra map[string]interface{}, platformType string) storage.ClusterMetadata_Type {
	platform := strings.ToLower(platformType)
	tagsPath := []string{"status", "platformStatus", platform, "resourceTags"}
	tags, _, _ := nestedSlice(infra, tagsPath...)
	for _, tag := range tags {
		if tm, ok := tag.(map[string]interface{}); ok {
			if key, _ := tm["key"].(string); key == redHatClusterTypeTagKey {
				if val, _ := tm["value"].(string); val != "" {
					return clusterMetadataTypeFromResourceTag(strings.ToLower(val))
				}
			}
		}
	}
	return storage.ClusterMetadata_OCP
}

// nestedString is a helper to extract a string from nested maps.
func nestedString(obj map[string]interface{}, fields ...string) (string, bool, error) {
	val, found, err := nestedField(obj, fields...)
	if !found || err != nil {
		return "", found, err
	}
	s, ok := val.(string)
	return s, ok, nil
}

func nestedSlice(obj map[string]interface{}, fields ...string) ([]interface{}, bool, error) {
	val, found, err := nestedField(obj, fields...)
	if !found || err != nil {
		return nil, found, err
	}
	s, ok := val.([]interface{})
	return s, ok, nil
}

func nestedField(obj map[string]interface{}, fields ...string) (interface{}, bool, error) {
	var val interface{} = obj
	for _, field := range fields {
		m, ok := val.(map[string]interface{})
		if !ok {
			return nil, false, nil
		}
		val, ok = m[field]
		if !ok {
			return nil, false, nil
		}
	}
	return val, true, nil
}

func clusterMetadataTypeFromResourceTag(tagValue string) storage.ClusterMetadata_Type {
	switch tagValue {
	case "osd":
		return storage.ClusterMetadata_OSD
	case "rosa":
		return storage.ClusterMetadata_ROSA
	case "aro":
		return storage.ClusterMetadata_ARO
	default:
		return storage.ClusterMetadata_OCP
	}
}
