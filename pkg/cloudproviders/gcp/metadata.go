package gcp

import (
	"context"
	"errors"
	"strings"

	"cloud.google.com/go/compute/metadata"
	pkgErrors "github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
)

type gcpMetadata struct {
	ProjectID string
	Zone      string
}

var (
	log = logging.LoggerForModule()
)

func isNotDefinedError(err error) bool {
	_, ok := err.(metadata.NotDefinedError)
	return ok
}

// GetMetadata returns the cluster metadata if on GCP or an error
// If not on GCP, then returns nil, nil.
func GetMetadata(ctx context.Context) (*storage.ProviderMetadata, error) {
	// In offline mode we skip fetching instance metadata to suppress metadata.google.internal DNS lookup
	if env.OfflineModeEnv.BooleanSetting() {
		return nil, nil
	}

	if !metadata.OnGCE() {
		return nil, nil
	}

	c := metadata.NewClient(metadataHTTPClient)

	var verified bool
	var retrievalErrs error
	md, err := getMetadataFromIdentityToken(ctx)
	if err != nil {
		retrievalErrs = errors.Join(retrievalErrs, err)
	}
	if md != nil {
		verified = true
	} else {
		md, err = getMetadataFromAPI(c)
		verified = false
		if err != nil {
			retrievalErrs = errors.Join(retrievalErrs, err)
		}
	}

	if md == nil {
		return nil, pkgErrors.Wrap(retrievalErrs, "retrieving GCE metadata")
	}

	var region string
	regionSlice := strings.Split(md.Zone, "-")
	if len(regionSlice) > 1 {
		region = strings.Join(regionSlice[:len(regionSlice)-1], "-")
	}

	// clusterName only exists on GKE
	clusterName, err := c.InstanceAttributeValue("cluster-name")
	if err != nil && !isNotDefinedError(err) {
		return nil, err
	}
	clusterMetadata := getClusterMetadataFromAttributes(c)

	return &storage.ProviderMetadata{
		Region: region,
		Zone:   md.Zone,
		Provider: &storage.ProviderMetadata_Google{
			Google: &storage.GoogleProviderMetadata{
				Project:     md.ProjectID,
				ClusterName: clusterName,
			},
		},
		Verified: verified,
		Cluster:  clusterMetadata,
	}, nil
}

func getClusterMetadataFromAttributes(client *metadata.Client) *storage.ClusterMetadata {
	name, _ := client.InstanceAttributeValue("cluster-name")
	id, _ := client.InstanceAttributeValue("cluster-uid")
	if name != "" && id != "" {
		return &storage.ClusterMetadata{Type: storage.ClusterMetadata_GKE, Name: name, Id: id}
	}
	return nil
}
