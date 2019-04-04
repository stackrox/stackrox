package azure

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	timeout = 5 * time.Second
)

type azureComputeMetadata struct {
	Location       string
	Zone           string
	SubscriptionID string `json:"subscriptionId"`
}

type azureInstanceMetadata struct {
	Compute azureComputeMetadata
}

// GetMetadata tries to obtain the Azure instance metadata.
// If not on Azure, returns nil, nil.
func GetMetadata() (*storage.ProviderMetadata, error) {
	client := &http.Client{
		Timeout: timeout,
	}

	req, err := http.NewRequest(http.MethodGet, "http://169.254.169.254/metadata/instance", nil)
	if err != nil {
		return nil, errors.Wrap(err, "could not create HTTP request")
	}
	req.Header.Add("Metadata", "True")

	q := req.URL.Query()
	q.Add("format", "json")
	q.Add("api-version", "2018-04-02")
	req.URL.RawQuery = q.Encode()

	resp, err := client.Do(req)
	// Assume the service is unavailable if we encounter a transport error or a non-2xx status code
	if err != nil {
		return nil, nil
	}

	defer utils.IgnoreError(resp.Body.Close)

	if !httputil.Is2xxStatusCode(resp.StatusCode) {
		return nil, nil
	}

	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "reading response body")
	}

	var metadata azureInstanceMetadata

	if err := json.Unmarshal(contents, &metadata); err != nil {
		return nil, errors.Wrap(err, "unmarshaling response")
	}

	return &storage.ProviderMetadata{
		Region: metadata.Compute.Location,
		Zone:   metadata.Compute.Zone,
		Provider: &storage.ProviderMetadata_Azure{
			Azure: &storage.AzureProviderMetadata{
				SubscriptionId: metadata.Compute.SubscriptionID,
			},
		},
	}, nil
}
