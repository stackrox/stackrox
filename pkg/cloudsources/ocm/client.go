package ocm

import (
	"context"
	"errors"
	"strings"

	gogoProto "github.com/gogo/protobuf/types"
	sdkClient "github.com/openshift-online/ocm-sdk-go"
	accountsmgmtv1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
	pkgErrors "github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/cloudsources/discoveredclusters"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/urlfmt"
)

type ocmClient struct {
	conn          *sdkClient.Connection
	cloudSourceID string
}

// NewClient creates a client to interact with OCM APIs.
func NewClient(config *storage.CloudSource) (*ocmClient, error) {
	connection, err := sdkClient.NewConnectionBuilder().
		URL(urlfmt.FormatURL(config.GetOcm().GetEndpoint(), urlfmt.HTTPS, urlfmt.NoTrailingSlash)).
		Tokens(config.GetCredentials().GetSecret()).Agent(clientconn.GetUserAgent()).Build()

	if err != nil {
		return nil, pkgErrors.Wrap(err, "creating OCM connection")
	}

	return &ocmClient{
		conn:          connection,
		cloudSourceID: config.GetId(),
	}, nil
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
		"AND (plan.id IN ('ARO', 'OCP', 'MOA', 'OCP-AssistedInstall', 'MOA-HostedControlPlane', 'OSD', 'OSDTrial')) " +
		"AND (status IN  ('Active', 'Disconnected')) " +
		"AND (display_name ILIKE '%%' OR external_cluster_id ILIKE '%%' OR cluster_id ILIKE '%%')"

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
	var createClusterErrs error
	for _, sub := range subs {
		createdTime, err := gogoProto.TimestampProto(sub.CreatedAt())
		if err != nil {
			createClusterErrs = errors.Join(createClusterErrs, errox.InvariantViolation.New("converting timestamp").CausedBy(err))
			continue
		}
		clusters = append(clusters, &discoveredclusters.DiscoveredCluster{
			ID:                sub.ExternalClusterID(),
			Name:              sub.DisplayName(),
			Type:              getClusterMetadataType(sub),
			ProviderType:      getProviderType(sub),
			Region:            sub.RegionID(),
			FirstDiscoveredAt: createdTime,
			CloudSourceID:     c.cloudSourceID,
		})
	}

	if createClusterErrs != nil {
		return nil, createClusterErrs
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
	case "mao", "mao-hostedcontrolplane":
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
