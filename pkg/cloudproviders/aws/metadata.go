package aws

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/fullsailor/pkcs7"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/cloudproviders/utils"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/httputil/proxy"
	"github.com/stackrox/rox/pkg/k8sutil"
	"github.com/stackrox/rox/pkg/logging"
	"k8s.io/client-go/kubernetes"
)

const (
	timeout             = 5 * time.Second
	eksClusterNameLabel = "alpha.eksctl.io/cluster-name"
	eksClusterNameTag   = "eks:cluster-name"
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

	clusterMetadata := getClusterMetadata(ctx, mdClient, doc)

	return &storage.ProviderMetadata{
		Region: doc.Region,
		Zone:   doc.AvailabilityZone,
		Provider: &storage.ProviderMetadata_Aws{
			Aws: &storage.AWSProviderMetadata{
				AccountId: doc.AccountID,
			},
		},
		Verified: verified,
		Cluster:  clusterMetadata,
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

// getClusterMetadata attempts to get the EKS cluster name on a best effort basis.
// First, it tries to get the cluster name from the EC2 instance tags. Access to
// the tags must be explicitly enabled for the EC2 instance beforehand.
// Second, it tries the node labels. The label is only set when the EKS cluster
// was created via eksctl.
func getClusterMetadata(ctx context.Context,
	client *ec2metadata.EC2Metadata, doc *ec2metadata.EC2InstanceIdentityDocument,
) *storage.ClusterMetadata {
	clusterName, err := getClusterNameFromInstanceTags(ctx, client)
	if err == nil {
		return clusterMetadataFromName(clusterName, doc)
	}
	log.Errorf("Failed to get EKS cluster metadata from instance tags: %v", err)

	config, err := k8sutil.GetK8sInClusterConfig()
	if err != nil {
		log.Errorf("Obtaining in-cluster Kubernetes config: %v", err)
		return nil
	}
	k8sClient := k8sutil.MustCreateK8sClient(config)
	clusterName, err = getClusterNameFromNodeLabels(ctx, k8sClient)
	if err == nil {
		return clusterMetadataFromName(clusterName, doc)
	}
	log.Errorf("Failed to get EKS cluster metadata from node labels: %v", err)
	return nil
}

func getClusterNameFromInstanceTags(ctx context.Context, client *ec2metadata.EC2Metadata) (string, error) {
	clusterName, err := client.GetMetadataWithContext(ctx, fmt.Sprintf("/tags/instance/%s", eksClusterNameTag))
	if err != nil {
		return "", errors.Wrap(err, "getting cluster name tag")
	}
	return clusterName, nil
}

func getClusterNameFromNodeLabels(ctx context.Context, k8sClient kubernetes.Interface) (string, error) {
	nodeLabels, err := utils.GetAnyNodeLabels(ctx, k8sClient)
	if err != nil {
		return "", errors.Wrap(err, "getting node labels")
	}
	if clusterName := nodeLabels[eksClusterNameLabel]; clusterName != "" {
		return clusterName, nil
	}
	return "", errors.Errorf("node label %q not found", eksClusterNameLabel)
}

func clusterMetadataFromName(clusterName string, doc *ec2metadata.EC2InstanceIdentityDocument,
) *storage.ClusterMetadata {
	clusterARN := fmt.Sprintf("arn:aws:eks:%s:%s:cluster/%s", doc.Region, doc.AccountID, clusterName)
	return &storage.ClusterMetadata{Type: storage.ClusterMetadata_EKS, Name: clusterName, Id: clusterARN}
}
