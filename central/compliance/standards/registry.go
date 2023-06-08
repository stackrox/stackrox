package standards

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/compliance/framework"
	pgControl "github.com/stackrox/rox/central/compliance/standards/control"
	"github.com/stackrox/rox/central/compliance/standards/index"
	"github.com/stackrox/rox/central/compliance/standards/metadata"
	pgStandard "github.com/stackrox/rox/central/compliance/standards/standard"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/sync"
)

// Registry stores compliance standards by their ID.
type Registry struct {
	lock           sync.RWMutex
	standardsByID  map[string]*Standard
	categoriesByID map[string]*Category
	controlsByID   map[string]*Control

	standardStore   pgStandard.Store
	standardIndexer index.StandardIndexer

	controlStore   pgControl.Store
	controlIndexer index.ControlIndexer

	checkRegistry framework.CheckRegistry
}

// NewRegistry creates and returns a new standards registry.
func NewRegistry(standardStore pgStandard.Store, standardIndexer index.StandardIndexer, controlStore pgControl.Store, controlIndexer index.ControlIndexer, checkRegistry framework.CheckRegistry, standardMDs ...metadata.Standard) (*Registry, error) {
	r := &Registry{
		standardStore:   standardStore,
		standardIndexer: standardIndexer,
		controlStore:    controlStore,
		controlIndexer:  controlIndexer,

		standardsByID:  make(map[string]*Standard),
		categoriesByID: make(map[string]*Category),
		controlsByID:   make(map[string]*Control),
		checkRegistry:  checkRegistry,
	}
	ctx := sac.WithAllAccess(context.Background())

	// Standards and controls are completely upserted by the call to Register Standards
	if err := standardStore.DeleteByQuery(ctx, search.NewQueryBuilder().AddStrings(search.StandardID, search.WildcardString).ProtoQuery()); err != nil {
		return nil, err
	}
	if err := controlStore.DeleteByQuery(ctx, search.NewQueryBuilder().AddStrings(search.ControlID, search.WildcardString).ProtoQuery()); err != nil {
		return nil, err
	}

	if err := r.RegisterStandards(ctx, standardMDs...); err != nil {
		return nil, err
	}
	return r, nil
}

// RegisterCheck adds a check to the registry
func (r *Registry) RegisterCheck(check framework.Check) error {
	return r.checkRegistry.Register(check)
}

// RegisterStandards registers all of the standards in the standard registry
func (r *Registry) RegisterStandards(ctx context.Context, standardMDs ...metadata.Standard) error {
	for _, standardMD := range standardMDs {
		if err := r.RegisterStandard(ctx, standardMD, false); err != nil {
			return errors.Wrapf(err, "registering standard %q", standardMD.ID)
		}
	}
	return nil
}

// DeleteStandard removes a standard from the registry
func (r *Registry) DeleteStandard(ctx context.Context, id string) error {
	r.lock.Lock()
	defer r.lock.Unlock()
	standard := r.standardsByID[id]
	if standard == nil {
		return nil
	}
	delete(r.standardsByID, id)

	if err := r.standardStore.Delete(ctx, id); err != nil {
		return err
	}

	for id := range r.controlsByID {
		if ChildOfStandard(id, standard.ID) {
			delete(r.controlsByID, id)

			if err := r.controlStore.Delete(ctx, id); err != nil {
				return err
			}
			r.checkRegistry.Delete(id)
		}
	}
	for id := range r.categoriesByID {
		if ChildOfStandard(id, standard.ID) {
			delete(r.categoriesByID, id)
		}
	}
	return nil
}

// DeleteControl removes a control from the registry
func (r *Registry) DeleteControl(ctx context.Context, id string) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	r.checkRegistry.Delete(id)
	delete(r.controlsByID, id)

	if err := r.controlStore.Delete(ctx, id); err != nil {
		return err
	}
	return nil
}

// RegisterStandard registers an individual standard
func (r *Registry) RegisterStandard(ctx context.Context, standardMD metadata.Standard, overwrite bool) error {
	r.lock.Lock()
	defer r.lock.Unlock()
	if _, existing := r.standardsByID[standardMD.ID]; existing && !overwrite {
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

	storageStandard := standard.ToStorageProto()
	if err := r.standardStore.Upsert(ctx, storageStandard); err != nil {
		return err
	}
	return r.controlStore.UpsertMany(ctx, storageStandard.GetControls())
}

// LookupStandard returns the standard object with the given ID.
func (r *Registry) LookupStandard(id string) *Standard {
	r.lock.RLock()
	defer r.lock.RUnlock()
	return r.standardsByID[id]
}

// AllStandards returns all registered standards.
func (r *Registry) AllStandards() []*Standard {
	r.lock.RLock()
	defer r.lock.RUnlock()
	result := make([]*Standard, 0, len(r.standardsByID))
	for _, standard := range r.standardsByID {
		result = append(result, standard)
	}
	return result
}

// Standards returns the metadata protos for all registered compliance standards.
func (r *Registry) Standards() ([]*v1.ComplianceStandardMetadata, error) {
	r.lock.RLock()
	defer r.lock.RUnlock()
	result := make([]*v1.ComplianceStandardMetadata, 0, len(r.standardsByID))
	for _, standard := range r.standardsByID {
		result = append(result, standard.V1MetadataProto())
	}
	return result, nil
}

// Standard returns the full proto definition of the compliance standard with the given ID.
func (r *Registry) Standard(id string) (*v1.ComplianceStandard, bool, error) {
	r.lock.RLock()
	defer r.lock.RUnlock()
	standard := r.standardsByID[id]
	if standard == nil {
		return nil, false, nil
	}
	return standard.ToV1Proto(), true, nil
}

// StandardMetadata returns the metadata proto for the compliance standard with the given ID.
func (r *Registry) StandardMetadata(id string) (*v1.ComplianceStandardMetadata, bool, error) {
	r.lock.RLock()
	defer r.lock.RUnlock()
	standard := r.standardsByID[id]
	if standard == nil {
		return nil, false, nil
	}
	return standard.V1MetadataProto(), true, nil
}

// Controls returns the list of controls for the given compliance standard.
func (r *Registry) Controls(standardID string) ([]*v1.ComplianceControl, error) {
	r.lock.RLock()
	defer r.lock.RUnlock()
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
	r.lock.RLock()
	defer r.lock.RUnlock()
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
	r.lock.RLock()
	defer r.lock.RUnlock()
	ctrl := r.controlsByID[controlID]
	if ctrl == nil {
		return nil
	}
	return ctrl.Category
}

// Control returns the control for the ID, if it matches.
func (r *Registry) Control(controlID string) *v1.ComplianceControl {
	r.lock.RLock()
	defer r.lock.RUnlock()
	return r.controlsByID[controlID].ToV1Proto()
}

// Group returns the proto object for a single group
func (r *Registry) Group(groupID string) *v1.ComplianceControlGroup {
	r.lock.RLock()
	defer r.lock.RUnlock()
	return r.categoriesByID[groupID].ToV1Proto()
}

// GetCISDockerStandardID returns the Docker CIS standard ID.
func (r *Registry) GetCISDockerStandardID() (string, error) {
	r.lock.RLock()
	defer r.lock.RUnlock()
	for _, standard := range r.standardsByID {
		if strings.Contains(standard.Name, "CIS Docker") {
			return standard.ID, nil
		}
	}
	return "", errors.New("Unable to find CIS Docker standard")
}

// GetCISKubernetesStandardID returns the kubernetes CIS standard ID.
func (r *Registry) GetCISKubernetesStandardID() (string, error) {
	r.lock.RLock()
	defer r.lock.RUnlock()
	for _, standard := range r.standardsByID {
		if strings.Contains(standard.Name, "CIS Kubernetes") {
			return standard.ID, nil
		}
	}
	return "", errors.New("Unable to find CIS Kubernetes standard")
}

// SearchStandards searches across standards
func (r *Registry) SearchStandards(ctx context.Context, q *v1.Query) ([]search.Result, error) {
	r.lock.RLock()
	defer r.lock.RUnlock()

	return r.standardIndexer.Search(ctx, q)
}

// SearchControls searches across controls
func (r *Registry) SearchControls(ctx context.Context, q *v1.Query) ([]search.Result, error) {
	r.lock.RLock()
	defer r.lock.RUnlock()

	return r.controlIndexer.Search(ctx, q)
}
