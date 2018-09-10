package dnrintegration

import (
	"crypto/tls"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/stackrox/rox/central/deployment/datastore"
	"golang.org/x/time/rate"
)

const (
	httpTimeout = 5 * time.Second
)

var (
	// Reuse a long-lived client so we don't end up creating too many connections.
	// Note that clients _are_ thread-safe.
	client = &http.Client{
		Timeout: httpTimeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
)

type preventDeploymentMetadata struct {
	name      string
	namespace string
}

type dnrServiceMetadata struct {
	serviceID string
	clusterID string
}

type serviceMapping map[preventDeploymentMetadata]dnrServiceMetadata

type dnrIntegrationImpl struct {
	portalURL *url.URL
	authToken string
	client    *http.Client

	// true if the version of D&R doesn't support multi-cluster.
	// (Necessary because the API change between versions.)
	isPreMultiCluster           bool
	serviceMappingSingleCluster serviceMapping

	// mapping from prevent cluster id -> D&R cluster id.
	clusterIDMapping map[string]string

	// service mappings for each prevent cluster
	serviceMappings            map[string]serviceMapping
	serviceMappingsLock        sync.RWMutex
	serviceMappingsRateLimiter *rate.Limiter

	deploymentStore datastore.DataStore
}
