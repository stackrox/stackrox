package azure

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/httputil"
	"github.com/stackrox/stackrox/pkg/logging"
	"github.com/stackrox/stackrox/pkg/utils"
)

type azureInstanceMetadata struct {
	Compute struct {
		Location       string `json:"location"`
		Zone           string `json:"zone"`
		SubscriptionID string `json:"subscriptionId"`
		VMID           string `json:"vmId"`
	} `json:"compute"`
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

	return &storage.ProviderMetadata{
		Region: metadata.Compute.Location,
		Zone:   metadata.Compute.Zone,
		Provider: &storage.ProviderMetadata_Azure{
			Azure: &storage.AzureProviderMetadata{
				SubscriptionId: metadata.Compute.SubscriptionID,
			},
		},
		Verified: verified,
	}, nil
}
