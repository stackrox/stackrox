package ocm

import (
	"context"
	"strings"
	"time"

	sdkClient "github.com/openshift-online/ocm-sdk-go"
	accountsmgmtv1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
	pkgErrors "github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/cloudsources/discoveredclusters"
	"github.com/stackrox/rox/pkg/cloudsources/opts"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/urlfmt"
)

type ocmClient struct {
	conn          *sdkClient.Connection
	cloudSourceID string
}

// NewClient creates a client to interact with OCM APIs.
func NewClient(ctx context.Context, config *storage.CloudSource, options ...opts.ClientOpts) (*ocmClient, error) {
	opt := opts.DefaultOpts()
	for _, option := range options {
		option(opt)
	}

	connection, err := sdkClient.NewConnectionBuilder().
		RetryLimit(opt.Retries).
		URL(urlfmt.FormatURL(config.GetOcm().GetEndpoint(), urlfmt.HTTPS, urlfmt.NoTrailingSlash)).
		Client(config.GetCredentials().GetClientId(), config.GetCredentials().GetClientSecret()).
		Tokens(config.GetCredentials().GetSecret()).Agent(clientconn.GetUserAgent()).BuildContext(ctx)

	if err != nil {
		return nil, pkgErrors.Wrap(err, "creating OCM connection")
	}

	return &ocmClient{
		conn:          connection,
		cloudSourceID: config.GetId(),
	}, nil
}

func (c *ocmClient) Ping(ctx context.Context) error {
	// For the ocm-sdk, it is currently not possible to set the request timeout on the created client, instead it
	// needs to be set on the given context passed to the request. This has the disadvantage of creating
	// "context cancelled" error messages in case of hitting a timeout while reaching the endpoint.
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	_, err := c.conn.AccountsMgmt().V1().CurrentAccount().Get().SendContext(ctx)
	return err
}

func (c *ocmClient) GetDiscoveredClusters(ctx context.Context) ([]*discoveredclusters.DiscoveredCluster, error) {
	var (
		subs  []*accountsmgmtv1.Subscription
		total int
		page  = 1
	)

	// Taken from console.redhat.com/openshift.
	// Filter for all subscriptions which have:
	//	- a cluster associated with it
	// 	- the cluster is a valid OpenShift plan
	//	- the status is "Active", "Disconnected"
	//  - name / cluster_id / external_cluster_id are given
	subscriptionSearch := "(cluster_id!='') " +
		"AND (external_cluster_id!='') " +
		"AND (plan.id IN ('ARO', 'OCP', 'MOA', 'OCP-AssistedInstall', 'MOA-HostedControlPlane', 'OSD', 'OSDTrial')) " +
		"AND (status IN  ('Active'))"

	for {
		// As an alternative, there's also the clustermgmt API. However, during testing the subscription API
		// performed better on queries. The console also favors the subscription API for creating the list view.
		resp, err := c.conn.AccountsMgmt().V1().Subscriptions().List().Size(100).Page(page).Search(subscriptionSearch).SendContext(ctx)
		if err != nil {
			return nil, pkgErrors.Wrap(err, "retrieving cluster subscriptions")
		}
		total = resp.Total()
		subs = append(subs, resp.Items().Slice()...)
		if len(subs) == total {
			// We've fetched all clusters.
			break
		}
		page++
	}

	return c.mapToDiscoveredClusters(subs)
}

func (c *ocmClient) mapToDiscoveredClusters(subs []*accountsmgmtv1.Subscription) ([]*discoveredclusters.DiscoveredCluster, error) {
	clusters := make([]*discoveredclusters.DiscoveredCluster, 0, len(subs))
	clusterIDs := set.NewStringSet()
	for _, sub := range subs {
		createdTime := sub.CreatedAt()
		// We've seen duplicates being returned from the API and the search query doesn't seem to support DISTINCT
		// or unique as key-value.
		if clusterIDs.Contains(sub.ExternalClusterID()) {
			continue
		}
		clusters = append(clusters, &discoveredclusters.DiscoveredCluster{
			ID:                sub.ExternalClusterID(),
			Name:              sub.DisplayName(),
			Type:              getClusterMetadataType(sub),
			ProviderType:      getProviderType(sub),
			Region:            sub.RegionID(),
			FirstDiscoveredAt: &createdTime,
			CloudSourceID:     c.cloudSourceID,
		})
		clusterIDs.Add(sub.ExternalClusterID())
	}

	return clusters, nil
}

func getClusterMetadataType(sub *accountsmgmtv1.Subscription) storage.ClusterMetadata_Type {
	switch strings.ToLower(sub.Plan().Type()) {
	case "ocp":
		return storage.ClusterMetadata_OCP
	case "osd":
		return storage.ClusterMetadata_OSD
	case "aro":
		return storage.ClusterMetadata_ARO
	case "moa", "moa-hostedcontrolplane":
		return storage.ClusterMetadata_ROSA
	default:
		return storage.ClusterMetadata_UNSPECIFIED
	}
}

func getProviderType(sub *accountsmgmtv1.Subscription) storage.DiscoveredCluster_Metadata_ProviderType {
	switch strings.ToLower(sub.CloudProviderID()) {
	case "gcp":
		return storage.DiscoveredCluster_Metadata_PROVIDER_TYPE_GCP
	case "aws":
		return storage.DiscoveredCluster_Metadata_PROVIDER_TYPE_AWS
	case "azure":
		return storage.DiscoveredCluster_Metadata_PROVIDER_TYPE_AZURE
	default:
		// For older clusters, the cloud provider ID may not be specified. In some cases we can infer the provider type
		// from the cluster type.
		clusterType := getClusterMetadataType(sub)
		if clusterType == storage.ClusterMetadata_ARO {
			return storage.DiscoveredCluster_Metadata_PROVIDER_TYPE_AZURE
		}
		if clusterType == storage.ClusterMetadata_ROSA {
			return storage.DiscoveredCluster_Metadata_PROVIDER_TYPE_AWS
		}
		return storage.DiscoveredCluster_Metadata_PROVIDER_TYPE_UNSPECIFIED
	}
}
