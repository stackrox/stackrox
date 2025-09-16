package paladin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	pgkErrors "github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/administration/events/codes"
	"github.com/stackrox/rox/pkg/administration/events/option"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/cloudsources/discoveredclusters"
	"github.com/stackrox/rox/pkg/cloudsources/opts"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/httputil/proxy"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/urlfmt"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	// Time format for Paladin API timestamps.
	timeFormat = "2006-01-02 15:04:00+0000"

	clusterTypeAKS = "aks"
	clusterTypeGKE = "gke"
	clusterTypeEKS = "eks"
)

var log = logging.LoggerForModule(option.EnableAdministrationEvents())

// AssetsResponse holds the response returned by the Paladin Cloud API.
type AssetsResponse struct {
	Assets []Asset `json:"assets,omitempty"`
}

// Asset holds the asset as returned by the Paladin Cloud API.
type Asset struct {
	ID                string    `json:"id"`
	Name              string    `json:"name"`
	Type              string    `json:"type"`
	Source            string    `json:"source"`
	Region            string    `json:"region"`
	FirstDiscoveredAt time.Time `json:"firstDiscoveryDate"`
}

// UnmarshalJSON ummarshals Paladin Cloud API responses to the Asset struct.
// Paladin API uses a custom time format which leads to errors when unmarshalling JSON without
// customizations.
// Hence, need to use an intermediary struct, parse the time according to the layout, and then
// fill the asset struct accordingly.
func (a *Asset) UnmarshalJSON(b []byte) error {
	var intermediary struct {
		ID                string `json:"id"`
		Name              string `json:"name"`
		Type              string `json:"type"`
		Source            string `json:"source"`
		Region            string `json:"region"`
		FirstDiscoveredAt string `json:"firstDiscoveryDate"`
	}
	if err := json.Unmarshal(b, &intermediary); err != nil {
		return err
	}

	discoveredAt, err := time.Parse(timeFormat, intermediary.FirstDiscoveredAt)
	if err != nil {
		return err
	}

	*a = Asset{
		ID:                intermediary.ID,
		Name:              intermediary.Name,
		Type:              intermediary.Type,
		Source:            intermediary.Source,
		Region:            intermediary.Region,
		FirstDiscoveredAt: discoveredAt,
	}
	return nil
}

// GetName returns the name of the asset.
// In case no name is given, the name will be attempted to be parsed from the asset ID.
// In case no name can be parsed from the asset ID, the ID will be returned.
func (a *Asset) GetName() string {
	if a.Name != "" {
		return a.Name
	}

	// By default, in case we cannot retrieve the name from the ID (e.g. in case GKE clusters are given), return
	// the cluster ID as name.
	name := a.ID

	// For AKS and EKS clusters, the name is embedded within the cluster ID. It will be the last part of the ID
	// when split by "/"
	// AKS ID: "subscriptions/<id>/resourcegroups/<group>/azure resource names.../<cluster name>"
	// EKS ID: "arn:aws:eks:<region>:<account-id>:cluster/<cluster name>
	idParts := strings.Split(a.ID, "/")
	idPartsLen := len(idParts)
	if len(idParts) > 1 {
		name = idParts[idPartsLen-1]
	}
	return name
}

// GetID returns the ID of the asset compatible with the expected ID of secured clusters in ACS.
// For AKS, this requires parsing the ID and formatting it according to the structure defined in
// pkg/cloudproviders/azure.
func (a *Asset) GetID() string {
	// If the type is not AKS, we can safely return the ID from Paladin Cloud.
	// In the case of EKS, it will be the ARN, in the case of GCP it will be the cluster ID.
	if a.Type != clusterTypeAKS {
		return a.ID
	}

	// For Azure, our cluster ID is different from the one specified by Paladin Cloud.
	// Paladin CLoud has the Azure reference, which is of the format
	//	"subscriptions/<id>/resourcegroups/<group>/azure resource names.../<cluster name>"
	// Internally, our format for AKS clusters is
	//  "<subscription_id>_MC_<resource group>_<cluster name>_<location>

	// Index 1 -> subscription ID
	// Index 3 -> resource group name
	// Index 7 -> cluster name
	idParts := strings.Split(a.ID, "/")

	if len(idParts) != 8 {
		log.Warnf("Received unknown ID for AKS cluster from Paladin %q. This might lead to incorrect matching",
			a.ID)
		return a.ID
	}

	return fmt.Sprintf("%s_MC_%s_%s_%s", idParts[1], idParts[3], idParts[7], a.Region)
}

// GetType returns the asset type converted to storage.ClusterMetadata_Type.
func (a *Asset) GetType() storage.ClusterMetadata_Type {
	switch a.Type {
	case clusterTypeAKS:
		return storage.ClusterMetadata_AKS
	case clusterTypeGKE:
		return storage.ClusterMetadata_GKE
	case clusterTypeEKS:
		return storage.ClusterMetadata_EKS
	default:
		return storage.ClusterMetadata_UNSPECIFIED
	}
}

// GetProviderType returns the asset source converted to storage.DiscoveredCluster_Metadata_ProviderType.
func (a *Asset) GetProviderType() storage.DiscoveredCluster_Metadata_ProviderType {
	switch a.Source {
	case "gcp":
		return storage.DiscoveredCluster_Metadata_PROVIDER_TYPE_GCP
	case "aws":
		return storage.DiscoveredCluster_Metadata_PROVIDER_TYPE_AWS
	case "azure":
		return storage.DiscoveredCluster_Metadata_PROVIDER_TYPE_AZURE
	default:
		return storage.DiscoveredCluster_Metadata_PROVIDER_TYPE_UNSPECIFIED
	}
}

// paladinClient can be used to interact with the Paladin Cloud API.
type paladinClient struct {
	httpClient      *http.Client
	endpoint        string
	cloudSourceID   string
	cloudSourceName string
}

// paladinTransportWrapper adds auth information to the underlying transport as well as the user agent.
type paladinTransportWrapper struct {
	baseTransport http.RoundTripper
	token         string
}

func (p *paladinTransportWrapper) RoundTrip(req *http.Request) (*http.Response, error) {
	// The Paladin Cloud API expects the Authorization header in the format '"Authorization:" <TOKEN>'
	// instead of e.g. Bearer token format.
	req.Header.Add("Authorization", p.token)
	req.Header.Add("User-Agent", clientconn.GetUserAgent())
	return p.baseTransport.RoundTrip(req)
}

// NewClient creates a client to interact with Paladin Cloud APIs.
func NewClient(cfg *storage.CloudSource, options ...opts.ClientOpts) *paladinClient {
	opt := opts.DefaultOpts()
	for _, option := range options {
		option(opt)
	}

	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = opt.Retries
	retryClient.Logger = nil
	retryClient.HTTPClient.Transport = &paladinTransportWrapper{
		baseTransport: proxy.RoundTripper(),
		token:         cfg.GetCredentials().GetSecret(),
	}
	retryClient.HTTPClient.Timeout = opt.Timeout
	retryClient.RetryWaitMin = 10 * time.Second

	return &paladinClient{
		httpClient:      retryClient.StandardClient(),
		endpoint:        urlfmt.FormatURL(cfg.GetPaladinCloud().GetEndpoint(), urlfmt.HTTPS, urlfmt.NoTrailingSlash),
		cloudSourceID:   cfg.GetId(),
		cloudSourceName: cfg.GetName(),
	}
}

func (c *paladinClient) Ping(ctx context.Context) error {
	// At the current time, no better API is known besides the asset API to confirm correct authN/Z setup and
	// connectivity. In case we find a better API, we may switch to that.
	if _, err := c.getAssets(ctx); err != nil {
		log.Errorw("PaladinCloud: retrieving data",
			logging.Err(err),
			logging.ErrCode(codes.PaladinCloudGeneric),
			logging.CloudSourceName(c.cloudSourceName),
		)
		return err
	}
	return nil
}

// GetDiscoveredClusters returns the discovered clusters from the Paladin Cloud API.
func (c *paladinClient) GetDiscoveredClusters(ctx context.Context) ([]*discoveredclusters.DiscoveredCluster, error) {
	response, err := c.getAssets(ctx)
	if err != nil {
		log.Errorw("PaladinCloud: retrieving data",
			logging.Err(err),
			logging.ErrCode(codes.PaladinCloudGeneric),
			logging.CloudSourceName(c.cloudSourceName),
		)
		return nil, pgkErrors.Wrap(err, "retrieving data from paladin cloud")
	}

	var transformErrors error
	discoveredClusters := make([]*discoveredclusters.DiscoveredCluster, 0, len(response.Assets))
	for _, asset := range response.Assets {
		discoveredCluster, err := c.mapAssetToDiscoveredCluster(asset)
		if err != nil {
			transformErrors = errors.Join(transformErrors, err)
			continue
		}
		discoveredClusters = append(discoveredClusters, discoveredCluster)
	}

	if transformErrors != nil {
		log.Errorw("PaladinCloud: transforming assets",
			logging.Err(transformErrors),
			logging.ErrCode(codes.PaladinCloudGeneric),
			logging.CloudSourceName(c.cloudSourceName),
		)
		return nil, transformErrors
	}

	return discoveredClusters, nil
}

// getAssets returns the discovered assets from Paladin Cloud.
func (c *paladinClient) getAssets(ctx context.Context) (*AssetsResponse, error) {
	var assets AssetsResponse

	if err := c.sendRequest(ctx, http.MethodGet, "/v2/assets", "?category=k8s", &assets); err != nil {
		return nil, pgkErrors.Wrap(err, "retrieving assets")
	}

	return &assets, nil
}

func (c *paladinClient) sendRequest(ctx context.Context, method string, apiPath string, query string, response interface{}) error {
	path, err := url.JoinPath(c.endpoint, apiPath)
	if err != nil {
		return errox.InvalidArgs.CausedBy(err)
	}

	req, err := http.NewRequestWithContext(ctx, method, path+query, nil)
	if err != nil {
		return pgkErrors.Wrap(err, "creating request")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return pgkErrors.Wrap(err, "executing request")
	}

	defer utils.IgnoreError(resp.Body.Close)

	if !httputil.Is2xxStatusCode(resp.StatusCode) {
		return getErrorResponse(resp)
	}

	if err := json.NewDecoder(resp.Body).Decode(response); err != nil {
		return pgkErrors.Wrap(err, "decoding response body")
	}

	return nil
}

func getErrorResponse(resp *http.Response) error {
	buf := &bytes.Buffer{}
	if _, err := buf.ReadFrom(resp.Body); err != nil {
		return fmt.Errorf("request failed with status %d", resp.StatusCode)
	}
	return fmt.Errorf("request failed with status %d and response %q", resp.StatusCode, buf.String())
}

func (c *paladinClient) mapAssetToDiscoveredCluster(asset Asset) (*discoveredclusters.DiscoveredCluster, error) {
	d := &discoveredclusters.DiscoveredCluster{
		ID:                asset.GetID(),
		Name:              asset.GetName(),
		Type:              asset.GetType(),
		ProviderType:      asset.GetProviderType(),
		Region:            asset.Region,
		CloudSourceID:     c.cloudSourceID,
		FirstDiscoveredAt: &asset.FirstDiscoveredAt,
	}

	return d, nil
}
