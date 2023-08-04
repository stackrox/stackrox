package reportgenerator

import (
	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
)

// ReportRequest contains information needed to generate and notify a report
type ReportRequest struct {
	ReportConfig   *storage.ReportConfiguration
	ReportSnapshot *storage.ReportSnapshot
	Collection     *storage.ResourceCollection
	DataStartTime  *types.Timestamp
}

type reportEmailFormat struct {
	BrandedProductName string
	WhichVulns         string
	DateStr            string
	ImageTypes         string
}
