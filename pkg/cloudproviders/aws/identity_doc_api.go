package aws

import (
	"context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
)

func getInstanceIdentityDocFromAPI(ctx context.Context) (*ec2metadata.EC2InstanceIdentityDocument, error) {
	sess, err := session.NewSession()
	if err != nil {
		return nil, err
	}

	mdClient := ec2metadata.New(sess, &aws.Config{
		HTTPClient: httpClient,
	})
	if !mdClient.Available() {
		return nil, nil
	}

	identityDoc, err := mdClient.GetInstanceIdentityDocument()
	if err != nil {
		return nil, err
	}

	return &identityDoc, nil
}
