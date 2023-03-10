package aws

import (
	"context"
	"encoding/base64"
	"encoding/json"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/fullsailor/pkcs7"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
)

// GetMetadata tries to obtain the AWS instance metadata.
// If not on AWS, returns nil, nil.
func GetMetadata(ctx context.Context) (*storage.ProviderMetadata, error) {
	errs := errorhelpers.NewErrorList("retrieving AWS EC2 metadata")

	sess, err := session.NewSession()
	if err != nil {
		errs.AddError(err)
		return nil, errs.ToError()
	}

	mdClient := ec2metadata.New(sess, &aws.Config{
		HTTPClient: httpClient,
	})
	if !mdClient.Available() {
		errs.AddError(errors.New("metadata service unavailable"))
		return nil, errs.ToError()
	}

	verified := true
	doc, err := identityDocFromPKCS7(ctx, mdClient)
	if err != nil {
		errs.AddError(err)
		verified = false

		doc, err = identityDocFromAPI(ctx, mdClient)
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

func identityDocFromPKCS7(ctx context.Context, mdClient *ec2metadata.EC2Metadata) (*ec2metadata.EC2InstanceIdentityDocument, error) {
	pkcsBase64, err := mdClient.GetDynamicDataWithContext(ctx, "instance-identity/pkcs7")
	if err != nil {
		return nil, errors.Wrap(err, "retrieving PKCS7 signature")
	}

	pkcs7Raw, err := base64.StdEncoding.DecodeString(pkcsBase64)
	if err != nil {
		return nil, err
	}

	pks7, err := pkcs7.Parse(pkcs7Raw)
	if err != nil {
		return nil, err
	}

	// It is probably possible to determine which certificate to use
	// based on the region returned by the metadata service,
	// but there is no harm in just checking all known certs.
	pks7.Certificates = awsCerts
	if err := pks7.Verify(); err != nil {
		return nil, errors.Wrap(err, "verifying PKCS7 signature")
	}

	doc := &ec2metadata.EC2InstanceIdentityDocument{}
	if err := json.Unmarshal(pks7.Content, &doc); err != nil {
		return nil, errors.Wrap(err, "unmarshaling instance identity document")
	}

	return doc, nil
}

func identityDocFromAPI(ctx context.Context, mdClient *ec2metadata.EC2Metadata) (*ec2metadata.EC2InstanceIdentityDocument, error) {
	doc := &ec2metadata.EC2InstanceIdentityDocument{}
	var err error
	*doc, err = mdClient.GetInstanceIdentityDocumentWithContext(ctx)
	if err != nil {
		return nil, err
	}

	return doc, nil
}
