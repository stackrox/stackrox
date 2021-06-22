package standards

import (
	"sort"

	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/central/compliance/standards/metadata"
	v1 "github.com/stackrox/rox/generated/api/v1"
	pkgFramework "github.com/stackrox/rox/pkg/compliance/framework"
	"github.com/stackrox/rox/pkg/set"
)

// Standard contains information about a compliance standard.
type Standard struct {
	metadata.Standard
	allChecks   []framework.Check
	allDataDeps set.StringSet
	scopes      []pkgFramework.TargetKind

	categories map[string]*Category
	controls   map[string]*Control
}

// LookupCategory retrieves a category from this standard by its unqualified ID.
func (s *Standard) LookupCategory(unqualifiedID string) *Category {
	return s.categories[unqualifiedID]
}

// AllCategories returns all categories in a deterministic order.
func (s *Standard) AllCategories() []*Category {
	allCategories := make([]*Category, 0, len(s.categories))
	for _, cat := range s.categories {
		allCategories = append(allCategories, cat)
	}
	sort.Slice(allCategories, func(i, j int) bool {
		return allCategories[i].ID < allCategories[j].ID
	})
	return allCategories
}

// LookupControl retrieves a control from this standard by its unqualified ID.
func (s *Standard) LookupControl(unqualifiedID string) *Control {
	return s.controls[unqualifiedID]
}

// AllControls returns all controls in this standard in a deterministic order.
func (s *Standard) AllControls() []*Control {
	allControls := make([]*Control, 0, len(s.controls))
	for _, control := range s.controls {
		allControls = append(allControls, control)
	}
	sortControls(allControls)
	return allControls
}

// AllChecks returns all implemented checks for this compliance standard in a deterministic order.
func (s *Standard) AllChecks() []framework.Check {
	return s.allChecks
}

func (s *Standard) protoScopes() []v1.ComplianceStandardMetadata_Scope {
	scopes := []v1.ComplianceStandardMetadata_Scope{
		v1.ComplianceStandardMetadata_CLUSTER,
	}
	for _, s := range s.scopes {
		switch s {
		case pkgFramework.DeploymentKind:
			scopes = append(scopes, v1.ComplianceStandardMetadata_DEPLOYMENT)
			scopes = append(scopes, v1.ComplianceStandardMetadata_NAMESPACE)
		case pkgFramework.NodeKind:
			scopes = append(scopes, v1.ComplianceStandardMetadata_NODE)
		}
	}
	sort.Slice(scopes, func(i, j int) bool {
		return scopes[i] < scopes[j]
	})
	return scopes
}

// MetadataProto returns the proto representation of the standard's metadata.
func (s *Standard) MetadataProto() *v1.ComplianceStandardMetadata {
	return &v1.ComplianceStandardMetadata{
		Id:                   s.ID,
		Name:                 s.Name,
		Description:          s.Description,
		NumImplementedChecks: int32(len(s.allChecks)),
		Scopes:               s.protoScopes(),
		Dynamic:              s.Dynamic,
	}
}

// ToProto returns the proto definition of the entire compliance standard.
func (s *Standard) ToProto() *v1.ComplianceStandard {
	groups := make([]*v1.ComplianceControlGroup, 0, len(s.Categories))
	var controls []*v1.ComplianceControl

	for _, category := range s.AllCategories() {
		groups = append(groups, category.ToProto())
		for _, control := range category.AllControls() {
			controls = append(controls, control.ToProto())
		}
	}

	return &v1.ComplianceStandard{
		Metadata: s.MetadataProto(),
		Groups:   groups,
		Controls: controls,
	}
}

// HasAnyDataDependency checks if the given standard requires at least one of the given data dependencies.
func (s *Standard) HasAnyDataDependency(deps ...string) bool {
	for _, dep := range deps {
		if s.allDataDeps.Contains(dep) {
			return true
		}
	}
	return false
}

// AllDataDependencies returns all data dependencies of all checks in this standard in sorted order.
func (s *Standard) AllDataDependencies() []string {
	return s.allDataDeps.AsSortedSlice(func(i, j string) bool { return i < j })
}

// Category contains information about a compliance control category.
type Category struct {
	metadata.Category

	Standard *Standard

	controls  map[string]*Control
	allChecks []framework.Check

	qualifiedID string
}

// QualifiedID returns the qualified ID of this category.
func (c *Category) QualifiedID() string {
	if c.qualifiedID == "" {
		c.qualifiedID = BuildQualifiedID(c.Standard.ID, c.ID)
	}
	return c.qualifiedID
}

// LookupControl retrieves a control from this category by its unqualified ID.
func (c *Category) LookupControl(unqualifiedID string) *Control {
	return c.controls[unqualifiedID]
}

// AllControls returns all controls in this category in a deterministic order.
func (c *Category) AllControls() []*Control {
	allControls := make([]*Control, 0, len(c.controls))
	for _, control := range c.controls {
		allControls = append(allControls, control)
	}
	sortControls(allControls)
	return allControls
}

// ToProto returns the proto representation of the category's metadata.
func (c *Category) ToProto() *v1.ComplianceControlGroup {
	if c == nil {
		return nil
	}
	return &v1.ComplianceControlGroup{
		Id:                   c.QualifiedID(),
		StandardId:           c.Standard.ID,
		Name:                 c.Name,
		Description:          c.Description,
		NumImplementedChecks: int32(len(c.allChecks)),
	}
}

// Control contains information about a compliance control.
type Control struct {
	metadata.Control

	Standard *Standard
	Category *Category

	Check framework.Check

	qualifiedID string
}

// QualifiedID returns the qualified ID of this control.
func (c *Control) QualifiedID() string {
	if c.qualifiedID == "" {
		c.qualifiedID = BuildQualifiedID(c.Standard.ID, c.ID)
	}
	return c.qualifiedID
}

// ToProto returns the proto representation of a control.
func (c *Control) ToProto() *v1.ComplianceControl {
	if c == nil {
		return nil
	}
	var interpretationText string
	if c.Check != nil {
		interpretationText = c.Check.InterpretationText()
	}
	return &v1.ComplianceControl{
		Id:                 c.QualifiedID(),
		StandardId:         c.Standard.ID,
		GroupId:            c.Category.QualifiedID(),
		Name:               c.Name,
		Description:        c.Description,
		Implemented:        c.Check != nil,
		InterpretationText: interpretationText,
	}
}

func newStandard(standardMD metadata.Standard, checkRegistry framework.CheckRegistry) *Standard {
	s := &Standard{
		Standard:   standardMD,
		categories: make(map[string]*Category),
		controls:   make(map[string]*Control),
	}

	scopeMap := make(map[pkgFramework.TargetKind]struct{})

	for _, categoryMD := range standardMD.Categories {
		cat := &Category{
			Category: categoryMD,
			Standard: s,
			controls: make(map[string]*Control),
		}

		for _, controlMD := range categoryMD.Controls {
			ctrl := &Control{
				Control:  controlMD,
				Standard: s,
				Category: cat,
			}

			if checkRegistry != nil {
				ctrl.Check = checkRegistry.Lookup(ctrl.QualifiedID())
				if ctrl.Check != nil {
					scopeMap[ctrl.Check.Scope()] = struct{}{}
				}
			}
			if ctrl.Check != nil {
				cat.allChecks = append(cat.allChecks, ctrl.Check)
				s.allChecks = append(s.allChecks, ctrl.Check)
			}

			cat.controls[controlMD.ID] = ctrl
			s.controls[controlMD.ID] = ctrl
		}
		sortChecks(cat.allChecks)

		s.categories[categoryMD.ID] = cat
	}
	s.allDataDeps = gatherDataDependencies(s.allChecks)
	sortChecks(s.allChecks)

	scopes := make([]pkgFramework.TargetKind, 0, len(scopeMap))
	for s := range scopeMap {
		scopes = append(scopes, s)
	}
	sort.Slice(scopes, func(i, j int) bool {
		return scopes[i] < scopes[j]
	})
	s.scopes = scopes

	return s
}

func sortControls(controls []*Control) {
	sort.Slice(controls, func(i, j int) bool {
		return controls[i].ID < controls[j].ID
	})
}

func sortChecks(checks []framework.Check) {
	sort.Slice(checks, func(i, j int) bool {
		return checks[i].ID() < checks[j].ID()
	})
}
