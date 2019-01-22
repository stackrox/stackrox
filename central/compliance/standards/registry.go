package standards

import (
	"fmt"
	"sync"

	"github.com/stackrox/rox/generated/api/v1"
)

// Registry stores compliance standards by their ID.
type Registry struct {
	mutex         sync.RWMutex
	standardsByID map[string]*Standard
}

// NewRegistry creates and returns a new standards registry.
func NewRegistry() *Registry {
	return &Registry{
		standardsByID: make(map[string]*Standard),
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
