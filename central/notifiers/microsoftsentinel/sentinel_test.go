package microsoftsentinel

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/base64"
	"io"
	"net/http"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/monitor/ingestion/azlogs"
	"github.com/stackrox/rox/central/notifiers/microsoftsentinel/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/cryptoutils/cryptocodec"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	pkgNotifiers "github.com/stackrox/rox/pkg/notifiers"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

const (
	alertStreamName = "Custom-stackrox_notifier_CL"
	alertDcrID      = "aaaaaaaa-bbbb-4011-0000-111111111111"

	auditStreamName = "Custom-stackrox_audit_CL"
	auditDcrID      = "aaaaaaaa-bbbb-4022-0000-222222222222"
)

var (
	//go:embed testdata/sentinel-ca-key.pem
	sentinelCaKey string

	//go:embed testdata/sentinel-ca-cert.pem
	sentinelCaCert string
)

func TestSentinelNotifier(t *testing.T) {
	suite.Run(t, new(SentinelTestSuite))
}

type SentinelTestSuite struct {
	suite.Suite

	mockCtrl        *gomock.Controller
	mockAzureClient *mocks.MockazureLogsClient
}

func (suite *SentinelTestSuite) SetupTest() {
	suite.mockCtrl = gomock.NewController(suite.T())
	suite.mockAzureClient = mocks.NewMockazureLogsClient(suite.mockCtrl)

	suite.T().Setenv(features.MicrosoftSentinelNotifier.EnvVar(), "true")
}

func (suite *SentinelTestSuite) TestAlertNotify() {
	alert := &storage.Alert{
		Id:          uuid.NewDummy().String(),
		ClusterName: "test-cluster",
		Entity: &storage.Alert_Deployment_{
			Deployment: &storage.Alert_Deployment{
				Name: "nginx",
			},
		},
	}

	notifier := &sentinel{
		azlogsClient: suite.mockAzureClient,
		notifier:     getNotifierConfig(),
	}

	logsToSend, err := notifier.prepareLogsToSend(alert)
	suite.Require().NoError(err)

	// Assert call to library and marshalling is correct.
	suite.mockAzureClient.EXPECT().Upload(gomock.Any(), uuid.NewDummy().String(), alertStreamName, logsToSend, gomock.Any()).Times(1)
	require.NotNil(suite.T(), notifier)

	err = notifier.AlertNotify(context.Background(), alert)
	suite.Require().NoError(err)
}

func (suite *SentinelTestSuite) TestRetry() {
	notifier := &sentinel{
		azlogsClient: suite.mockAzureClient,
		notifier:     getNotifierConfig(),
	}

	body := bytes.NewBuffer([]byte("http error body"))
	respErr := &azcore.ResponseError{
		StatusCode: http.StatusServiceUnavailable,
		RawResponse: &http.Response{
			StatusCode: http.StatusServiceUnavailable,
			Body:       io.NopCloser(body),
		},
	}

	suite.mockAzureClient.EXPECT().
		Upload(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(3).
		Return(azlogs.UploadResponse{}, respErr)

	err := notifier.AlertNotify(context.Background(), &storage.Alert{})
	suite.Require().Error(err)
}

func (suite *SentinelTestSuite) TestValidate() {
	testCases := map[string]struct {
		Config                      *storage.MicrosoftSentinel
		ValidateSecret              bool
		ExpectedErrorMsg            string
		ExpectedErrorMsgNotContains string
	}{
		"Given a valid configuration validation should pass": {
			Config: &storage.MicrosoftSentinel{
				LogIngestionEndpoint: "portal.azure.com",
				ApplicationClientId:  uuid.NewDummy().String(),
				DirectoryTenantId:    uuid.NewDummy().String(),
				Secret:               "my secret value",
				AlertDcrConfig: &storage.MicrosoftSentinel_DataCollectionRuleConfig{
					Enabled:              true,
					DataCollectionRuleId: alertDcrID,
					StreamName:           alertStreamName,
				},
				AuditLogDcrConfig: &storage.MicrosoftSentinel_DataCollectionRuleConfig{
					Enabled:              true,
					DataCollectionRuleId: auditDcrID,
					StreamName:           auditStreamName,
				},
			},
			ExpectedErrorMsg: "",
			ValidateSecret:   true,
		},
		"given an invalid config validation should fail": {
			Config: &storage.MicrosoftSentinel{
				AlertDcrConfig: &storage.MicrosoftSentinel_DataCollectionRuleConfig{
					Enabled: true,
				},
				AuditLogDcrConfig: &storage.MicrosoftSentinel_DataCollectionRuleConfig{
					Enabled: true,
				},
			},
			ExpectedErrorMsg: "[Log Ingestion Endpoint must be specified, Audit Logging Data Collection Rule Id must be specified, Audit Logging Stream Name must be specified, Alert Data Collection Rule Id must be specified, Alert Stream Name must be specified, Directory Tenant Id must be specified, Application Client Id must be specified, Secret, Client Certificate or Workload Identity authentication must be specified]",
			ValidateSecret:   true,
		},
		"given alert log dcr config is enabled with an invalid config validation should not pass": {
			Config: &storage.MicrosoftSentinel{
				ApplicationClientId:  uuid.NewDummy().String(),
				DirectoryTenantId:    uuid.NewDummy().String(),
				LogIngestionEndpoint: "example.com",
				AlertDcrConfig: &storage.MicrosoftSentinel_DataCollectionRuleConfig{
					Enabled: true,
				},
			},
			ExpectedErrorMsg: "[Alert Data Collection Rule Id must be specified, Alert Stream Name must be specified]",
		},
		"given audit log dcr config is enabled with an invalid config validation should not pass": {
			Config: &storage.MicrosoftSentinel{
				ApplicationClientId:  uuid.NewDummy().String(),
				DirectoryTenantId:    uuid.NewDummy().String(),
				LogIngestionEndpoint: "example.com",
				AuditLogDcrConfig: &storage.MicrosoftSentinel_DataCollectionRuleConfig{
					Enabled: true,
				},
			},
			ExpectedErrorMsg: "[Audit Logging Data Collection Rule Id must be specified, Audit Logging Stream Name must be specified]",
		},
		"given client cert authentication validation should pass": {
			Config: &storage.MicrosoftSentinel{
				ApplicationClientId:  uuid.NewDummy().String(),
				DirectoryTenantId:    uuid.NewDummy().String(),
				LogIngestionEndpoint: "example.com",
				ClientCertAuthConfig: &storage.MicrosoftSentinel_ClientCertAuthConfig{
					ClientCert: "cert",
					PrivateKey: "key",
				},
			},
			ValidateSecret: true,
		},
		"given client cert authentication with missing private key validation should not pass": {
			Config: &storage.MicrosoftSentinel{
				ApplicationClientId:  uuid.NewDummy().String(),
				DirectoryTenantId:    uuid.NewDummy().String(),
				LogIngestionEndpoint: "example.com",
				ClientCertAuthConfig: &storage.MicrosoftSentinel_ClientCertAuthConfig{
					ClientCert: "cert",
					PrivateKey: "",
				},
			},
			ValidateSecret:   true,
			ExpectedErrorMsg: "Secret, Client Certificate or Workload Identity authentication must be specified",
		},
		"given client cert authentication with missing client certificate validation should not pass": {
			Config: &storage.MicrosoftSentinel{
				ApplicationClientId:  uuid.NewDummy().String(),
				DirectoryTenantId:    uuid.NewDummy().String(),
				LogIngestionEndpoint: "example.com",
				ClientCertAuthConfig: &storage.MicrosoftSentinel_ClientCertAuthConfig{
					ClientCert: "",
					PrivateKey: "key",
				},
			},
			ValidateSecret:   true,
			ExpectedErrorMsg: "Secret, Client Certificate or Workload Identity authentication must be specified",
		},
		"given only secret authentication validation should not pass": {
			Config: &storage.MicrosoftSentinel{
				ApplicationClientId:  uuid.NewDummy().String(),
				DirectoryTenantId:    uuid.NewDummy().String(),
				LogIngestionEndpoint: "example.com",
				Secret:               "secret",
			},
			ValidateSecret: true,
		},
		"given authentication configs are missing validation should not pass": {
			Config: &storage.MicrosoftSentinel{
				AlertDcrConfig: &storage.MicrosoftSentinel_DataCollectionRuleConfig{
					Enabled: true,
				},
			},
			ExpectedErrorMsg:            "[Log Ingestion Endpoint must be specified, Alert Data Collection Rule Id must be specified, Alert Stream Name must be specified, Directory Tenant Id must be specified, Application Client Id must be specified]",
			ExpectedErrorMsgNotContains: "secret",
			ValidateSecret:              false,
		},
		"given only workload identity authentication should pass": {
			Config: &storage.MicrosoftSentinel{
				ApplicationClientId:  uuid.NewDummy().String(),
				DirectoryTenantId:    uuid.NewDummy().String(),
				LogIngestionEndpoint: "example.com",
				WifEnabled:           true,
			},
			ValidateSecret: true,
		},
	}

	for name, testCase := range testCases {
		suite.T().Run(name, func(t *testing.T) {
			err := Validate(testCase.Config, testCase.ValidateSecret)
			if testCase.ExpectedErrorMsg == "" {
				assert.NoError(t, err)
			} else {
				assert.NotContains(t, testCase.ExpectedErrorMsgNotContains, err.Error())
				if testCase.ExpectedErrorMsg != "" {
					assert.Contains(t, err.Error(), testCase.ExpectedErrorMsg)
				}
			}
		})
	}
}

func (suite *SentinelTestSuite) TestAuditTestAlert() {
	config := getNotifierConfig()
	config.GetMicrosoftSentinel().AuditLogDcrConfig.Enabled = false

	notifier := &sentinel{
		azlogsClient: suite.mockAzureClient,
		notifier:     config,
	}

	testAlert := notifier.getTestAlert()
	bytesToSend, err := notifier.prepareLogsToSend(testAlert)
	suite.Require().NoError(err)

	suite.mockAzureClient.EXPECT().Upload(gomock.Any(), alertDcrID, alertStreamName, bytesToSend, gomock.Any()).Times(1)

	notifierErr := notifier.Test(context.TODO())
	suite.Require().Nil(notifierErr)
}

func (suite *SentinelTestSuite) TestTestAuditLogMessage() {
	config := getNotifierConfig()
	config.GetMicrosoftSentinel().AlertDcrConfig.Enabled = false

	notifier := &sentinel{
		azlogsClient: suite.mockAzureClient,
		notifier:     config,
	}

	testAuditMessage := notifier.getTestAuditLogMessage()
	bytesToSend, err := notifier.prepareLogsToSend(testAuditMessage)
	suite.Require().NoError(err)

	suite.mockAzureClient.EXPECT().Upload(gomock.Any(), auditDcrID, auditStreamName, bytesToSend, gomock.Any()).Times(1)

	notifierErr := notifier.Test(context.TODO())
	suite.Require().Nil(notifierErr)
}

func (suite *SentinelTestSuite) TestAuditLogEnabled() {
	notifier := &sentinel{
		azlogsClient: suite.mockAzureClient,
		notifier:     getNotifierConfig(),
	}
	suite.Assert().True(notifier.AuditLoggingEnabled())

	notifier.notifier.GetMicrosoftSentinel().GetAuditLogDcrConfig().Enabled = false
	suite.Assert().False(notifier.AuditLoggingEnabled())
}

func (suite *SentinelTestSuite) TestNewSentinelNotifier() {
	config := getNotifierConfig()
	config.GetMicrosoftSentinel().ClientCertAuthConfig = &storage.MicrosoftSentinel_ClientCertAuthConfig{
		ClientCert: sentinelCaCert,
		PrivateKey: sentinelCaKey,
	}

	notifier, err := newSentinelNotifier(config, nil, "")

	suite.Require().NoError(err)
	suite.NotNil(notifier)
}

func (suite *SentinelTestSuite) TestEncryption() {
	suite.T().Setenv(env.EncNotifierCreds.EnvVar(), "true")

	var exampleKey = []byte("key-string-12345")
	b64EncodedKey := base64.StdEncoding.EncodeToString(exampleKey)
	encryptedSecret, err := cryptocodec.NewGCMCryptoCodec().Encrypt(b64EncodedKey, "secret-for-sentinel")
	suite.Require().NoError(err)

	config := getNotifierConfig()
	config.NotifierSecret = encryptedSecret

	sentinelNotifier, err := newSentinelNotifier(config, cryptocodec.Singleton(), b64EncodedKey)

	suite.Require().NoError(err)
	suite.Require().NotNil(sentinelNotifier)

	// test with invalid secret encryption should fail
	config.NotifierSecret = ""

	sentinelNotifier, err = newSentinelNotifier(config, cryptocodec.Singleton(), b64EncodedKey)
	suite.ErrorContains(err, "Error decrypting notifier secret for notifier \"microsoft-sentinel\"")
	suite.Nil(sentinelNotifier)
}

func getNotifierConfig() *storage.Notifier {
	return &storage.Notifier{
		Name: "microsoft-sentinel",
		Type: pkgNotifiers.MicrosoftSentinelType,
		Config: &storage.Notifier_MicrosoftSentinel{
			MicrosoftSentinel: &storage.MicrosoftSentinel{
				LogIngestionEndpoint: "portal.azure.com",
				ApplicationClientId:  uuid.NewDummy().String(),
				DirectoryTenantId:    uuid.NewDummy().String(),
				AlertDcrConfig: &storage.MicrosoftSentinel_DataCollectionRuleConfig{
					DataCollectionRuleId: alertDcrID,
					StreamName:           alertStreamName,
					Enabled:              true,
				},
				AuditLogDcrConfig: &storage.MicrosoftSentinel_DataCollectionRuleConfig{
					DataCollectionRuleId: auditDcrID,
					StreamName:           auditStreamName,
					Enabled:              true,
				},
			},
		},
	}
}
