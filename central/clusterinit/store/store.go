package store

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	// ErrInitBundleNotFound signals that a requested init bundle or CRS could not be located in the store.
	ErrInitBundleNotFound = errors.New("init bundle or CRS not found")

	// ErrInitBundleIDCollision signals that an init bundle or a CRS could not be added to the store due to an ID collision.
	ErrInitBundleIDCollision = errors.New("init bundle or CRS ID collision")

	// ErrInitBundleDuplicateName signals that an init bundle or a CRS could not be added because the name already exists for
	// a non-revoked init bundle or CRS.
	ErrInitBundleDuplicateName = errors.New("init bundle or CRS already exists")

	log = logging.LoggerForModule()
)

// Store interface for managing persisted cluster init bundles.
//
//go:generate mockgen-wrapper
type Store interface {
	GetAll(ctx context.Context) ([]*storage.InitBundleMeta, error)
	GetAllCRS(ctx context.Context) ([]*storage.InitBundleMeta, error)
	Get(ctx context.Context, id string) (*storage.InitBundleMeta, error)
	Add(ctx context.Context, bundleMeta *storage.InitBundleMeta) error
	Delete(ctx context.Context, id string) error
	Upsert(ctx context.Context, crs *storage.InitBundleMeta) error
	Revoke(ctx context.Context, id string) error
	InitiateClusterRegistration(ctx context.Context, initArtifactId, clusterName string) error
	MarkClusterRegistrationComplete(ctx context.Context, initArtifactId, clusterName string) error
}

// UnderlyingStore is the base store that actually accesses the data
type UnderlyingStore interface {
	Exists(ctx context.Context, id string) (bool, error)
	Get(ctx context.Context, id string) (*storage.InitBundleMeta, bool, error)
	Upsert(ctx context.Context, obj *storage.InitBundleMeta) error
	Delete(ctx context.Context, id string) error
	Walk(ctx context.Context, fn func(obj *storage.InitBundleMeta) error) error
}

type storeImpl struct {
	store             UnderlyingStore
	uniqueIDMutex     sync.Mutex
	uniqueUpdateMutex sync.Mutex
}

// NewStore returns a wrapper store for cluster init bundles
func NewStore(store UnderlyingStore) Store {
	return &storeImpl{store: store}
}

func (w *storeImpl) GetAll(ctx context.Context) ([]*storage.InitBundleMeta, error) {
	var result []*storage.InitBundleMeta
	if err := w.store.Walk(ctx, func(obj *storage.InitBundleMeta) error {
		if obj.GetVersion() != storage.InitBundleMeta_INIT_BUNDLE {
			return nil
		}
		if obj.GetIsRevoked() {
			return nil
		}
		result = append(result, obj)
		return nil
	}); err != nil {
		return nil, err
	}
	return result, nil
}

func (w *storeImpl) GetAllCRS(ctx context.Context) ([]*storage.InitBundleMeta, error) {
	var result []*storage.InitBundleMeta
	if err := w.store.Walk(ctx, func(obj *storage.InitBundleMeta) error {
		if obj.GetVersion() != storage.InitBundleMeta_CRS {
			return nil
		}
		if obj.GetIsRevoked() {
			return nil
		}
		result = append(result, obj)
		return nil
	}); err != nil {
		return nil, err
	}
	return result, nil
}

func (w *storeImpl) Get(ctx context.Context, id string) (*storage.InitBundleMeta, error) {
	obj, exists, err := w.store.Get(ctx, id)
	if err != nil {
		return nil, err
	} else if !exists {
		return nil, ErrInitBundleNotFound
	}
	return obj, nil
}

func (w *storeImpl) Add(ctx context.Context, meta *storage.InitBundleMeta) error {
	w.uniqueIDMutex.Lock()
	defer w.uniqueIDMutex.Unlock()

	if err := w.checkDuplicateName(ctx, meta); err != nil {
		return err
	}

	if exists, err := w.store.Exists(ctx, meta.GetId()); err != nil {
		return err
	} else if exists {
		return ErrInitBundleIDCollision
	}

	return w.store.Upsert(ctx, meta)
}

func (w *storeImpl) Delete(ctx context.Context, id string) error {
	return w.store.Delete(ctx, id)
}

func (w *storeImpl) checkDuplicateName(ctx context.Context, meta *storage.InitBundleMeta) error {
	metas, err := w.GetAll(ctx)
	if err != nil {
		return err
	}
	crsMetas, err := w.GetAllCRS(ctx)
	if err != nil {
		return err
	}
	metas = append(metas, crsMetas...)
	for _, m := range metas {
		if m.GetName() == meta.GetName() && !m.GetIsRevoked() {
			return ErrInitBundleDuplicateName
		}
	}
	return nil
}

func (w *storeImpl) Upsert(ctx context.Context, crs *storage.InitBundleMeta) error {
	w.uniqueUpdateMutex.Lock()
	defer w.uniqueUpdateMutex.Unlock()

	return w.store.Upsert(ctx, crs)
}

func (w *storeImpl) Revoke(ctx context.Context, id string) error {
	w.uniqueUpdateMutex.Lock()
	defer w.uniqueUpdateMutex.Unlock()

	meta, err := w.Get(ctx, id)
	if err != nil {
		if errors.Is(err, ErrInitBundleNotFound) {
			return errors.Errorf("init bundle %q does not exist", meta.GetId())
		}
		return errors.Wrapf(err, "reading init bundle %q", id)
	}

	meta.IsRevoked = true
	if err := w.store.Upsert(ctx, meta); err != nil {
		return err
	}
	return nil
}

// InitiateClusterRegistration checks if another registration using the CRS with the provided CRS ID is possible.
// If the provided id belongs to an init bundle, then registration is always allowed, without any bookkeeping.
func (w *storeImpl) InitiateClusterRegistration(ctx context.Context, initArtifactId, clusterName string) error {
	w.uniqueUpdateMutex.Lock()
	defer w.uniqueUpdateMutex.Unlock()

	initArtifactMeta, err := w.Get(ctx, initArtifactId)
	if err != nil {
		return errors.Wrapf(err, "retrieving init artifact meta data for ID %q", initArtifactId)
	}

	log.Infof("Attempting registration for cluster %s using %s %s.", clusterName, initArtifactMeta.GetVersion().String(), initArtifactMeta.GetId())
	if initArtifactMeta.GetIsRevoked() {
		log.Warnf("Init artifact %s is revoked, registration of cluster %s not allowed.", initArtifactId, clusterName)
		return errors.Errorf("Init artifact %s is revoked", initArtifactMeta.GetId())
	}

	if initArtifactMeta.GetVersion() == storage.InitBundleMeta_INIT_BUNDLE {
		return nil
	}

	crsMeta := initArtifactMeta
	maxRegistrations := crsMeta.GetMaxRegistrations()

	if maxRegistrations == 0 {
		// We don't do any bookkeeping in this case to prevent the clusterinit storage holding the CRS IDs from growing unbounded.
		log.Infof("Allowing registration of cluster %s using CRS %s without registration limit.", clusterName, crsMeta.GetId())
		return nil
	}

	// Bookkeeping for registration-limited CRS.
	registrationsInitiatedSet := set.NewStringSet(crsMeta.GetRegistrationsInitiated()...)
	registrationsCompletedSet := set.NewStringSet(crsMeta.GetRegistrationsCompleted()...)
	numRegistrationsTotal := uint64(len(registrationsInitiatedSet) + len(registrationsCompletedSet))
	if numRegistrationsTotal >= maxRegistrations {
		log.Warnf("maximum number of cluster registrations (%d/%d) with the provided cluster registration secret %s/%q reached.",
			numRegistrationsTotal, maxRegistrations,
			crsMeta.GetId(), crsMeta.GetName())
		return errors.New("maximum number of allowed cluster registrations reached")
	}
	if registrationsCompletedSet.Contains(clusterName) {
		return errors.Errorf("cluster %s already registered with cluster registration secret %s/%q", clusterName, crsMeta.GetId(), crsMeta.GetName())
	}
	if registrationsInitiatedSet.Contains(clusterName) {
		log.Warnf("Attempting to initiate registration of cluster %s, even though it is already associated with CRS %s.", clusterName, crsMeta.GetId())
	} else {
		_ = registrationsInitiatedSet.Add(clusterName)
		crsMeta.RegistrationsInitiated = registrationsInitiatedSet.AsSlice()
		if err := w.store.Upsert(ctx, crsMeta); err != nil {
			return errors.Wrapf(err, "updating meta data for cluster registration secret %s/%q", crsMeta.GetId(), crsMeta.GetName())
		}

		log.Infof("Added cluster %s to list of initiated registrations for CRS %s (%s).", clusterName, crsMeta.GetName(), crsMeta.GetId())
	}

	return nil
}

func (w *storeImpl) MarkClusterRegistrationComplete(ctx context.Context, initArtifactId, clusterName string) error {
	w.uniqueUpdateMutex.Lock()
	defer w.uniqueUpdateMutex.Unlock()

	initArtifactMeta, err := w.Get(ctx, initArtifactId)
	if err != nil {
		return errors.Wrapf(err, "retrieving init artifact meta data for ID %q", initArtifactId)
	}

	log.Infof("Completing registration of cluster %s using %s %s.", clusterName, initArtifactMeta.GetVersion().String(), initArtifactMeta.GetId())

	if initArtifactMeta.GetVersion() == storage.InitBundleMeta_INIT_BUNDLE {
		return nil
	}

	crsMeta := initArtifactMeta
	maxRegistrations := crsMeta.GetMaxRegistrations()

	if maxRegistrations == 0 {
		log.Infof("Completing the registration of cluster %s using CRS %s without allowed registration limit.", clusterName, crsMeta.GetId())
		return nil
	}

	// Bookkeeping for registration-limited CRS.

	log.Infof("Marking registration of cluster %s using CRS %s as complete.", clusterName, crsMeta.GetId())
	registrationsInitiatedSet := set.NewStringSet(crsMeta.GetRegistrationsInitiated()...)
	registrationsCompletedSet := set.NewStringSet(crsMeta.GetRegistrationsCompleted()...)

	if registrationsCompletedSet.Contains(clusterName) {
		// Already done?
		log.Infof("Registration of cluster %s using CRS %s already completed.", clusterName, initArtifactId)
		return nil
	}

	if !registrationsInitiatedSet.Contains(clusterName) {
		return errors.Errorf("registration for cluster %s using cluster registration secret %s/%q not initiated", clusterName, crsMeta.GetId(), crsMeta.GetName())
	}

	_ = registrationsInitiatedSet.Remove(clusterName)
	_ = registrationsCompletedSet.Add(clusterName)
	updatedRegistrationsInitiated := uint64(len(registrationsInitiatedSet))
	updatedRegistrationsCompleted := uint64(len(registrationsCompletedSet))
	// Revoke CRS, if the limit for completed registrations is reached and if no registrations are currently in flight.
	if updatedRegistrationsCompleted >= maxRegistrations && updatedRegistrationsInitiated == 0 {
		crsMeta.IsRevoked = true
	}

	crsMeta.RegistrationsInitiated = registrationsInitiatedSet.AsSlice()
	crsMeta.RegistrationsCompleted = registrationsCompletedSet.AsSlice()
	if err := w.store.Upsert(ctx, crsMeta); err != nil {
		return errors.Wrapf(err, "updating meta data for cluster registration secret %q", crsMeta.GetId())
	}

	if crsMeta.GetIsRevoked() {
		log.Infof("Marked CRS %s as revoked.", crsMeta.GetId())
	}

	return nil
}
