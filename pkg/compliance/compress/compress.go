package compress

import (
	"github.com/stackrox/rox/generated/internalapi/compliance"
)

// ResultWrapper wraps a map of ComplianceStandardResults in an easyjson wrapper
// easyjson:json
type ResultWrapper struct {
	ResultMap map[string]*compliance.ComplianceStandardResult
}
