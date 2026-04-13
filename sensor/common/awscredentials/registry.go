// Package awscredentials provides Sensor components that can retrieve, cache,
// refresh and offer AWS-based credentials and tokens.
package awscredentials

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	awsimds "github.com/stackrox/rox/pkg/cloudproviders/aws"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/docker/config"
	"github.com/stackrox/rox/pkg/httputil/proxy"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	ecrRegistryRegex     = regexp.MustCompile(`(\d+)\.dkr\.ecr\.([^.]+)\.amazonaws\.com`)
	ecrRegexAccountGroup = 1
	ecrRegexRegionGroup  = 2

	clientTimeout = 30 * time.Second

	log = logging.LoggerForModule()
)

// ecrCredentialsManager manages credentials pulled from global ECR registries.
type ecrCredentialsManager struct {
	dockerConfigEntry *config.DockerConfigEntry
	dockerConfigLock  sync.RWMutex
	region            string
	expiresAt         time.Time
	stopSignal        concurrency.Signal
}

// NewECRCredentialsManager checks for AWS provider information and, if valid,
// creates an ECR credential manager instance.
func NewECRCredentialsManager(providerID string) (RegistryCredentialsManager, error) {
	if !strings.HasPrefix(providerID, "aws://") {
		return nil, fmt.Errorf("node provider is not AWS: %v", providerID)
	}
	log.Infof("detected AWS-based node: providerId=%s", providerID)

	ctx := context.Background()
	mdClient := awsimds.NewIMDSClient(&http.Client{
		Timeout:   clientTimeout,
		Transport: proxy.Without(),
	})
	mdClient.GetToken(ctx)
	region, err := mdClient.GetRegion(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting region from EC2 metadata service: %v", err)
	}
	log.Infof("EC2 instance metadata service is active: awsRegion=%q", region)

	return &ecrCredentialsManager{
		region:     region,
		stopSignal: concurrency.NewSignal(),
	}, nil
}

func (m *ecrCredentialsManager) Start() {
	const refreshInterval = 5 * time.Minute
	go m.refreshLoop(refreshInterval)
}

// refreshLoop ticks every refresh interval when the auth token is close to expiring.
//
// We currently use 1h threshold to renew the auth token. The rationale is,
// `GetAuthorizationToken` tokens have a lifetime of 12h, and we don't really
// need to refresh regularly, only when close to expire. One hour seemed
// reasonable enough, to accommodate for any temporary failure that might arise
// that prevents us from getting a new token. Notice we also retry linearly,
// which also seemed reasonable given the `GetAuthorizationToken` API call rate
// is 500 rps.
func (m *ecrCredentialsManager) refreshLoop(refreshInterval time.Duration) {
	log.Infof("starting ECR credentials manager, refresh interval: %v", refreshInterval)
	ticker := time.NewTicker(refreshInterval)
	defer ticker.Stop()
	ctx := context.Background()
	for {
		if m.authWillExpireIn(time.Hour) {
			err := m.refreshAuthToken(ctx)
			if err != nil {
				log.Warnf("failed to refresh ECR authentication token: %v", err)
			}
		}
		select {
		case <-ticker.C:
			continue
		case <-m.stopSignal.Done():
			return
		}
	}
}

func (m *ecrCredentialsManager) Stop() {
	m.stopSignal.Signal()
}

func (m *ecrCredentialsManager) GetRegistryCredentials(registry string) *RegistryCredentials {
	acc, reg, ok := findECRURLAccountAndRegion(registry)
	if !ok {
		// Invalid ECR registry URL, so credentials are not available.
		return nil
	}
	cfg := m.getConfigIfValid()
	if cfg == nil {
		return nil
	}
	return &RegistryCredentials{
		AWSAccount:   acc,
		AWSRegion:    reg,
		DockerConfig: cfg,
		ExpirestAt:   m.expiresAt,
	}
}

// findECRURLAccountAndRegion returns the account and region ECR registry
// URL, if it's not a valid ECR registry URL returns nils and false.
func findECRURLAccountAndRegion(registry string) (account, region string, ok bool) {
	match := ecrRegistryRegex.FindStringSubmatch(registry)
	if match != nil {
		account, region = match[ecrRegexAccountGroup], match[ecrRegexRegionGroup]
		ok = true
	}
	return
}

// refreshAuthToken resolves AWS credentials and calls ECR GetAuthorizationToken.
func (m *ecrCredentialsManager) refreshAuthToken(ctx context.Context) error {
	creds, err := awsimds.ResolveCredentials(ctx, m.region)
	if err != nil {
		return fmt.Errorf("resolving AWS credentials: %v", err)
	}

	token, err := awsimds.GetECRAuthorizationToken(ctx, creds, m.region)
	if err != nil {
		return fmt.Errorf("getting ECR auth token: %v", err)
	}

	dockerConfigEntry, err := config.CreateFromAuthString(token.AuthorizationToken)
	if err != nil {
		return fmt.Errorf("creating docker config from token: %v", err)
	}

	m.setConfig(dockerConfigEntry, token.ExpiresAt)
	log.Infof("ECR's auth token refreshed, expires at: %v", token.ExpiresAt)
	return nil
}

// getConfigIfValid returns the current docker config if it is valid (i.e. not
// expired), otherwise returns nil
func (m *ecrCredentialsManager) getConfigIfValid() *config.DockerConfigEntry {
	m.dockerConfigLock.RLock()
	defer m.dockerConfigLock.RUnlock()
	if m.authIsValid() && m.dockerConfigEntry != nil {
		// Make a copy to encapsulate the config object.
		entry := *m.dockerConfigEntry
		return &entry
	}
	return nil
}

// setConfig sets the current docker config and its expiration timestamp.
func (m *ecrCredentialsManager) setConfig(dockerConfigEntry config.DockerConfigEntry, expiresAt time.Time) {
	m.dockerConfigLock.Lock()
	defer m.dockerConfigLock.Unlock()
	m.dockerConfigEntry = &dockerConfigEntry
	m.expiresAt = expiresAt
}

// authIsValid returns true if the current auth token hasn't expired.
func (m *ecrCredentialsManager) authIsValid() bool {
	return time.Now().Before(m.expiresAt)
}

// authWillExpireIn returns true if auth token is expired or will expire within
// the given duration.
func (m *ecrCredentialsManager) authWillExpireIn(duration time.Duration) bool {
	return time.Now().Add(duration).After(m.expiresAt)
}
