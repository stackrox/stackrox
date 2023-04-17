package aws

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/fullsailor/pkcs7"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/httputil/proxy"
	"github.com/stackrox/rox/pkg/logging"
)

const (
	timeout = 5 * time.Second
)

var (
	log = logging.LoggerForModule()

	httpClient = &http.Client{
		Timeout:   timeout,
		Transport: proxy.Without(),
	}
)

// GetMetadata tries to obtain the AWS instance metadata.
// If not on AWS, returns nil, nil.
func GetMetadata(ctx context.Context) (*storage.ProviderMetadata, error) {
	sess, err := session.NewSession()
	if err != nil {
		return nil, errors.Wrap(err, "creating AWS session")
	}

	mdClient := ec2metadata.New(sess, &aws.Config{
		HTTPClient: httpClient,
	})
	if !mdClient.Available() {
		return nil, nil
	}

	errs := errorhelpers.NewErrorList("retrieving AWS EC2 metadata")
	verified := true
	doc, err := signedIdentityDoc(ctx, mdClient)
	if err != nil {
		log.Warnf("Could not verify AWS public certificate: %v", err)
		errs.AddError(err)
		verified = false

		doc, err = plaintextIdentityDoc(ctx, mdClient)
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

func signedIdentityDoc(ctx context.Context, mdClient *ec2metadata.EC2Metadata) (*ec2metadata.EC2InstanceIdentityDocument, error) {
	// This endpoint returns PKCS #7 structured data.
	p7Base64, err := mdClient.GetDynamicDataWithContext(ctx, "instance-identity/rsa2048")
	if err != nil {
		return nil, errors.Wrap(err, "retrieving RSA-2048 signature")
	}

	p7Raw, err := base64.StdEncoding.DecodeString(p7Base64)
	if err != nil {
		return nil, err
	}

	p7, err := pkcs7.Parse(p7Raw)
	if err != nil {
		return nil, err
	}

	p7.Certificates = awsCerts
	if err := p7.Verify(); err != nil {
		return nil, errors.Wrap(err, "verifying RSA-2048 signature")
	}

	doc := &ec2metadata.EC2InstanceIdentityDocument{}
	if err := json.Unmarshal(p7.Content, doc); err != nil {
		return nil, errors.Wrap(err, "unmarshaling instance identity document")
	}

	return doc, nil
}

func plaintextIdentityDoc(ctx context.Context, mdClient *ec2metadata.EC2Metadata) (*ec2metadata.EC2InstanceIdentityDocument, error) {
	doc := &ec2metadata.EC2InstanceIdentityDocument{}
	var err error
	*doc, err = mdClient.GetInstanceIdentityDocumentWithContext(ctx)
	if err != nil {
		return nil, err
	}

	return doc, nil
}
