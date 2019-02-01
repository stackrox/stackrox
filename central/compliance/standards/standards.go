package standards

import "github.com/stackrox/rox/generated/api/v1"

// Repository is an interface for a collection of standards and controls
type Repository interface {
	Standards() ([]*v1.ComplianceStandardMetadata, error)
	StandardMetadata(id string) (*v1.ComplianceStandardMetadata, bool, error)
	Standard(id string) (*v1.ComplianceStandard, bool, error)
	Controls(standardID string) ([]*v1.ComplianceControl, error)
	GetCategoryByControl(controlID string) *Category
	Groups(standardID string) ([]*v1.ComplianceControlGroup, error)
	GetCISDockerStandardID() (string, error)
	GetCISKubernetesStandardID() (string, error)
}
