package store

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	// ErrInitBundleNotFound signals that a requested init bundle could not be located in the store.
	ErrInitBundleNotFound = errors.New("init bundle not found")

	// ErrInitBundleIDCollision signals that an init bundle could not be added to the store due to an ID collision.
	ErrInitBundleIDCollision = errors.New("init bundle ID collision")

	// ErrInitBundleDuplicateName signals that an init bundle could not be added because the name already exists on a non-revoked init bundle
	ErrInitBundleDuplicateName = errors.New("init bundle already exists")
)

// Store interface for managing persisted cluster init bundles.
type Store interface {
	GetAll(ctx context.Context) ([]*storage.InitBundleMeta, error)
	Get(ctx context.Context, id string) (*storage.InitBundleMeta, error)
	Add(ctx context.Context, bundleMeta *storage.InitBundleMeta) error
	Revoke(ctx context.Context, id string) error
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
