package azure

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	cpUtils "github.com/stackrox/rox/pkg/cloudproviders/utils"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/k8sutil"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/utils"
	"google.golang.org/protobuf/proto"
	"k8s.io/client-go/kubernetes"
)

const (
	loggingRateLimiter  = "azure-metadata"
	aksClusterNameLabel = "kubernetes.azure.com/cluster"
)

type computeMetadata struct {
	Location       string `json:"location"`
	Zone           string `json:"zone"`
	SubscriptionID string `json:"subscriptionId"`
	VMID           string `json:"vmId"`
}

type azureInstanceMetadata struct {
	Compute *computeMetadata `json:"compute"`
}

var (
	log = logging.LoggerForModule()
)

// GetMetadata tries to obtain the Azure instance metadata.
// If not on Azure, returns nil, nil.
func GetMetadata(ctx context.Context) (*storage.ProviderMetadata, error) {
	req, err := http.NewRequest(http.MethodGet, "http://169.254.169.254/metadata/instance", nil)
	if err != nil {
		return nil, errors.Wrap(err, "could not create HTTP request")
	}
	req = req.WithContext(ctx)
	req.Header.Add("Metadata", "True")

	q := req.URL.Query()
	q.Add("format", "json")
	q.Add("api-version", "2018-04-02")
	req.URL.RawQuery = q.Encode()

	resp, err := metadataHTTPClient.Do(req)
	// Assume the service is unavailable if we encounter a transport error or a non-2xx status code
	if err != nil {
		return nil, nil
	}

	defer utils.IgnoreError(resp.Body.Close)

	if !httputil.Is2xxStatusCode(resp.StatusCode) {
		return nil, nil
	}

	contents, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "reading response body")
	}

	var metadata azureInstanceMetadata

	if err := json.Unmarshal(contents, &metadata); err != nil {
		return nil, errors.Wrap(err, "unmarshaling response")
	}

	attestedVMID, err := getAttestedVMID(ctx)
	if err != nil {
		log.Errorf("error getting attested VM ID: %v", err)
	}
	verified := attestedVMID != "" && attestedVMID == metadata.Compute.VMID

	clusterMetadata := getClusterMetadata(ctx, &metadata)

	apm := &storage.AzureProviderMetadata{}
	apm.SetSubscriptionId(metadata.Compute.SubscriptionID)
	pm := &storage.ProviderMetadata{}
	pm.SetRegion(metadata.Compute.Location)
	pm.SetZone(metadata.Compute.Zone)
	pm.SetAzure(proto.ValueOrDefault(apm))
	pm.SetVerified(verified)
	pm.SetCluster(clusterMetadata)
	return pm, nil
}

func getClusterMetadata(ctx context.Context, metadata *azureInstanceMetadata) *storage.ClusterMetadata {
	config, err := k8sutil.GetK8sInClusterConfig()
	if err != nil {
		logging.GetRateLimitedLogger().DebugL(loggingRateLimiter, "Obtaining in-cluster Kubernetes config: %s", err)
		return nil
	}
	k8sClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		logging.GetRateLimitedLogger().DebugL(loggingRateLimiter, "Creating Kubernetes clientset: %s", err)
		return nil
	}
	return getClusterMetadataFromNodeLabels(ctx, k8sClient, metadata)
}

func getClusterMetadataFromNodeLabels(ctx context.Context,
	k8sClient kubernetes.Interface, metadata *azureInstanceMetadata,
) *storage.ClusterMetadata {
	nodeLabels, err := cpUtils.GetAnyNodeLabels(ctx, k8sClient)
	if err != nil {
		logging.GetRateLimitedLogger().DebugL(loggingRateLimiter, "Failed to get node labels: %s", err)
		return nil
	}

	// The label is of the form "MC_<resource-group>_<cluster-name>_<location>.
	// Since both <resource-group> and <cluster-name> may contain underscores,
	// we cannot further separate the resource group and cluster name.
	if value, ok := nodeLabels[aksClusterNameLabel]; ok {
		clusterName := strings.TrimPrefix(value, "MC_")
		clusterName = strings.TrimSuffix(clusterName, fmt.Sprintf("_%s", metadata.Compute.Location))
		clusterID := fmt.Sprintf("%s_%s", metadata.Compute.SubscriptionID, value)
		cm := &storage.ClusterMetadata{}
		cm.SetType(storage.ClusterMetadata_AKS)
		cm.SetName(clusterName)
		cm.SetId(clusterID)
		return cm
	}
	return nil
}
