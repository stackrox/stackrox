// Package awscredentials provides Sensor components that can retrieve, cache,
// refresh and offer AWS-based credentials and tokens.
package awscredentials

import (
	"context"
	"net/http"
	"regexp"
	"strings"
	"time"

	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/pkg/errors"
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

// RegistryCredentials carries credential information to access AWS-based
// registries.
type RegistryCredentials struct {
	AWSAccount   string
	AWSRegion    string
	DockerConfig *config.DockerConfigEntry
	ExpirestAt   time.Time
}

// RegistryCredentialsManager is a sensor component that manages
// credentials for docker registries.
//
//go:generate mockgen-wrapper
type RegistryCredentialsManager interface {
	// GetRegistryCredentials returns the most recent registry credential for the given
	// registry URI, or `nil` if not available.
	GetRegistryCredentials(r string) *RegistryCredentials
	Start()
	Stop()
}

// ecrCredentialsManager manages credentials pulled from global ECR registries.
type ecrCredentialsManager struct {
	dockerConfigEntry *config.DockerConfigEntry
	dockerConfigLock  sync.RWMutex
	ecrClient         *ecr.Client
	expiresAt         time.Time
	stopSignal        concurrency.Signal
}

// NewECRCredentialsManager checks for AWS provider information and, if valid,
// creates an ECR credential manager instance.
func NewECRCredentialsManager(providerID string) (RegistryCredentialsManager, error) {
	if !strings.HasPrefix(providerID, "aws://") {
		return nil, errors.Errorf("node provider is not AWS: %v", providerID)
	}
	log.Infof("detected AWS-based node: providerId=%s", providerID)
	client, err := createECRClient(context.Background())
	if err != nil {
		return nil, errors.Wrap(err, "creating ECR client")
	}
	return &ecrCredentialsManager{
		ecrClient:  client,
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

// refreshAuthToken Contact AWS ECR to get a new auth token.
func (m *ecrCredentialsManager) refreshAuthToken(ctx context.Context) error {
	authToken, err := m.ecrClient.GetAuthorizationToken(ctx, &ecr.GetAuthorizationTokenInput{})
	if err != nil {
		return errors.Errorf("failed to get token: %v", err)
	}
	if len(authToken.AuthorizationData) == 0 {
		return errors.Errorf("received empty token: %v", authToken)
	}
	authData := authToken.AuthorizationData[0]
	dockerConfigEntry, err := config.CreateFromAuthString(*authData.AuthorizationToken)
	if err != nil {
		return errors.Errorf("failed to create docker config from token: %v", err)
	}
	expiresAt := *authData.ExpiresAt
	m.setConfig(dockerConfigEntry, expiresAt)
	log.Infof("ECR's auth token refreshed, expires at: %v", expiresAt)
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

// getRegion reads the EC2 instance region via the AWS metadata service.
func getRegion(ctx context.Context) (string, error) {
	imdsConfig, err := awsConfig.LoadDefaultConfig(ctx,
		awsConfig.WithHTTPClient(&http.Client{
			Timeout: clientTimeout,
			// Only EC2 internal network requests are allowed to the metadata service.
			Transport: proxy.Without(),
		}),
	)
	if err != nil {
		return "", err
	}
	imdsClient := imds.NewFromConfig(imdsConfig)
	regionOutput, err := imdsClient.GetRegion(ctx, &imds.GetRegionInput{})
	if err != nil {
		return "", err
	}
	return regionOutput.Region, nil
}

// createECRClient creates an AWS ECR SDK client based on the integration config.
func createECRClient(ctx context.Context) (*ecr.Client, error) {
	region, err := getRegion(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "unable to get the region of the EC2 instance")
	}
	log.Infof("EC2 instance metadata service is active: awsRegion=%q", region)

	opts := []func(*awsConfig.LoadOptions) error{
		awsConfig.WithRegion(region),
		awsConfig.WithHTTPClient(&http.Client{
			Timeout:   clientTimeout,
			Transport: proxy.RoundTripper(),
		}),
	}
	awsConfig, err := awsConfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, errors.Wrap(err, "unable to load the aws config")
	}
	return ecr.NewFromConfig(awsConfig), nil
}
