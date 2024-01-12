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
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/k8sutil"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/utils"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const aksClusterNameLabel = "kubernetes.azure.com/cluster"

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

	return &storage.ProviderMetadata{
		Region: metadata.Compute.Location,
		Zone:   metadata.Compute.Zone,
		Provider: &storage.ProviderMetadata_Azure{
			Azure: &storage.AzureProviderMetadata{
				SubscriptionId: metadata.Compute.SubscriptionID,
			},
		},
		Verified: verified,
		Cluster:  clusterMetadata,
	}, nil
}

func getClusterMetadata(ctx context.Context, metadata *azureInstanceMetadata) *storage.ClusterMetadata {
	k8sClient, err := k8sutil.GetK8sInClusterClient()
	if err != nil {
		log.Error("Failed to create kubernetes client: ", err)
		return nil
	}
	return getClusterMetadataFromNodeLabels(ctx, k8sClient, metadata)
}

func getClusterMetadataFromNodeLabels(ctx context.Context,
	k8sClient kubernetes.Interface, metadata *azureInstanceMetadata,
) *storage.ClusterMetadata {
	nodeLabels, err := getAnyNodeLabels(ctx, k8sClient)
	if err != nil {
		log.Error("Failed to get node labels: ", err)
		return nil
	}

	// The label is of the form "MC_<resource-group>_<cluster-name>_<location>.
	// Since both <resource-group> and <cluster-name> may contain underscores,
	// we cannot further separate the resource group and cluster name.
	if clusterID, ok := nodeLabels[aksClusterNameLabel]; ok {
		clusterName := strings.TrimPrefix(clusterID, "MC_")
		clusterName = strings.TrimSuffix(clusterName, fmt.Sprintf("_%s", metadata.Compute.Location))
		return &storage.ClusterMetadata{Type: storage.ClusterMetadata_AKS, Name: clusterName, Id: clusterID}
	}
	return nil
}

// getAnyNodeLabels returns the labels of an arbitrary node. This is useful
// to extract global labels such as the cluster name.
func getAnyNodeLabels(ctx context.Context, client kubernetes.Interface) (map[string]string, error) {
	nodeList, err := client.CoreV1().Nodes().List(ctx, v1.ListOptions{Limit: 1})
	if err != nil {
		return nil, errors.Wrap(err, "listing nodes")
	}
	if nodeList.Size() == 0 {
		return nil, errors.Errorf("no nodes found: %v", err)
	}
	return nodeList.Items[0].GetLabels(), nil
}
