package microsoftsentinel

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/monitor/ingestion/azlogs"
	"github.com/stackrox/rox/central/notifiers/microsoftsentinel/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
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
		"Test valid configuration": {
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
		"Test invalid config": {
			Config: &storage.MicrosoftSentinel{
				AlertDcrConfig: &storage.MicrosoftSentinel_DataCollectionRuleConfig{
					Enabled: true,
				},
				AuditLogDcrConfig: &storage.MicrosoftSentinel_DataCollectionRuleConfig{
					Enabled: true,
				},
			},
			ExpectedErrorMsg: "[Log Ingestion Endpoint must be specified, Audit Logging Data Collection Rule Id must be specified, Audit Logging Stream Name must be specified, Alert Data Collection Rule Id must be specified, Alert Stream Name must be specified, Directory Tenant Id must be specified, Application Client Id must be specified, Secret must be specified]",
			ValidateSecret:   true,
		},
		"Test invalid config without secret": {
			Config: &storage.MicrosoftSentinel{
				AlertDcrConfig: &storage.MicrosoftSentinel_DataCollectionRuleConfig{
					Enabled: true,
				},
			},
			ExpectedErrorMsg:            "[Log Ingestion Endpoint must be specified, Alert Data Collection Rule Id must be specified, Alert Stream Name must be specified, Directory Tenant Id must be specified, Application Client Id must be specified]",
			ExpectedErrorMsgNotContains: "secret",
			ValidateSecret:              false,
		},
		"Test only alert notifier is invalid": {
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
		"Test only audit log notifier is invalid": {
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
	}

	for name, testCase := range testCases {
		suite.T().Run(name, func(t *testing.T) {
			err := Validate(testCase.Config, testCase.ValidateSecret)
			if testCase.ExpectedErrorMsg == "" {
				assert.NoError(t, err)
			} else {
				assert.NotContains(t, testCase.ExpectedErrorMsgNotContains, err.Error())
				if testCase.ExpectedErrorMsg != "" {
					assert.Equal(t, fmt.Sprintf("Microsoft Sentinel validation errors: %s", testCase.ExpectedErrorMsg), err.Error())
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

func getNotifierConfig() *storage.Notifier {
	return &storage.Notifier{
		Name: "microsoft-sentinel",
		Config: &storage.Notifier_MicrosoftSentinel{
			MicrosoftSentinel: &storage.MicrosoftSentinel{
				LogIngestionEndpoint: "portal.azure.com",
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
