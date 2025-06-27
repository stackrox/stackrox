package acscsemail

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/stackrox/rox/central/notifiers/acscsemail/message"
	acscsMocks "github.com/stackrox/rox/central/notifiers/acscsemail/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/branding"
	mitreMocks "github.com/stackrox/rox/pkg/mitre/datastore/mocks"
	"github.com/stackrox/rox/pkg/notifiers/mocks"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.yaml.in/yaml/v3"
)

func TestReportNotify(t *testing.T) {
	mockController := gomock.NewController(t)
	clientMock := acscsMocks.NewMockClient(mockController)

	acscsEmail := &acscsEmail{
		client: clientMock,
	}

	var attachBuf bytes.Buffer
	attachBuf.WriteString("test attachement")

	expectTo := []string{"test@test.acscs-mail-test.com"}
	expectedSubject := "Test Email"

	var actualMsg message.AcscsEmail
	clientMock.EXPECT().SendMessage(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, msg message.AcscsEmail) error {
		actualMsg = msg
		return nil
	})
	sampleReportName := "Test Report"

	err := acscsEmail.ReportNotify(context.Background(), &attachBuf, expectTo, expectedSubject, "here is your report", sampleReportName)
	mockController.Finish()

	require.NoError(t, err, "unexpected error for ReportNotify")
	assert.Equal(t, expectTo, actualMsg.To)

	// assert raw message has expected SMTP data, and no From,To,Subject headers
	msgStr := string(actualMsg.RawMessage)

	assert.Contains(t, msgStr, "Subject: Test Email\r\n")
	expectedFilePrefix := fmt.Sprintf("%s_Test_Report", branding.GetProductNameShort())
	// report file
	assert.Contains(t, msgStr, "Content-Type: multipart/mixed;")
	assert.Contains(t, msgStr, "Content-Type: application/zip\r\n")
	assert.Contains(t, msgStr, "Content-Transfer-Encoding: base64\r\n")
	assert.Contains(t, msgStr, fmt.Sprintf("Content-Disposition: attachment; filename=%s", expectedFilePrefix))
	assert.Contains(t, msgStr, base64.StdEncoding.EncodeToString(attachBuf.Bytes()))

	// logo file
	assert.Contains(t, msgStr, "Content-Type: image/png; name=logo.png\r\n")
	assert.Contains(t, msgStr, "Content-Transfer-Encoding: base64\r\n")
	assert.Contains(t, msgStr, "Content-Disposition: inline; filename=logo.png\r\n")
	assert.Contains(t, msgStr, "Content-ID: <logo.png>\r\n")
	assert.Contains(t, msgStr, "X-Attachment-Id: logo.png\r\n")

	// Does not contain headers that are expected to be set by acscs email service
	assert.NotContains(t, msgStr, "From:")
	assert.NotContains(t, msgStr, "To:")

	// text message
	assert.Contains(t, msgStr, "here is your report")

}

func TestAlertNotify(t *testing.T) {
	inputAlert := storage.Alert{
		Id: "test-id",
		Policy: &storage.Policy{
			Name: "test-policy",
		},
		Time: protocompat.TimestampNow(),
	}
	expectedTo := "default@test.acscs-email-test.com"
	expectedSubject := "Policy 'test-policy' violated"

	mockController := gomock.NewController(t)
	metadataGetter := mocks.NewMockMetadataGetter(mockController)
	metadataGetter.EXPECT().GetAnnotationValue(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(expectedTo)

	mockClient := acscsMocks.NewMockClient(mockController)
	var actualMsg message.AcscsEmail
	mockClient.EXPECT().SendMessage(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, msg message.AcscsEmail) error {
		actualMsg = msg
		return nil
	})

	mitreStore := mitreMocks.NewMockAttackReadOnlyDataStore(mockController)
	mitreStore.EXPECT().Get(gomock.Any()).Return(&storage.MitreAttackVector{}, nil).AnyTimes()

	acscsEmail := &acscsEmail{
		client:         mockClient,
		metadataGetter: metadataGetter,
		notifier: &storage.Notifier{
			LabelDefault: "default@test.acscs-email-test.com",
		},
	}

	err := acscsEmail.AlertNotify(context.Background(), &inputAlert)
	require.NoError(t, err, "unexpected error on AlertNotify")
	mockController.Finish()

	assert.Equal(t, []string{expectedTo}, actualMsg.To)

	msgStr := string(actualMsg.RawMessage)

	assert.Contains(t, msgStr, fmt.Sprintf("Subject: %s\r\n", expectedSubject))
	assert.Contains(t, msgStr, "Content-Type: text/plain")
	assert.Contains(t, msgStr, fmt.Sprintf("Alert ID: %s", inputAlert.Id))
	assert.NotContains(t, msgStr, "From:")
	assert.NotContains(t, msgStr, "To:")
}

func TestNetworkPolicyYAMLNotify(t *testing.T) {
	mockController := gomock.NewController(t)
	mockClient := acscsMocks.NewMockClient(mockController)

	var actualMsg message.AcscsEmail
	mockClient.EXPECT().SendMessage(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, msg message.AcscsEmail) error {
		actualMsg = msg
		return nil
	})

	expectedTo := "default@test.acscs-email-test.com"
	expectedClusterName := "test-cluster"
	expectedSubject := fmt.Sprintf("New network policy YAML for cluster '%s' needs to be applied", expectedClusterName)

	acscsEmail := &acscsEmail{
		client: mockClient,
		notifier: &storage.Notifier{
			LabelDefault: expectedTo,
		},
	}

	exampleYamlObject := struct {
		Test1 string
		Test2 string
	}{
		Test1: "testValue1",
		Test2: "testValue2",
	}

	yamlBytes, err := yaml.Marshal(&exampleYamlObject)
	require.NoError(t, err, "unexpected error on marshaling test object")
	err = acscsEmail.NetworkPolicyYAMLNotify(context.Background(), string(yamlBytes), expectedClusterName)
	require.NoError(t, err, "unexpedted error on NetworkPolicyYAMLNotify")
	mockController.Finish()

	assert.Equal(t, []string{expectedTo}, actualMsg.To)

	msgStr := string(actualMsg.RawMessage)

	assert.Contains(t, msgStr, fmt.Sprintf("Subject: %s", expectedSubject))
	assert.Contains(t, msgStr, "test1: testValue1")
	assert.Contains(t, msgStr, "test2: testValue2")

	assert.NotContains(t, msgStr, "From:")
	assert.NotContains(t, msgStr, "To:")
}

func TestACSCSEmailTest(t *testing.T) {
	mockController := gomock.NewController(t)
	mockClient := acscsMocks.NewMockClient(mockController)

	var actualMsg message.AcscsEmail
	mockClient.EXPECT().SendMessage(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, msg message.AcscsEmail) error {
		actualMsg = msg
		return nil
	})

	expectedTo := "default@test.acscs-email-test.com"

	acscsEmail := &acscsEmail{
		client: mockClient,
		notifier: &storage.Notifier{
			LabelDefault: expectedTo,
		},
	}

	err := acscsEmail.Test(context.Background())
	require.Nil(t, err, "unexpected error on Test function")
	mockController.Finish()

	assert.Equal(t, []string{expectedTo}, actualMsg.To)

	msgStr := string(actualMsg.RawMessage)
	assert.Contains(t, msgStr, "Subject: RHACS Cloud Service Test Email")
}
