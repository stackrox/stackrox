package microsoftsentinel

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stackrox/rox/central/notifiers/microsoftsentinel/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/protocompat"
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

	mockCtrl *gomock.Controller
}

func (suite *SentinelTestSuite) SetupTest() {
	suite.mockCtrl = gomock.NewController(suite.T())
	suite.T().Setenv(features.MicrosoftSentinelNotifier.EnvVar(), "true")
}

func (suite *SentinelTestSuite) TestAlertNotify() {
	mockAzureClient := mocks.NewMockazureLogsClient(suite.mockCtrl)

	alert := &storage.Alert{
		Id:          uuid.NewDummy().String(),
		ClusterName: "test-cluster",
		Entity: &storage.Alert_Deployment_{
			Deployment: &storage.Alert_Deployment{
				Name: "nginx",
			},
		},
	}

	result, err := protocompat.MarshalMap(alert)
	require.NoError(suite.T(), err)

	// Sentinel expects logs to be sent as a JSON array.
	logsToSend, err := json.Marshal([]map[string]interface{}{result})
	require.NoError(suite.T(), err)

	s := &sentinel{
		azlogsClient: mockAzureClient,
		notifier: &storage.Notifier{
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
		},
	}

	// Assert call to library and marshalling is correct.
	mockAzureClient.EXPECT().Upload(gomock.Any(), uuid.NewDummy().String(), streamName, logsToSend, gomock.Any()).Times(1)
	require.NotNil(suite.T(), s)

	err = s.AlertNotify(context.Background(), alert)
	require.NoError(suite.T(), err)
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
