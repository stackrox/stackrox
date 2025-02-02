package store

import (
	"context"
	"math"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	// ErrInitBundleNotFound signals that a requested init bundle could not be located in the store.
	ErrInitBundleNotFound = errors.New("init bundle or CRS not found")

	// ErrInitBundleIDCollision signals that an init bundle could not be added to the store due to an ID collision.
	ErrInitBundleIDCollision = errors.New("init bundle or CRS ID collision")

	// ErrInitBundleDuplicateName signals that an init bundle or CRS could not be added because the name already exists on a non-revoked init bundle or CRS.
	ErrInitBundleDuplicateName = errors.New("init bundle or CRS already exists")
)

// Store interface for managing persisted cluster init bundles.
//
//go:generate mockgen-wrapper
type Store interface {
	GetAll(ctx context.Context) ([]*storage.InitBundleMeta, error)
	GetAllCRS(ctx context.Context) ([]*storage.InitBundleMeta, error)
	Get(ctx context.Context, id string) (*storage.InitBundleMeta, error)
	Add(ctx context.Context, bundleMeta *storage.InitBundleMeta) error
	Revoke(ctx context.Context, id string) error
	RegistrationPossible(ctx context.Context, id string) error
	RecordInitiatedRegistration(ctx context.Context, id string) error
	RecordCompletedRegistration(ctx context.Context, id string) error
	RevokeIfMaxRegistrationsReached(ctx context.Context, id string) error
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

// RegistrationPossible checks if another registration using the CRS with the provided CRS ID is possible,
// based on the current value of the `RegistrationsInitiated` counter.
func (w *storeImpl) RegistrationPossible(ctx context.Context, id string) error {
	w.uniqueUpdateMutex.Lock()
	defer w.uniqueUpdateMutex.Unlock()

	crs, err := w.Get(ctx, id)
	if err != nil {
		return errors.Wrapf(err, "retrieving cluster registration secret meta data for %q", id)
	}

	switch crs.Version {
	case storage.InitBundleMeta_INIT_BUNDLE:
		// We don't support registration limits for init bundles.
		return nil
	case storage.InitBundleMeta_CRS:
		if crs.MaxRegistrations == 0 || crs.RegistrationsInitiated < crs.MaxRegistrations {
			return nil
		}
	}

	return errors.Errorf("maximum number of clusters registrations (%d/%d) with the provided secret %q/%q reached",
		crs.RegistrationsInitiated, crs.MaxRegistrations, crs.Name, crs.Id)
}

func (w *storeImpl) recordRegistration(
	ctx context.Context, id string,
	getNumRegistrations func(*storage.InitBundleMeta) uint32,
	setNumRegistrations func(*storage.InitBundleMeta, uint32),
) error {
	w.uniqueUpdateMutex.Lock()
	defer w.uniqueUpdateMutex.Unlock()

	crsMeta, err := w.Get(ctx, id)
	if err != nil {
		return errors.Wrapf(err, "retrieving cluster registration secret meta data for %q", id)
	}
	if crsMeta.GetVersion() != storage.InitBundleMeta_CRS {
		// The caller of this function doesn't know if it holds the ID of an init bundle or the ID of a CRS.
		// Hence, we will silently skip init bundles here instead of caring about how to implement
		// registration counting logic for init bundles.
		return nil
	}

	maxRegistrations := crsMeta.GetMaxRegistrations()
	numRegistrations := getNumRegistrations(crsMeta)

	// maxRegistrations == 0 means no restrictions on the number of cluster registrations.
	if maxRegistrations == 0 {
		// We stop recording at MaxUint32 to prevent overflows.
		if numRegistrations < math.MaxUint32 {
			setNumRegistrations(crsMeta, numRegistrations+1)
		}
	} else {
		if numRegistrations >= maxRegistrations { // ">" should never happen.
			return errors.New("maximum number of allowed cluster registrations reached")
		}
		setNumRegistrations(crsMeta, numRegistrations+1)
	}

	err = w.store.Upsert(ctx, crsMeta)
	if err != nil {
		return errors.Wrapf(err, "updating meta data for cluster registration secret %q", id)
	}
	return nil
}

func (w *storeImpl) RecordInitiatedRegistration(ctx context.Context, id string) error {
	getInitiatedRegistrations := func(crsMeta *storage.InitBundleMeta) uint32 {
		return crsMeta.GetRegistrationsInitiated()
	}
	setInitiatedRegistrations := func(crsMeta *storage.InitBundleMeta, n uint32) {
		crsMeta.RegistrationsInitiated = n
	}

	err := w.recordRegistration(ctx, id, getInitiatedRegistrations, setInitiatedRegistrations)
	if err != nil {
		return errors.Wrap(err, "recording initiated registrations")
	}

	return nil
}

func (w *storeImpl) RecordCompletedRegistration(ctx context.Context, id string) error {
	getCompletedRegistrations := func(crsMeta *storage.InitBundleMeta) uint32 {
		return crsMeta.GetRegistrationsCompleted()
	}
	setCompletedRegistrations := func(crsMeta *storage.InitBundleMeta, n uint32) {
		crsMeta.RegistrationsCompleted = n
	}

	err := w.recordRegistration(ctx, id, getCompletedRegistrations, setCompletedRegistrations)
	if err != nil {
		return errors.Wrap(err, "recording completed registrations")
	}

	return nil
}

func (w *storeImpl) RevokeIfMaxRegistrationsReached(ctx context.Context, id string) error {
	w.uniqueUpdateMutex.Lock()
	defer w.uniqueUpdateMutex.Unlock()

	crsMeta, err := w.Get(ctx, id)
	if err != nil {
		return errors.Wrapf(err, "retrieving cluster registration secret meta data for %q", id)
	}

	if crsMeta.GetVersion() != storage.InitBundleMeta_CRS {
		// We only support auto-revocation for CRS.
		return nil
	}
	if crsMeta.GetMaxRegistrations() == 0 {
		// No limit on the number of registrations.
		return nil
	}
	if crsMeta.GetIsRevoked() {
		// Already revoked.
		return nil
	}
	if crsMeta.GetRegistrationsCompleted() < crsMeta.GetMaxRegistrations() {
		// Registration limit not reached yet.
		return nil
	}

	// Limit is in fact reached.
	crsMeta.IsRevoked = true

	err = w.store.Upsert(ctx, crsMeta)
	if err != nil {
		return errors.Wrapf(err, "updating meta data for cluster registration secret %q", id)
	}
	return nil
}
