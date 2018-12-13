package tests

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	slackTestChannelWebhookEnvVar = `SLACK_TEST_WEBHOOK`
)

var (
	notifierConfig = &storage.Notifier{
		Enabled:    true,
		Name:       slackNotifierName,
		Type:       `slack`,
		UiEndpoint: "http://localhost:8000",
	}
)

func init() {
	notifierConfig.LabelDefault = os.Getenv(slackTestChannelWebhookEnvVar)
}

func TestNotifierCRUD(t *testing.T) {
	require.NotEmpty(t, notifierConfig.LabelDefault)

	conn, err := grpcConnection()
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
	postResp, err := service.PostNotifier(ctx, notifierConfig)
	cancel()
	require.NoError(t, err)

	notifierConfig.Id = postResp.GetId()
	assert.Equal(t, notifierConfig, postResp)
}

func verifyReadNotifier(t *testing.T, service v1.NotifierServiceClient) {

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	getResp, err := service.GetNotifier(ctx, &v1.ResourceByID{Id: notifierConfig.GetId()})
	cancel()
	require.NoError(t, err)
	assert.Equal(t, notifierConfig, getResp)

	ctx, cancel = context.WithTimeout(context.Background(), time.Minute)
	getManyResp, err := service.GetNotifiers(ctx, &v1.GetNotifiersRequest{Name: notifierConfig.GetName()})
	cancel()
	require.NoError(t, err)
	assert.Equal(t, 1, len(getManyResp.GetNotifiers()))
	if len(getManyResp.GetNotifiers()) > 0 {
		assert.Equal(t, notifierConfig, getManyResp.GetNotifiers()[0])
	}
}

func verifyUpdateNotifier(t *testing.T, service v1.NotifierServiceClient) {
	notifierConfig.UiEndpoint = "http://localhost:3000"
	notifierConfig.Name += "1"

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	_, err := service.PutNotifier(ctx, notifierConfig)
	cancel()
	require.NoError(t, err)

	ctx, cancel = context.WithTimeout(context.Background(), time.Minute)
	getResp, err := service.GetNotifier(ctx, &v1.ResourceByID{Id: notifierConfig.GetId()})
	cancel()
	require.NoError(t, err)
	assert.Equal(t, notifierConfig, getResp)
}

func verifyDeleteNotifier(t *testing.T, service v1.NotifierServiceClient) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	_, err := service.DeleteNotifier(ctx, &v1.DeleteNotifierRequest{Id: notifierConfig.GetId()})
	cancel()
	require.NoError(t, err)

	ctx, cancel = context.WithTimeout(context.Background(), time.Minute)
	_, err = service.GetNotifier(ctx, &v1.ResourceByID{Id: notifierConfig.GetId()})
	cancel()
	s, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.NotFound, s.Code())
}
