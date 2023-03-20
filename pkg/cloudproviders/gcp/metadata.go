package gcp

import (
	"context"
	"strings"

	"cloud.google.com/go/compute/metadata"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/errorhelpers"
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
		log.Error("[GCP GetMetadata] offline mode skipping")
		return nil, nil
	}

	if !metadata.OnGCE() {
		log.Error("[GCP GetMetadata] not on GCE")
		return nil, nil
	}

	c := metadata.NewClient(metadataHTTPClient)

	var verified bool
	errs := errorhelpers.NewErrorList("retrieving GCE metadata")
	md, err := getMetadataFromIdentityToken(ctx)
	errs.AddError(err)
	if md != nil {
		verified = true
	} else {
		log.Error("[GCP GetMetadata] md from identity is nil")
		md, err = getMetadataFromAPI(c)
		verified = false
		log.Errorf("[GCP GetMetadata] err getMetadataFromAPI %+q", err)
		errs.AddError(err)
	}

	if md == nil {
		log.Errorf("[GCP GetMetadata] all errs %+q", errs.ToError())
		return nil, errs.ToError()
	}

	log.Errorf("[GCP GetMetadata] gcp metadata %+q", *md)

	var region string
	regionSlice := strings.Split(md.Zone, "-")
	if len(regionSlice) > 1 {
		region = strings.Join(regionSlice[:len(regionSlice)-1], "-")
	}

	// clusterName only exists on GKE
	clusterName, err := c.InstanceAttributeValue("cluster-name")
	if err != nil && !isNotDefinedError(err) {
		log.Errorf("[GCP GetMetadata] err InstanceAttributeValue %+q", err)
		return nil, err
	}
	log.Errorf("[GCP GetMetadata] clusterName %+q", clusterName)

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
	}, nil
}
