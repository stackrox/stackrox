package tests

import (
	"context"
	"os"
	"testing"
	"time"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/clientconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	slackTestChannelWebhookEnvVar = `SLACK_TEST_WEBHOOK`
)

var (
	notifierConfig = &v1.Notifier{
		Config: map[string]string{
			"channel": `#slack-test`,
		},
		Enabled:    true,
		Name:       slackNotifierName,
		Type:       `slack`,
		UiEndpoint: "http://localhost:8000",
	}
)

func init() {
	notifierConfig.Config["webhook"] = os.Getenv(slackTestChannelWebhookEnvVar)

}

func TestNotifierCRUD(t *testing.T) {
	require.NotEmpty(t, notifierConfig.Config["webhook"])

	conn, err := clientconn.UnauthenticatedGRPCConnection(apiEndpoint)
	require.NoError(t, err)

	service := v1.NewNotifierServiceClient(conn)

	subtests := []struct {
		name string
		test func(t *testing.T, service v1.NotifierServiceClient)
	}{
		{
			name: "create",
			test: verifyCreateNotifier,
		},
		{
			name: "read",
			test: verifyReadNotifier,
		},
		{
			name: "update",
			test: verifyUpdateNotifier,
		},
		{
			name: "delete",
			test: verifyDeleteNotifier,
		},
	}

	for _, sub := range subtests {
		t.Run(sub.name, func(t *testing.T) {
			sub.test(t, service)
		})
	}
}

func verifyCreateNotifier(t *testing.T, service v1.NotifierServiceClient) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	postResp, err := service.PostNotifier(ctx, notifierConfig)
	require.NoError(t, err)

	notifierConfig.Id = postResp.GetId()
	assert.Equal(t, notifierConfig, postResp)
}

func verifyReadNotifier(t *testing.T, service v1.NotifierServiceClient) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	getResp, err := service.GetNotifier(ctx, &v1.ResourceByID{Id: notifierConfig.GetId()})
	require.NoError(t, err)
	assert.Equal(t, notifierConfig, getResp)

	getManyResp, err := service.GetNotifiers(ctx, &v1.GetNotifiersRequest{Name: notifierConfig.GetName()})
	require.NoError(t, err)
	assert.Equal(t, 1, len(getManyResp.GetNotifiers()))
	if len(getManyResp.GetNotifiers()) > 0 {
		assert.Equal(t, notifierConfig, getManyResp.GetNotifiers()[0])
	}
}

func verifyUpdateNotifier(t *testing.T, service v1.NotifierServiceClient) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	notifierConfig.UiEndpoint = "http://localhost:3000"
	notifierConfig.Config["description"] = "A Slack Notifier"

	_, err := service.PutNotifier(ctx, notifierConfig)
	require.NoError(t, err)

	getResp, err := service.GetNotifier(ctx, &v1.ResourceByID{Id: notifierConfig.GetId()})
	require.NoError(t, err)
	assert.Equal(t, notifierConfig, getResp)
}

func verifyDeleteNotifier(t *testing.T, service v1.NotifierServiceClient) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	_, err := service.DeleteNotifier(ctx, &v1.DeleteNotifierRequest{Id: notifierConfig.GetId()})
	require.NoError(t, err)

	_, err = service.GetNotifier(ctx, &v1.ResourceByID{Id: notifierConfig.GetId()})
	s, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.NotFound, s.Code())
}
