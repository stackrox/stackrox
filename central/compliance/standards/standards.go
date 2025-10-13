package standards

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
)

// Repository is an interface for a collection of standards and controls
type Repository interface {
	Standards() ([]*v1.ComplianceStandardMetadata, error)
	StandardMetadata(id string) (*v1.ComplianceStandardMetadata, bool, error)
	Standard(id string) (*v1.ComplianceStandard, bool, error)
	Controls(standardID string) ([]*v1.ComplianceControl, error)
	Control(controlID string) *v1.ComplianceControl
	GetCategoryByControl(controlID string) *Category
	Groups(standardID string) ([]*v1.ComplianceControlGroup, error)
	Group(groupID string) *v1.ComplianceControlGroup
	GetCISKubernetesStandardID() (string, error)

	SearchStandards(q *v1.Query) ([]search.Result, error)
	SearchControls(q *v1.Query) ([]search.Result, error)
}
