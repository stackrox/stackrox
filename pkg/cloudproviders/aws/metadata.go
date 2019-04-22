package aws

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
)

// GetMetadata tries to obtain the AWS instance metadata.
// If not on AWS, returns nil, nil.
func GetMetadata(ctx context.Context) (*storage.ProviderMetadata, error) {
	var verified bool

	errs := errorhelpers.NewErrorList("retrieving AWS EC2 metadata")
	doc, err := getIdentityDocFromPKCS7(ctx)
	errs.AddError(err)
	if doc != nil {
		verified = true
	} else {
		doc, err = getInstanceIdentityDocFromAPI(ctx)
		verified = false
		errs.AddError(err)
	}

	if doc == nil {
		return nil, errs.ToError()
	}

	return &storage.ProviderMetadata{
		Region: doc.Region,
		Zone:   doc.AvailabilityZone,
		Provider: &storage.ProviderMetadata_Aws{
			Aws: &storage.AWSProviderMetadata{
				AccountId: doc.AccountID,
			},
		},
		Verified: verified,
	}, nil
}
