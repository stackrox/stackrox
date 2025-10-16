package clusterstatus

import (
	"context"
	"strings"

	configv1 "github.com/openshift/api/config/v1"
	configVersioned "github.com/openshift/client-go/config/clientset/versioned"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"google.golang.org/protobuf/proto"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// Each resource tag for different cloud providers will have this set in case special "flavors" for installing
	// OpenShift were used (e.g. OSD, ARO, ROSA).
	redHatClusterTypeTagKey = "red-hat-clustertype"
)

type providerMetadataFromOpenShift = func(ctx context.Context, p configVersioned.Interface) (*storage.ProviderMetadata, error)

func getProviderMetadataFromOpenShiftConfig(ctx context.Context,
	client configVersioned.Interface) (*storage.ProviderMetadata, error) {
	infraCR, err := client.ConfigV1().Infrastructures().Get(ctx, "cluster", metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "retrieving cluster infrastructure CR")
	}

	clusterVersionCR, err := client.ConfigV1().ClusterVersions().Get(ctx, "version", metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "retrieving cluster version CR")
	}

	return openShiftCRsToProviderMetadata(infraCR, clusterVersionCR), nil
}

func openShiftCRsToProviderMetadata(infra *configv1.Infrastructure,
	clusterVersion *configv1.ClusterVersion) *storage.ProviderMetadata {
	// The platform status is required to read out the provider specific information. If it is unset,
	// we can short-circuit here.
	if infra.Status.PlatformStatus == nil {
		return nil
	}

	switch infra.Status.PlatformStatus.Type {
	case configv1.AWSPlatformType:
		cm := &storage.ClusterMetadata{}
		cm.SetType(clusterTypeFromAWSResourceTags(infra.Status.PlatformStatus.AWS.ResourceTags))
		cm.SetName(infra.Status.InfrastructureName)
		cm.SetId(string(clusterVersion.Spec.ClusterID))
		pm := &storage.ProviderMetadata{}
		pm.SetRegion(infra.Status.PlatformStatus.AWS.Region)
		pm.SetAws(&storage.AWSProviderMetadata{})
		pm.SetVerified(true)
		pm.SetCluster(cm)
		return pm
	case configv1.GCPPlatformType:
		gpm := &storage.GoogleProviderMetadata{}
		gpm.SetProject(infra.Status.PlatformStatus.GCP.ProjectID)
		cm := &storage.ClusterMetadata{}
		cm.SetType(clusterTypeFromGCPResourceTags(infra.Status.PlatformStatus.GCP.ResourceTags))
		cm.SetName(infra.Status.InfrastructureName)
		cm.SetId(string(clusterVersion.Spec.ClusterID))
		pm := &storage.ProviderMetadata{}
		pm.SetRegion(infra.Status.PlatformStatus.GCP.Region)
		pm.SetGoogle(proto.ValueOrDefault(gpm))
		pm.SetVerified(true)
		pm.SetCluster(cm)
		return pm
	case configv1.AzurePlatformType:
		cm := &storage.ClusterMetadata{}
		cm.SetType(clusterTypeFromAzureResourceTags(infra.Status.PlatformStatus.Azure.ResourceTags))
		cm.SetName(infra.Status.InfrastructureName)
		cm.SetId(string(clusterVersion.Spec.ClusterID))
		pm := &storage.ProviderMetadata{}
		pm.SetRegion("")
		pm.SetAzure(&storage.AzureProviderMetadata{})
		pm.SetVerified(true)
		pm.SetCluster(cm)
		return pm
	default:
		cm := &storage.ClusterMetadata{}
		cm.SetType(storage.ClusterMetadata_OCP)
		cm.SetName(infra.Status.InfrastructureName)
		cm.SetId(string(clusterVersion.Spec.ClusterID))
		pm := &storage.ProviderMetadata{}
		pm.SetCluster(cm)
		return pm
	}
}

func clusterTypeFromAWSResourceTags(tags []configv1.AWSResourceTag) storage.ClusterMetadata_Type {
	var clusterType string
	for _, tag := range tags {
		if tag.Key == redHatClusterTypeTagKey {
			clusterType = tag.Value
		}
	}
	return clusterMetadataTypeFromResourceTag(strings.ToLower(clusterType))
}

func clusterTypeFromGCPResourceTags(tags []configv1.GCPResourceTag) storage.ClusterMetadata_Type {
	var clusterType string
	for _, tag := range tags {
		if tag.Key == redHatClusterTypeTagKey {
			clusterType = tag.Value
		}
	}
	return clusterMetadataTypeFromResourceTag(strings.ToLower(clusterType))
}

func clusterTypeFromAzureResourceTags(tags []configv1.AzureResourceTag) storage.ClusterMetadata_Type {
	var clusterType string
	for _, tag := range tags {
		if tag.Key == redHatClusterTypeTagKey {
			clusterType = tag.Value
		}
	}
	return clusterMetadataTypeFromResourceTag(strings.ToLower(clusterType))
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
