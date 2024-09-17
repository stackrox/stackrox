package reportgenerator

import (
	"time"

	"github.com/stackrox/rox/generated/storage"
)

// ReportRequest contains information needed to generate and notify a report
type ReportRequest struct {
	ReportSnapshot *storage.ReportSnapshot
	Collection     *storage.ResourceCollection
	DataStartTime  time.Time
}

type reportEmailBodyFormat struct {
	BrandedPrefix string
}

type reportEmailSubjectFormat struct {
	BrandedProductNameShort string
	ReportConfigName        string
	CollectionName          string
}
