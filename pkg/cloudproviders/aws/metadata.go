package aws

import (
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/stackrox/rox/generated/storage"
)

const (
	timeout = 5 * time.Second
)

// GetMetadata tries to obtain the AWS instance metadata.
// If not on AWS, returns nil, nil.
func GetMetadata() (*storage.ProviderMetadata, error) {
	sess, err := session.NewSession()
	if err != nil {
		return nil, err
	}

	mdClient := ec2metadata.New(sess, &aws.Config{
		HTTPClient: &http.Client{
			Timeout: timeout,
		},
	})
	if !mdClient.Available() {
		return nil, nil
	}

	identityDoc, err := mdClient.GetInstanceIdentityDocument()
	if err != nil {
		return nil, err
	}

	return &storage.ProviderMetadata{
		Region: identityDoc.Region,
		Zone:   identityDoc.AvailabilityZone,
		Provider: &storage.ProviderMetadata_Aws{
			Aws: &storage.AWSProviderMetadata{
				AccountId: identityDoc.AccountID,
			},
		},
	}, nil
}
