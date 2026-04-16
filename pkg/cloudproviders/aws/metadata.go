package aws

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
	"github.com/pkg/errors"
	"github.com/stackrox/pkcs7"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/cloudproviders/utils"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/httputil/proxy"
	"github.com/stackrox/rox/pkg/k8sutil"
	"github.com/stackrox/rox/pkg/logging"
	"k8s.io/client-go/kubernetes"
)

const (
	loggingRateLimiter  = "aws-metadata"
	timeout             = 5 * time.Second
	eksClusterNameLabel = "alpha.eksctl.io/cluster-name"
	eksClusterNameTag   = "eks:cluster-name"
	instanceTagsPath    = "/tags/instance"
)

var httpClient = &http.Client{
	Timeout:   timeout,
	Transport: proxy.Without(),
}

// GetMetadata tries to obtain the AWS instance metadata.
// If not on AWS, returns nil, nil.
func GetMetadata(ctx context.Context) (*storage.ProviderMetadata, error) {
	awsConfig, err := config.LoadDefaultConfig(ctx, config.WithHTTPClient(httpClient))
	if err != nil {
		return nil, errors.Wrap(err, "creating AWS config")
	}
	mdClient := imds.NewFromConfig(awsConfig)

	errs := errorhelpers.NewErrorList("retrieving AWS EC2 metadata")
	verified := true
	doc, err := signedIdentityDoc(ctx, mdClient)
	if err != nil {
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

func signedIdentityDoc(ctx context.Context, mdClient *imds.Client) (*imds.InstanceIdentityDocument, error) {
	output, err := mdClient.GetDynamicData(ctx, &imds.GetDynamicDataInput{Path: "/instance-identity/rsa2048"})
	if err != nil {
		return nil, errors.Wrap(err, "retrieving RSA-2048 signature")
	}

	reader := base64.NewDecoder(base64.StdEncoding, output.Content)
	p7Raw, err := io.ReadAll(reader)
	if err != nil {
		return nil, errors.Wrap(err, "reading RSA-2048 signature")
	}

	p7, err := pkcs7.Parse(p7Raw)
	if err != nil {
		return nil, errors.Wrap(err, "parsing RSA-2048 signature")
	}

	p7.Certificates = awsCerts
	if err := p7.Verify(); err != nil {
		return nil, errors.Wrap(err, "verifying RSA-2048 signature")
	}

	doc := &imds.InstanceIdentityDocument{}
	if err := json.Unmarshal(p7.Content, doc); err != nil {
		return nil, errors.Wrap(err, "unmarshaling instance identity document")
	}

	return doc, nil
}

func plaintextIdentityDoc(ctx context.Context, mdClient *imds.Client) (*imds.InstanceIdentityDocument, error) {
	output, err := mdClient.GetInstanceIdentityDocument(ctx, &imds.GetInstanceIdentityDocumentInput{})
	if err != nil {
		return nil, err
	}
	return &output.InstanceIdentityDocument, nil
}

// getClusterMetadata attempts to get the EKS cluster name on a best effort basis.
// First, it tries to get the cluster name from the EC2 instance tags. Access to
// the tags must be explicitly enabled for the EC2 instance beforehand.
// Second, it tries the node labels. The label is only set when the EKS cluster
// was created via eksctl.
func getClusterMetadata(ctx context.Context,
	client *imds.Client, doc *imds.InstanceIdentityDocument,
) *storage.ClusterMetadata {
	clusterName, err := getClusterNameFromInstanceTags(ctx, client)
	if err == nil {
		return clusterMetadataFromName(clusterName, doc)
	}
	logging.GetRateLimitedLogger().DebugL(loggingRateLimiter, "Failed to get EKS cluster metadata from instance tags: %s", err)

	config, err := k8sutil.GetK8sInClusterConfig()
	if err != nil {
		logging.GetRateLimitedLogger().DebugL(loggingRateLimiter, "Failed to get EKS cluster metadata: Obtaining in-cluster Kubernetes config: %s", err)
		return nil
	}
	k8sClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		logging.GetRateLimitedLogger().DebugL(loggingRateLimiter, "Failed to get EKS cluster metadata: Creating Kubernetes clientset: %s", err)
		return nil
	}
	clusterName, err = getClusterNameFromNodeLabels(ctx, k8sClient)
	if err != nil {
		logging.GetRateLimitedLogger().DebugL(loggingRateLimiter, "Failed to get EKS cluster metadata from node labels: %s", err)
		return nil
	}
	return clusterMetadataFromName(clusterName, doc)
}

func getClusterNameFromInstanceTags(ctx context.Context, client *imds.Client) (string, error) {
	output, err := client.GetMetadata(ctx, &imds.GetMetadataInput{Path: instanceTagsPath})
	if err != nil {
		return "", errors.Wrap(err, "getting cluster metadata")
	}
	clusterName, ok := output.ResultMetadata.Get(eksClusterNameTag).(string)
	if !ok {
		return "", errors.New("getting cluster name tag")
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

func clusterMetadataFromName(clusterName string, doc *imds.InstanceIdentityDocument,
) *storage.ClusterMetadata {
	clusterARN := fmt.Sprintf("arn:aws:eks:%s:%s:cluster/%s", doc.Region, doc.AccountID, clusterName)
	return &storage.ClusterMetadata{Type: storage.ClusterMetadata_EKS, Name: clusterName, Id: clusterARN}
}
