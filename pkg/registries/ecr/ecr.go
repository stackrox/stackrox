package ecr

import (
	"context"
	"fmt"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	awsECR "github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/heroku/docker-registry-client/registry"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/httputil/proxy"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/registries/docker"
	"github.com/stackrox/rox/pkg/registries/types"
	"github.com/stackrox/rox/pkg/urlfmt"
)

var log = logging.LoggerForModule()

var _ types.Registry = (*ecr)(nil)

// ecr implements docker registry access to AWS ECR. The docker credentials
// are either taken from the datastore, in which case they have been synced
// by Sensor, or they are derived from short-lived access tokens. The access
// token is refreshed as part of the transport.
type ecr struct {
	*docker.Registry

	config      *storage.ECRConfig
	integration *storage.ImageIntegration
	transport   *awsTransport
}

// sanitizeConfiguration validates and cleans-up the integration configuration.
func sanitizeConfiguration(ecr *storage.ECRConfig) error {
	errorList := errorhelpers.NewErrorList("ECR Validation")
	if ecr.GetRegistryId() == "" {
		errorList.AddString("Registry ID must be specified")
	}
	// Erase authorization data if any other auth mechanism was set.
	if ecr.GetUseIam() || ecr.GetAccessKeyId() != "" || ecr.GetSecretAccessKey() != "" || ecr.GetUseAssumeRole() {
		ecr.AuthorizationData = nil
	}
	if ecr.GetAuthorizationData() != nil {
		if ecr.GetAuthorizationData().GetUsername() == "" {
			errorList.AddString("Username must be specified in authorization data.")
		}
		if ecr.GetAuthorizationData().GetPassword() == "" {
			errorList.AddString("Password must be specified in authorization data.")
		}
		if ecr.GetAuthorizationData().GetExpiresAt() == nil {
			errorList.AddString("Expires At must be specified in authorization data.")
		}
	} else {
		if !ecr.GetUseIam() {
			if ecr.GetAccessKeyId() == "" {
				errorList.AddString("Access Key ID must be specified if not using IAM")
			}
			if ecr.GetSecretAccessKey() == "" {
				errorList.AddString("Secret Access Key must be specified if not using IAM")
			}
		}
		if ecr.GetUseAssumeRole() {
			if ecr.GetEndpoint() != "" {
				errorList.AddString("AssumeRole cannot be done with an endpoint defined")
			}
			if ecr.GetAssumeRoleId() == "" {
				errorList.AddString("AssumeRole ID must be set to use AssumeRole")
			}
		}
	}
	if ecr.GetRegion() == "" {
		errorList.AddString("Region must be specified")
	}
	return errorList.ToError()
}

// Config returns an up to date docker registry configuration.
func (e *ecr) Config(ctx context.Context) *types.Config {
	// No need for synchronization if there is no transport.
	if e.transport == nil {
		return e.Registry.Config(ctx)
	}
	if err := e.transport.ensureValid(ctx); err != nil {
		log.Errorf("Failed to ensure access token validity for image integration %q: %v", e.transport.name, err)
	}
	return e.Registry.Config(ctx)
}

// Test tests the current registry and makes sure that it is working properly.
func (e *ecr) Test() error {
	_, err := e.Registry.Client.Repositories()
	// the following code taken from generic Test method
	if err != nil {
		log.Errorf("error testing ECR integration: %v", err)
		if e, _ := err.(*registry.ClientError); e != nil {
			return errors.Errorf("error testing ECR integration (code: %d). Please check Central logs for full error", e.Code())
		}
		return err
	}
	return nil
}

// Creator provides the type and registries.Creator to add to the registries Registry.
func Creator() (string, types.Creator) {
	return types.ECRType,
		func(integration *storage.ImageIntegration, options ...types.CreatorOption) (types.Registry, error) {
			cfg := types.ApplyCreatorOptions(options...)
			reg, err := newRegistry(integration, false, cfg.GetMetricsHandler())
			return reg, err
		}
}

// CreatorWithoutRepoList provides the type and registries.Creator to add to the registries Registry.
// Populating the internal repo list will be disabled.
func CreatorWithoutRepoList() (string, types.Creator) {
	return types.ECRType,
		func(integration *storage.ImageIntegration, options ...types.CreatorOption) (types.Registry, error) {
			cfg := types.ApplyCreatorOptions(options...)
			reg, err := newRegistry(integration, true, cfg.GetMetricsHandler())
			return reg, err
		}
}

func newRegistry(integration *storage.ImageIntegration, disableRepoList bool,
	metricsHandler *types.MetricsHandler,
) (*ecr, error) {
	conf := integration.GetEcr()
	if conf == nil {
		return nil, errors.New("ECR configuration required")
	}
	if err := sanitizeConfiguration(conf); err != nil {
		return nil, err
	}
	reg := &ecr{
		config:      conf,
		integration: integration,
	}
	endpoint := fmt.Sprintf("%s.dkr.ecr.%s.amazonaws.com", conf.GetRegistryId(), conf.GetRegion())
	// If the ECR configuration provides Authorization Data, we do not initialize an
	// ECR client, but instead, we create the registry immediately since the
	// Authorization Data payload provides the credentials statically.
	cfg := &docker.Config{
		Endpoint:        endpoint,
		DisableRepoList: disableRepoList,
		MetricsHandler:  metricsHandler,
		RegistryType:    integration.GetType(),
	}
	if authData := conf.GetAuthorizationData(); authData != nil {
		cfg.SetCredentials(authData.GetUsername(), authData.GetPassword())
		dockerRegistry, err := docker.NewDockerRegistryWithConfig(cfg, reg.integration)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create docker registry")
		}
		reg.Registry = dockerRegistry
		return reg, nil
	}

	// TODO(ROX-25474) refactor to pass parent context.
	ctx := context.Background()
	client, err := createECRClient(ctx, conf)
	if err != nil {
		log.Error("Failed to create ECR client: ", err)
		return nil, err
	}
	reg.transport = newAWSTransport(integration.GetName(), cfg, client)
	dockerRegistry, err := docker.NewDockerRegistryWithConfig(cfg, reg.integration, reg.transport)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create docker registry")
	}
	reg.Registry = dockerRegistry
	return reg, nil
}

// createECRClient creates an AWS ECR SDK client based on the integration config.
func createECRClient(ctx context.Context, conf *storage.ECRConfig) (*awsECR.Client, error) {
	opts := []func(*config.LoadOptions) error{
		config.WithRegion(conf.GetRegion()),
		config.WithHTTPClient(&http.Client{Transport: proxy.RoundTripper()}),
	}
	if !conf.GetUseIam() {
		opts = append(opts,
			config.WithCredentialsProvider(
				credentials.NewStaticCredentialsProvider(conf.GetAccessKeyId(), conf.GetSecretAccessKey(), ""),
			),
		)
	}
	awsConfig, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, errors.Wrap(err, "unable to load the aws config")
	}

	if conf.GetUseAssumeRole() {
		if conf.GetEndpoint() != "" {
			return nil, errox.InvalidArgs.CausedBy("AssumeRole and Endpoint cannot both be enabled")
		}
		if conf.GetAssumeRoleId() == "" {
			return nil, errox.InvalidArgs.CausedBy("AssumeRole ID is required to use AssumeRole")
		}

		roleToAssumeArn := fmt.Sprintf("arn:aws:iam::%s:role/%s", conf.RegistryId, conf.AssumeRoleId)
		stsClient := sts.NewFromConfig(awsConfig)
		awsConfig.Credentials = stscreds.NewAssumeRoleProvider(stsClient, roleToAssumeArn,
			func(p *stscreds.AssumeRoleOptions) {
				if externalID := conf.GetAssumeRoleExternalId(); externalID != "" {
					p.ExternalID = aws.String(externalID)
				}
			},
		)
	}

	var clientOpts []func(*awsECR.Options)
	if endpoint := conf.GetEndpoint(); endpoint != "" {
		clientOpts = append(clientOpts, func(o *awsECR.Options) {
			o.BaseEndpoint = aws.String(urlfmt.FormatURL(endpoint, urlfmt.HTTPS, urlfmt.NoTrailingSlash))
		})
	}
	return awsECR.NewFromConfig(awsConfig, clientOpts...), nil
}
