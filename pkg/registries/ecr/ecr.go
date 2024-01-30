package ecr

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	awsECR "github.com/aws/aws-sdk-go/service/ecr"
	"github.com/heroku/docker-registry-client/registry"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/registries/docker"
	"github.com/stackrox/rox/pkg/registries/types"
)

var log = logging.LoggerForModule()

var _ types.Registry = (*ecr)(nil)

type ecr struct {
	*docker.Registry

	config      *storage.ECRConfig
	integration *storage.ImageIntegration
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
		func(integration *storage.ImageIntegration, _ ...types.CreatorOption) (types.Registry, error) {
			reg, err := newRegistry(integration, false)
			return reg, err
		}
}

// CreatorWithoutRepoList provides the type and registries.Creator to add to the registries Registry.
// Populating the internal repo list will be disabled.
func CreatorWithoutRepoList() (string, types.Creator) {
	return types.ECRType,
		func(integration *storage.ImageIntegration, _ ...types.CreatorOption) (types.Registry, error) {
			reg, err := newRegistry(integration, true)
			return reg, err
		}
}

func newRegistry(integration *storage.ImageIntegration, disableRepoList bool) (*ecr, error) {
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
	}
	if authData := conf.GetAuthorizationData(); authData != nil {
		cfg.Username = authData.GetUsername()
		cfg.Password = authData.GetPassword()
	} else {
		client, err := createECRClient(conf)
		if err != nil {
			log.Error("Failed to create ECR client: ", err)
			return nil, err
		}
		cfg.Transport = newAWSTransport(cfg, client)
	}
	dockerRegistry, err := docker.NewDockerRegistryWithConfig(cfg, reg.integration)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create docker registry")
	}
	reg.Registry = dockerRegistry
	return reg, nil
}

// createECRClient creates an AWS ECR SDK client based on the integration config.
func createECRClient(conf *storage.ECRConfig) (*awsECR.ECR, error) {
	awsConfig := &aws.Config{
		Region: aws.String(conf.GetRegion()),
	}

	endpoint := conf.GetEndpoint()
	if endpoint != "" {
		awsConfig.Endpoint = aws.String(endpoint)
	}

	if !conf.GetUseIam() {
		awsConfig.Credentials = credentials.NewStaticCredentials(conf.GetAccessKeyId(), conf.GetSecretAccessKey(), "")
	}
	sess, err := session.NewSession(awsConfig)
	if err != nil {
		return nil, err
	}

	if conf.GetUseAssumeRole() {
		if endpoint != "" {
			return nil, errox.InvalidArgs.CausedBy("AssumeRole and Endpoint cannot both be enabled")
		}
		if conf.GetAssumeRoleId() == "" {
			return nil, errox.InvalidArgs.CausedBy("AssumeRole ID is required to use AssumeRole")
		}

		roleToAssumeArn := fmt.Sprintf("arn:aws:iam::%s:role/%s", conf.RegistryId, conf.AssumeRoleId)
		stsCred := stscreds.NewCredentials(sess, roleToAssumeArn, func(p *stscreds.AssumeRoleProvider) {
			assumeRoleExternalID := conf.GetAssumeRoleExternalId()
			if assumeRoleExternalID != "" {
				p.ExternalID = &assumeRoleExternalID
			}
		})

		return awsECR.New(sess, &aws.Config{Credentials: stsCred}), nil
	}
	return awsECR.New(sess), nil
}
