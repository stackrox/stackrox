package standards

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/stackrox/rox/central/compliance/standards/index"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
)

var (
	standardRegistry = make(map[string]*Standard)
)

// RegisterStandard adds the standard to the standard registry
func RegisterStandard(s *Standard) error {
	if _, ok := standardRegistry[s.ID]; ok {
		return fmt.Errorf("Standard %s is already registered", s.ID)
	}
	standardRegistry[s.ID] = s
	return nil
}

// Registry stores compliance standards by their ID.
type Registry struct {
	mutex             sync.RWMutex
	standardsByID     map[string]*Standard
	controlToCategory map[string]*Category
	controls          map[string]*Control
	indexer           index.Indexer
}

// NewRegistry creates and returns a new standards registry.
func NewRegistry(indexer index.Indexer) *Registry {
	return &Registry{
		standardsByID:     make(map[string]*Standard),
		controlToCategory: make(map[string]*Category),
		controls:          make(map[string]*Control),
		indexer:           indexer,
	}
}

func getFullyQualifiedName(standardID, controlID string) string {
	return fmt.Sprintf("%s:%s", standardID, controlID)
}

// RegisterStandards registers all of the standards in the standard registry
func (r *Registry) RegisterStandards() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	for _, standard := range standardRegistry {
		existingStandard := r.standardsByID[standard.ID]
		if existingStandard == nil {
			r.standardsByID[standard.ID] = standard
		} else if existingStandard != standard {
			return fmt.Errorf("different compliance standard with id %q already registered", standard.ID)
		}

		for i, category := range standard.Categories {
			for _, control := range category.Controls {
				fqn := getFullyQualifiedName(standard.ID, control.ID)
				r.controlToCategory[fqn] = &standard.Categories[i]
				if err := r.registerControl(standard.ID, control); err != nil {
					return err
				}
			}
		}
		if err := r.indexer.IndexStandard(standard.ToProto()); err != nil {
			return err
		}
	}
	return nil
}

// registerControl does not need to be locked, because it is locked by registerStandard
func (r *Registry) registerControl(fqn string, control Control) error {
	if existingControl, ok := r.controls[control.ID]; !ok {
		r.controls[fqn] = &control
	} else if *existingControl != control {
		return fmt.Errorf("different compliance control with id %q already registered", fqn)
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

func (r *Registry) controlByID(id string) *Control {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return r.controls[id]
}

// SearchStandards searches across standards
func (r *Registry) SearchStandards(q *v1.Query) ([]search.Result, error) {
	return r.indexer.SearchStandards(q)
}

// SearchControls searches across controls
func (r *Registry) SearchControls(q *v1.Query) ([]search.Result, error) {
	return r.indexer.SearchControls(q)
}
