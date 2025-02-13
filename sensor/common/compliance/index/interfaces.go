package index

import v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"

// Report wraps a v4.IndexReport with additional fields required by Central and Sensor.
type Report struct {
	NodeName    string
	NodeID      string
	IndexReport *v4.IndexReport
}
