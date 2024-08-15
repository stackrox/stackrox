package index

import v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"

// IndexReportWrap wraps a v4.IndexReport with additional fields required by Sensor and Central
type IndexReportWrap struct {
	NodeName    string
	NodeID      string
	IndexReport *v4.IndexReport
}
