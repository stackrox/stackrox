package microsoftsentinel

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/monitor/ingestion/azlogs"
)

// azureLogsClient is a wrapper interface to enable testing the azure client.
//
//go:generate mockgen-wrapper
type azureLogsClient interface {
	Upload(ctx context.Context, ruleID string, streamName string, logs []byte, options *azlogs.UploadOptions) (azlogs.UploadResponse, error)
}

type azureLogsClientImpl struct {
	client *azlogs.Client
}

func (a *azureLogsClientImpl) Upload(ctx context.Context, ruleID string, streamName string, logs []byte, options *azlogs.UploadOptions) (azlogs.UploadResponse, error) {
	return a.client.Upload(ctx, ruleID, streamName, logs, options)
}
