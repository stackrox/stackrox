package awssh

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/securityhub"
)

// Client is a subset of the AWS security hub client.
//
//go:generate mockgen-wrapper
type Client interface {
	BatchImportFindings(context.Context, *securityhub.BatchImportFindingsInput, ...func(*securityhub.Options)) (*securityhub.BatchImportFindingsOutput, error)
	GetFindings(context.Context, *securityhub.GetFindingsInput, ...func(*securityhub.Options)) (*securityhub.GetFindingsOutput, error)
}
