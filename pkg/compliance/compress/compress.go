package compress

import (
	"github.com/stackrox/stackrox/generated/internalapi/compliance"
)

// ResultWrapper wraps a map of ComplianceStandardResults in an easyjson wrapper
// easyjson:json
type ResultWrapper struct {
	ResultMap map[string]*compliance.ComplianceStandardResult
}
