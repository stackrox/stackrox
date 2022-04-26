package ecr

import (
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	awsECR "github.com/aws/aws-sdk-go/service/ecr"
	protobuftypes "github.com/gogo/protobuf/types"
	"github.com/heroku/docker-registry-client/registry"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/registries/docker"
	"github.com/stackrox/rox/pkg/registries/types"
)

var (
	log = logging.LoggerForModule()
)

type ecr struct {
	*docker.Registry

	config      *storage.ECRConfig
	integration *storage.ImageIntegration

	endpoint   string
	service    *awsECR.ECR
	expiryTime time.Time
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

func (e *ecr) refreshDockerClient() error {
	if e.expiryTime.After(time.Now()) {
		return nil
	}
	if e.integration.GetEcr().GetAuthorizationData() != nil {
		// This integration has static authorization data, and we never refresh the
		// tokens in central, rather we wait for sensor to update them.
		return errors.New("failed to refresh the auto-generated integration credentials")
	}
	authToken, err := e.service.GetAuthorizationToken(&awsECR.GetAuthorizationTokenInput{})
	if err != nil {
		return err
	}

	if len(authToken.AuthorizationData) == 0 {
		return fmt.Errorf("received empty authorization data in token: %s", authToken)
	}

	authData := authToken.AuthorizationData[0]

	decoded, err := base64.StdEncoding.DecodeString(*authData.AuthorizationToken)
	if err != nil {
		return err
	}
	basicAuth := string(decoded)
	colon := strings.Index(basicAuth, ":")
	if colon == -1 {
		return fmt.Errorf("malformed basic auth response from AWS '%s'", basicAuth)
	}
	return e.setRegistry(basicAuth[:colon], basicAuth[colon+1:], *authData.ExpiresAt)
}

// Metadata returns the metadata via this registry's implementation.
func (e *ecr) Metadata(image *storage.Image) (*storage.ImageMetadata, error) {
	if err := e.refreshDockerClient(); err != nil {
		return nil, err
	}
	return e.Registry.Metadata(image)
}

// Config returns the config via this registry's implementation.
func (e *ecr) Config() *types.Config {
	// TODO(ROX-9868): Return nil-config to caller.
	if err := e.refreshDockerClient(); err != nil {
		log.Errorf("Error refreshing docker client for registry %q: %v", e.Name(), err)
	}
	return e.Registry.Config()
}

// Test tests the current registry and makes sure that it is working properly
func (e *ecr) Test() error {
	if err := e.refreshDockerClient(); err != nil {
		return err
	}

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
func Creator() (string, func(integration *storage.ImageIntegration) (types.Registry, error)) {
	return "ecr", func(integration *storage.ImageIntegration) (types.Registry, error) {
		reg, err := newRegistry(integration)
		return reg, err
	}
}

func newRegistry(integration *storage.ImageIntegration) (*ecr, error) {
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
		// docker endpoint
		endpoint: fmt.Sprintf("%s.dkr.ecr.%s.amazonaws.com", conf.GetRegistryId(), conf.GetRegion()),
	}
	// If the ECR configuration provides Authorization Data, we do not initialize an
	// ECR client, but instead, we create the registry immediately since the
	// Authorization Data payload provides the credentials statically.
	if authData := conf.GetAuthorizationData(); authData != nil {
		expiresAt, err := protobuftypes.TimestampFromProto(authData.GetExpiresAt())
		if err != nil {
			return nil, errors.New("invalid authorization data")
		}
		if err = reg.setRegistry(authData.GetUsername(), authData.GetPassword(), expiresAt); err != nil {
			return nil, errors.Wrap(err, "failed to create registry client")
		}
	} else {
		service, err := createECRClient(conf)
		if err != nil {
			return nil, err
		}
		reg.service = service
		// Refreshing the client will force the creation of the registry client using AWS ECR.
		if err := reg.refreshDockerClient(); err != nil {
			return nil, err
		}
	}
	return reg, nil
}

// setRegistry creates and sets the docker registry client based on the
// credentials provided.
func (e *ecr) setRegistry(username, password string, expiresAt time.Time) error {
	conf := docker.Config{
		Endpoint: e.endpoint,
		Username: username,
		Password: password,
	}
	client, err := docker.NewDockerRegistryWithConfig(conf, e.integration)
	if err != nil {
		return err
	}
	e.Registry = client
	e.expiryTime = expiresAt
	return err
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
			return nil, errox.NewErrInvalidArgs("AssumeRole and Endpoint cannot both be enabled")
		}
		if conf.GetAssumeRoleId() == "" {
			return nil, errox.NewErrInvalidArgs("AssumeRole ID is required to use AssumeRole")
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
