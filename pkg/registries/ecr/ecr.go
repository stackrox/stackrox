package ecr

import (
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	awsECR "github.com/aws/aws-sdk-go/service/ecr"
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

	if ecr.GetRegion() == "" {
		errorList.AddString("Region must be specified")
	}
	return errorList.ToError()
}

func (e *ecr) refreshDockerClient() error {
	if e.expiryTime.After(time.Now()) {
		return nil
	}
	authToken, err := e.service.GetAuthorizationToken(&awsECR.GetAuthorizationTokenInput{
		RegistryIds: []*string{
			aws.String(e.config.GetRegistryId()),
		},
	})
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
	return e.Registry.Test()
}

// Creator provides the type and registries.Creator to add to the registries Registry.
func Creator() (string, func(integration *storage.ImageIntegration) (types.ImageRegistry, error)) {
	return "ecr", func(integration *storage.ImageIntegration) (types.ImageRegistry, error) {
		reg, err := newRegistry(integration)
		return reg, err
	}
}

func newRegistry(integration *storage.ImageIntegration) (*ecr, error) {
	ecrConfig, ok := integration.IntegrationConfig.(*storage.ImageIntegration_Ecr)
	if !ok {
		return nil, fmt.Errorf("ECR configuration required")
	}
	conf := ecrConfig.Ecr
	if err := validate(conf); err != nil {
		return nil, err
	}

	endpoint := fmt.Sprintf("%s.dkr.ecr.%s.amazonaws.com", conf.GetRegistryId(), conf.GetRegion())

	var err error
	var sess *session.Session
	if conf.GetUseIam() {
		sess, err = session.NewSession(&aws.Config{
			Region: aws.String(conf.GetRegion()),
		})
	} else {
		creds := credentials.NewStaticCredentials(conf.GetAccessKeyId(), conf.GetSecretAccessKey(), "")
		sess, err = session.NewSession(&aws.Config{
			Region:      aws.String(conf.GetRegion()),
			Credentials: creds,
		})
	}
	if err != nil {
		return nil, err
	}
	service := awsECR.New(sess)
	reg := &ecr{
		config:      conf,
		integration: integration,

		endpoint: endpoint,
		service:  service,
	}
	if err := reg.refreshDockerClient(); err != nil {
		return nil, err
	}
	return reg, nil
}
