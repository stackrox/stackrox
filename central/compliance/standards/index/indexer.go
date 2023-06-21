package index

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
)

var (
	// StandardOptions is the search options map for a compliance standard
	StandardOptions = search.Walk(v1.SearchCategory_COMPLIANCE_STANDARD, "standard", (*v1.ComplianceStandard)(nil))
	// ControlOptions is the search options map for a compliance control
	ControlOptions = search.Walk(v1.SearchCategory_COMPLIANCE_CONTROL, "control", (*v1.ComplianceControl)(nil))
)
