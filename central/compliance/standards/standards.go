package standards

import "github.com/stackrox/rox/generated/api/v1"

// Standards is an interface for a collection of standards and controls
type Standards interface {
	Standards() ([]*v1.ComplianceStandardMetadata, error)
	Standard(id string) (*v1.ComplianceStandardMetadata, bool, error)
	Controls(standardID string) ([]*v1.ComplianceControl, error)
}
