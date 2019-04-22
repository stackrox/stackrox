package gcp

import (
	"cloud.google.com/go/compute/metadata"
	"github.com/pkg/errors"
)

func getMetadataFromAPI(client *metadata.Client) (*gcpMetadata, error) {
	zone, err := client.Zone()
	if err != nil {
		return nil, errors.Wrap(err, "retrieving zone")
	}
	projectID, err := client.ProjectID()
	if err != nil {
		return nil, errors.Wrap(err, "retrieving project ID")
	}

	return &gcpMetadata{
		ProjectID: projectID,
		Zone:      zone,
	}, nil
}
