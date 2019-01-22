package standards

import "github.com/stackrox/rox/generated/api/v1"

// Control is the metadata of a control in a compliance standard.
type Control struct {
	ID          string
	Name        string
	Description string
}

// ToProto returns the proto representation of a control.
func (c *Control) ToProto() *v1.ComplianceControl {
	return &v1.ComplianceControl{
		Id:          c.ID,
		Name:        c.Name,
		Description: c.Description,
	}
}

// Category represents a category in a compliance standard.
type Category struct {
	ID          string
	Name        string
	Description string

	Controls []Control
}

// ToProto returns the proto representation of the category's metadata.
func (c *Category) ToProto() *v1.ComplianceControlGroup {
	return &v1.ComplianceControlGroup{
		Id:          c.ID,
		Name:        c.Name,
		Description: c.Description,
	}
}

// Standard represents a compliance standard.
type Standard struct {
	ID          string
	Name        string
	Description string

	Categories []Category
}

// AllControlIDs returns the IDs for all controls in this check, either qualified or not.
func (s *Standard) AllControlIDs(qualified bool) []string {
	qualifierPrefix := ""
	if qualified {
		qualifierPrefix = s.ID + ":"
	}
	var result []string
	for _, cat := range s.Categories {
		for _, ctrl := range cat.Controls {
			result = append(result, qualifierPrefix+ctrl.ID)
		}
	}
	return result
}

// MetadataProto returns the proto representation of the standard's metadata.
func (s *Standard) MetadataProto() *v1.ComplianceStandardMetadata {
	return &v1.ComplianceStandardMetadata{
		Id:          s.ID,
		Name:        s.Name,
		Description: s.Description,
	}
}

// ToProto returns the proto definition of the entire compliance standard.
func (s *Standard) ToProto() *v1.ComplianceStandard {
	groups := make([]*v1.ComplianceControlGroup, 0, len(s.Categories))
	var controls []*v1.ComplianceControl

	for _, category := range s.Categories {
		groups = append(groups, category.ToProto())
		for _, control := range category.Controls {
			controlProto := control.ToProto()
			controlProto.GroupId = category.ID
			controls = append(controls, controlProto)
		}
	}

	return &v1.ComplianceStandard{
		Metadata: s.MetadataProto(),
		Groups:   groups,
		Controls: controls,
	}
}
