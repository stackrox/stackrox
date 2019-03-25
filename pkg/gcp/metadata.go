package gcp

import (
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/compute/metadata"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
)

const timeout = 5 * time.Second

var (
	log = logging.LoggerForModule()
)

// GetMetadata returns the cluster metadata if on GCP or an error
// If not on GCP, then returns nil, nil
func GetMetadata() (*storage.ProviderMetadata, error) {
	if !metadata.OnGCE() {
		return nil, nil
	}

	client := &http.Client{
		Timeout: timeout,
	}
	c := metadata.NewClient(client)

	zone, err := c.Zone()
	if err != nil {
		return nil, err
	}

	var region string
	regionSlice := strings.Split(zone, "-")
	if len(regionSlice) > 1 {
		region = strings.Join(regionSlice[:len(regionSlice)-1], "-")
	}

	clusterName, err := c.InstanceAttributeValue("cluster-name")
	if err != nil {
		return nil, err
	}

	project, err := c.ProjectID()
	if err != nil {
		return nil, err
	}

	return &storage.ProviderMetadata{
		Region: region,
		Zone:   zone,
		Provider: &storage.ProviderMetadata_Google{
			Google: &storage.GoogleProviderMetadata{
				Project:     project,
				ClusterName: clusterName,
			},
		},
	}, nil
}
