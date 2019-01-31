package standards

import (
	"fmt"
	"sync"

	"github.com/stackrox/rox/generated/api/v1"
)

// Registry stores compliance standards by their ID.
type Registry struct {
	mutex             sync.RWMutex
	standardsByID     map[string]*Standard
	controlToCategory map[string]*Category
}

// NewRegistry creates and returns a new standards registry.
func NewRegistry() *Registry {
	return &Registry{
		standardsByID:     make(map[string]*Standard),
		controlToCategory: make(map[string]*Category),
	}
}

// RegisterStandard registers a given standard. This function returns an error if a different standard with the same
// ID is already registered.
func (r *Registry) RegisterStandard(standard *Standard) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	existingStandard := r.standardsByID[standard.ID]
	if existingStandard == nil {
		r.standardsByID[standard.ID] = standard
	} else if existingStandard != standard {
		return fmt.Errorf("different compliance standard with id %q already registered", standard.ID)
	}

	for i, category := range standard.Categories {
		for _, control := range category.Controls {
			r.controlToCategory[fmt.Sprintf("%s:%s", standard.ID, control.ID)] = &standard.Categories[i]
		}
	}

	return nil
}

// LookupStandard returns the standard object with the given ID.
func (r *Registry) LookupStandard(id string) *Standard {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return r.standardsByID[id]
}

// AllStandards returns all registered standards.
func (r *Registry) AllStandards() []*Standard {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	result := make([]*Standard, 0, len(r.standardsByID))
	for _, standard := range r.standardsByID {
		result = append(result, standard)
	}
	return result
}

// Standards returns the metadata protos for all registered compliance standards.
func (r *Registry) Standards() ([]*v1.ComplianceStandardMetadata, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	result := make([]*v1.ComplianceStandardMetadata, 0, len(r.standardsByID))
	for _, standard := range r.standardsByID {
		result = append(result, standard.MetadataProto())
	}
	return result, nil
}

// Standard returns the full proto definition of the compliance standard with the given ID.
func (r *Registry) Standard(id string) (*v1.ComplianceStandard, bool, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	standard := r.standardsByID[id]
	if standard == nil {
		return nil, false, nil
	}
	return standard.ToProto(), true, nil
}

// StandardMetadata returns the metadata proto for the compliance standard with the given ID.
func (r *Registry) StandardMetadata(id string) (*v1.ComplianceStandardMetadata, bool, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	standard := r.standardsByID[id]
	if standard == nil {
		return nil, false, nil
	}
	return standard.MetadataProto(), true, nil
}

// Controls returns the list of controls for the given compliance standard.
func (r *Registry) Controls(standardID string) ([]*v1.ComplianceControl, error) {
	standard, exists, err := r.Standard(standardID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("standard with ID %q not found", standardID)
	}

	controls := standard.GetControls()
	for _, control := range controls {
		control.StandardId = standard.GetMetadata().GetId()
	}
	return controls, nil
}

// Groups returns the list of groups for the given compliance standard
func (r *Registry) Groups(standardID string) ([]*v1.ComplianceControlGroup, error) {
	standard, exists, err := r.Standard(standardID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("standard with ID %q not found", standardID)
	}
	groups := standard.GetGroups()
	for _, group := range groups {
		group.StandardId = standard.GetMetadata().GetId()
	}
	return groups, nil
}

// GetCategoryByControl returns the category that corresponds to the passed control ID
func (r *Registry) GetCategoryByControl(controlID string) *Category {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return r.controlToCategory[controlID]
}
