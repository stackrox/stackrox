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
	"github.com/heroku/docker-registry-client/registry"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
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

func validate(ecr *storage.ECRConfig) error {
	errorList := errorhelpers.NewErrorList("ECR Validation")
	if ecr.GetRegistryId() == "" {
		errorList.AddString("Registry ID must be specified")
	}
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

	if ecr.GetRegion() == "" {
		errorList.AddString("Region must be specified")
	}
	return errorList.ToError()
}

func (e *ecr) refreshDockerClient() error {
	if e.expiryTime.After(time.Now()) {
		return nil
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
		return fmt.Errorf("Malformed basic auth response from AWS '%s'", basicAuth)
	}

	conf := docker.Config{
		Endpoint: e.endpoint,
		Username: basicAuth[:colon],
		Password: basicAuth[colon+1:],
	}

	client, err := docker.NewDockerRegistryWithConfig(conf, e.integration)
	if err != nil {
		return err
	}

	e.Registry = client
	e.expiryTime = *authData.ExpiresAt
	return nil
}

// Metadata returns the metadata via this registries implementation
func (e *ecr) Metadata(image *storage.Image) (*storage.ImageMetadata, error) {
	if err := e.refreshDockerClient(); err != nil {
		return nil, err
	}
	return e.Registry.Metadata(image)
}

// Test tests the current registry and makes sure that it is working properly
func (e *ecr) Test() error {
	if err := e.refreshDockerClient(); err != nil {
		return err
	}

	_, err := e.Registry.Client.Repositories()

	// the following code taken from generic Test method
	if err != nil {
		logging.Errorf("error testing ECR integration: %v", err)
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
	ecrConfig, ok := integration.IntegrationConfig.(*storage.ImageIntegration_Ecr)
	if !ok {
		return nil, errors.New("ECR configuration required")
	}
	conf := ecrConfig.Ecr
	if err := validate(conf); err != nil {
		return nil, err
	}

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

	var service *awsECR.ECR

	if conf.GetUseAssumeRole() {
		if endpoint != "" {
			return nil, errorhelpers.NewErrInvalidArgs("AssumeRole and Endpoint cannot both be enabled")
		}
		if conf.GetAssumeRoleId() == "" {
			return nil, errorhelpers.NewErrInvalidArgs("AssumeRole ID is required to use AssumeRole")
		}

		roleToAssumeArn := fmt.Sprintf("arn:aws:iam::%s:role/%s", conf.RegistryId, conf.AssumeRoleId)
		stsCred := stscreds.NewCredentials(sess, roleToAssumeArn, func(p *stscreds.AssumeRoleProvider) {
			assumeRoleExternalID := conf.GetAssumeRoleExternalId()
			if assumeRoleExternalID != "" {
				p.ExternalID = &assumeRoleExternalID
			}
		})

		service = awsECR.New(sess, &aws.Config{Credentials: stsCred})
	} else {
		service = awsECR.New(sess)
	}

	reg := &ecr{
		config:      conf,
		integration: integration,
		// docker endpoint
		endpoint: fmt.Sprintf("%s.dkr.ecr.%s.amazonaws.com", conf.GetRegistryId(), conf.GetRegion()),
		service:  service,
	}
	if err := reg.refreshDockerClient(); err != nil {
		return nil, err
	}
	return reg, nil
}
