package reportgenerator

import (
	"context"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
)

// ReportRequest contains information needed to generate and notify a report
type ReportRequest struct {
	ReportConfig   *storage.ReportConfiguration
	ReportMetadata *storage.ReportMetadata
	Collection     *storage.ResourceCollection
	DataStartTime  *types.Timestamp
	Ctx            context.Context
}

type reportEmailFormat struct {
	BrandedProductName string
	WhichVulns         string
	DateStr            string
	ImageTypes         string
}
