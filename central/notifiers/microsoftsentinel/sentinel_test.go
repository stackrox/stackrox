package microsoftsentinel

import (
	"context"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/monitor/ingestion/azlogs"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/notifiers/microsoftsentinel/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

const streamName = "Custom-stackrox_notifier_CL"

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
	suite.mockAzureClient.EXPECT().Upload(gomock.Any(), uuid.NewDummy().String(), streamName, logsToSend, gomock.Any()).Times(1)
	require.NotNil(suite.T(), notifier)

	err = notifier.AlertNotify(context.Background(), alert)
	require.NoError(suite.T(), err)
}

func (suite *SentinelTestSuite) TestRetry() {
	notifier := &sentinel{
		azlogsClient: suite.mockAzureClient,
		notifier:     getNotifierConfig(),
	}

	suite.mockAzureClient.EXPECT().
		Upload(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(3).
		Return(azlogs.UploadResponse{}, errors.New("test"))

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
					DataCollectionRuleId: uuid.NewDummy().String(),
					StreamName:           streamName,
				},
			},
			ExpectedErrorMsg: "",
			ValidateSecret:   true,
		},
		"Test invalid config": {
			Config:           &storage.MicrosoftSentinel{},
			ExpectedErrorMsg: "Microsoft Sentinel validation errors: [Log Ingestion Endpoint must be specified, Data Collection Rule Id must be specified, Stream Name must be specified, Directory Tenant Id must be specified, Application Client Id must be specified, Secret must be specified]",
			ValidateSecret:   true,
		},
		"Test invalid config without secret": {
			Config:                      &storage.MicrosoftSentinel{},
			ExpectedErrorMsg:            "Microsoft Sentinel validation errors: [Log Ingestion Endpoint must be specified, Data Collection Rule Id must be specified, Stream Name must be specified, Directory Tenant Id must be specified, Application Client Id must be specified]",
			ExpectedErrorMsgNotContains: "secret",
			ValidateSecret:              false,
		},
	}

	for name, testCase := range testCases {
		suite.T().Run(name, func(t *testing.T) {
			err := Validate(testCase.Config, testCase.ValidateSecret)
			if testCase.ExpectedErrorMsg == "" {
				assert.NoError(t, err)
			} else {
				assert.NotContains(t, testCase.ExpectedErrorMsgNotContains, err.Error())
				assert.Contains(t, testCase.ExpectedErrorMsg, err.Error())
			}
		})
	}
}

func (suite *SentinelTestSuite) TestTestAlert() {
	notifier := &sentinel{
		azlogsClient: suite.mockAzureClient,
		notifier:     getNotifierConfig(),
	}

	testAlert := notifier.getTestAlert()
	bytesToSend, err := notifier.prepareLogsToSend(testAlert)
	suite.Require().NoError(err)

	suite.mockAzureClient.EXPECT().Upload(gomock.Any(), uuid.NewDummy().String(), streamName, bytesToSend, gomock.Any()).Times(1)

	notifierErr := notifier.Test(context.TODO())
	suite.Require().Nil(notifierErr)
}

func getNotifierConfig() *storage.Notifier {
	return &storage.Notifier{
		Name: "microsoft-sentinel",
		Config: &storage.Notifier_MicrosoftSentinel{
			MicrosoftSentinel: &storage.MicrosoftSentinel{
				LogIngestionEndpoint: "portal.azure.com",
				AlertDcrConfig: &storage.MicrosoftSentinel_DataCollectionRuleConfig{
					DataCollectionRuleId: uuid.NewDummy().String(),
					StreamName:           streamName,
				},
			},
		},
	}
}
