package standards

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/central/compliance/standards/index"
	"github.com/stackrox/rox/central/compliance/standards/metadata"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/sync"
)

// Registry stores compliance standards by their ID.
type Registry struct {
	mutex          sync.RWMutex
	standardsByID  map[string]*Standard
	categoriesByID map[string]*Category
	controlsByID   map[string]*Control

	indexer       index.Indexer
	checkRegistry framework.CheckRegistry
}

// NewRegistry creates and returns a new standards registry.
func NewRegistry(indexer index.Indexer, checkRegistry framework.CheckRegistry) *Registry {
	r := &Registry{
		standardsByID:  make(map[string]*Standard),
		categoriesByID: make(map[string]*Category),
		controlsByID:   make(map[string]*Control),
		indexer:        indexer,
		checkRegistry:  checkRegistry,
	}
	return r
}

// RegisterStandards registers all of the standards in the standard registry
func (r *Registry) RegisterStandards(standardMDs ...metadata.Standard) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	for _, standardMD := range standardMDs {
		if err := r.registerStandardNoLock(standardMD); err != nil {
			return errors.Wrapf(err, "registering standard %q", standardMD.ID)
		}
	}

	return nil
}

func (r *Registry) registerStandardNoLock(standardMD metadata.Standard) error {
	if _, existing := r.standardsByID[standardMD.ID]; existing {
		return fmt.Errorf("compliance standard with ID %q already registered", standardMD.ID)
	}

	standard := newStandard(standardMD, r.checkRegistry)
	r.standardsByID[standard.ID] = standard

	for _, category := range standard.categories {
		r.categoriesByID[category.QualifiedID()] = category
	}
	for _, ctrl := range standard.controls {
		r.controlsByID[ctrl.QualifiedID()] = ctrl
	}

	if r.indexer == nil {
		return nil
	}
	return r.indexer.IndexStandard(standard.ToProto())
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

	ctrl := r.controlsByID[controlID]
	if ctrl == nil {
		return nil
	}
	return ctrl.Category
}

// Control returns the control for the ID, if it matches.
func (r *Registry) Control(controlID string) *v1.ComplianceControl {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return r.controlsByID[controlID].ToProto()
}

// Group returns the proto object for a single group
func (r *Registry) Group(groupID string) *v1.ComplianceControlGroup {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return r.categoriesByID[groupID].ToProto()
}

// GetCISDockerStandardID returns the Docker CIS standard ID.
func (r *Registry) GetCISDockerStandardID() (string, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	for _, standard := range r.standardsByID {
		if strings.Contains(standard.Name, "CIS Docker") {
			return standard.ID, nil
		}
	}
	return "", errors.New("Unable to find CIS Docker standard")
}

// GetCISKubernetesStandardID returns the kubernetes CIS standard ID.
func (r *Registry) GetCISKubernetesStandardID() (string, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	for _, standard := range r.standardsByID {
		if strings.Contains(standard.Name, "CIS Kubernetes") {
			return standard.ID, nil
		}
	}
	return "", errors.New("Unable to find CIS Kubernetes standard")
}

// SearchStandards searches across standards
func (r *Registry) SearchStandards(q *v1.Query) ([]search.Result, error) {
	return r.indexer.SearchStandards(q)
}

// SearchControls searches across controls
func (r *Registry) SearchControls(q *v1.Query) ([]search.Result, error) {
	return r.indexer.SearchControls(q)
}
