package reportgenerator

import (
	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
)

// ReportRequest contains information needed to generate and notify a report
type ReportRequest struct {
	ReportSnapshot *storage.ReportSnapshot
	Collection     *storage.ResourceCollection
	DataStartTime  *types.Timestamp
}

type reportEmailBodyFormat struct {
	BrandedPrefix string
}

type reportEmailSubjectFormat struct {
	BrandedProductNameShort string
	ReportConfigName        string
	CollectionName          string
}
