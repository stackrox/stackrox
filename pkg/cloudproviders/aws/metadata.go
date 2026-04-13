package aws

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/pkcs7"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/cloudproviders/utils"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/httputil/proxy"
	"github.com/stackrox/rox/pkg/k8sutil"
	"github.com/stackrox/rox/pkg/logging"
	"k8s.io/client-go/dynamic"
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

// GetMetadata tries to obtain the AWS instance metadata via IMDS.
// Uses a lightweight HTTP client instead of the AWS SDK.
// If not on AWS, returns nil, nil.
func GetMetadata(ctx context.Context) (*storage.ProviderMetadata, error) {
	mdClient := NewIMDSClient(httpClient)
	mdClient.GetToken(ctx)

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

func signedIdentityDoc(ctx context.Context, mdClient *IMDSClient) (*InstanceIdentityDocument, error) {
	rawSig, err := mdClient.GetSignedIdentityDocument(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "retrieving RSA-2048 signature")
	}

	p7Raw, err := base64.StdEncoding.DecodeString(string(rawSig))
	if err != nil {
		return nil, errors.Wrap(err, "decoding RSA-2048 signature")
	}

	p7, err := pkcs7.Parse(p7Raw)
	if err != nil {
		return nil, errors.Wrap(err, "parsing RSA-2048 signature")
	}

	p7.Certificates = awsCerts()
	if err := p7.Verify(); err != nil {
		return nil, errors.Wrap(err, "verifying RSA-2048 signature")
	}

	doc := &InstanceIdentityDocument{}
	if err := json.Unmarshal(p7.Content, doc); err != nil {
		return nil, errors.Wrap(err, "unmarshaling instance identity document")
	}

	return doc, nil
}

func plaintextIdentityDoc(ctx context.Context, mdClient *IMDSClient) (*InstanceIdentityDocument, error) {
	return mdClient.GetIdentityDocument(ctx)
}

func getClusterMetadata(ctx context.Context,
	mdClient *IMDSClient, doc *InstanceIdentityDocument,
) *storage.ClusterMetadata {
	clusterName, err := getClusterNameFromInstanceTags(ctx, mdClient)
	if err == nil {
		return clusterMetadataFromName(clusterName, doc)
	}
	logging.GetRateLimitedLogger().DebugL(loggingRateLimiter, "Failed to get EKS cluster metadata from instance tags: %s", err)

	config, err := k8sutil.GetK8sInClusterConfig()
	if err != nil {
		logging.GetRateLimitedLogger().DebugL(loggingRateLimiter, "Failed to get EKS cluster metadata: Obtaining in-cluster Kubernetes config: %s", err)
		return nil
	}
	dynClient, err := dynamic.NewForConfig(config)
	if err != nil {
		logging.GetRateLimitedLogger().DebugL(loggingRateLimiter, "Failed to get EKS cluster metadata: Creating dynamic Kubernetes client: %s", err)
		return nil
	}
	clusterName, err = getClusterNameFromNodeLabels(ctx, dynClient)
	if err != nil {
		logging.GetRateLimitedLogger().DebugL(loggingRateLimiter, "Failed to get EKS cluster metadata from node labels: %s", err)
		return nil
	}
	return clusterMetadataFromName(clusterName, doc)
}

func getClusterNameFromInstanceTags(ctx context.Context, mdClient *IMDSClient) (string, error) {
	// Instance tags are at /latest/meta-data/tags/instance/<tag-key>
	tagValue, err := mdClient.GetMetadata(ctx, instanceTagsPath+"/"+eksClusterNameTag)
	if err != nil {
		return "", errors.Wrap(err, "getting cluster name tag")
	}
	if tagValue == "" {
		return "", errors.New("empty cluster name tag")
	}
	return tagValue, nil
}

func getClusterNameFromNodeLabels(ctx context.Context, dynClient dynamic.Interface) (string, error) {
	nodeLabels, err := utils.GetAnyNodeLabels(ctx, dynClient)
	if err != nil {
		return "", errors.Wrap(err, "getting node labels")
	}
	if clusterName := nodeLabels[eksClusterNameLabel]; clusterName != "" {
		return clusterName, nil
	}
	return "", errors.Errorf("node label %q not found", eksClusterNameLabel)
}

func clusterMetadataFromName(clusterName string, doc *InstanceIdentityDocument,
) *storage.ClusterMetadata {
	clusterARN := fmt.Sprintf("arn:aws:eks:%s:%s:cluster/%s", doc.Region, doc.AccountID, clusterName)
	return &storage.ClusterMetadata{Type: storage.ClusterMetadata_EKS, Name: clusterName, Id: clusterARN}
}
