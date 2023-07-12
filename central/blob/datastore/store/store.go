package store

import (
	"context"
	"io"

	"github.com/jackc/pgx/v5"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/blob/datastore/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	pgPkg "github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
)

var (
	log          = logging.LoggerForModule()
	scopeChecker = sac.ForResource(resources.Administration)
)

// Store is the interface to interact with the storage for storage.Blob
//
//go:generate mockgen-wrapper
type Store interface {
	Upsert(ctx context.Context, obj *storage.Blob, reader io.Reader) error
	Get(ctx context.Context, name string, writer io.Writer) (*storage.Blob, bool, error)
	Delete(ctx context.Context, name string) error
	GetMetadataByQuery(ctx context.Context, query *v1.Query) ([]*storage.Blob, error)
	GetIDs(ctx context.Context) ([]string, error)
	GetMetadata(ctx context.Context, name string) (*storage.Blob, bool, error)
}

type storeImpl struct {
	db    pgPkg.DB
	store postgres.Store
}

// New creates a new Blob store
func New(db pgPkg.DB) Store {
	return &storeImpl{
		db:    db,
		store: postgres.New(db),
	}
}

func wrapRollback(ctx context.Context, tx *pgPkg.Tx, err error) error {
	rollbackErr := tx.Rollback(ctx)
	if rollbackErr != nil {
		return errors.Wrapf(rollbackErr, "rolling back due to err: %v", err)
	}
	return err
}

// Upsert adds a blob to the database
func (s *storeImpl) Upsert(ctx context.Context, obj *storage.Blob, reader io.Reader) error {
	if err := sac.VerifyAuthzOK(scopeChecker.WriteAllowed(ctx)); err != nil {
		return err
	}
	// Augment permission because we require read permission internally
	ctx = sac.WithGlobalAccessScopeChecker(ctx,
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Administration)))

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	ctx = pgPkg.ContextWithTx(ctx, tx)
	existingBlob, exists, err := s.store.Get(ctx, obj.GetName())
	if err != nil {
		return wrapRollback(ctx, tx, err)
	}

	los := tx.LargeObjects()
	var lo *pgx.LargeObject
	if exists {
		lo, err = los.Open(ctx, existingBlob.GetOid(), pgx.LargeObjectModeWrite)
		if err != nil {
			return wrapRollback(ctx, tx, errors.Wrapf(err, "opening blob with oid %d", existingBlob.GetOid()))
		}
		if err := lo.Truncate(0); err != nil {
			return wrapRollback(ctx, tx, errors.Wrapf(err, "truncating blob with oid %d", existingBlob.GetOid()))
		}
		obj.Oid = existingBlob.GetOid()
	} else {
		oid, err := los.Create(ctx, 0)
		if err != nil {
			return wrapRollback(ctx, tx, errors.Wrap(err, "error creating new blob"))
		}
		lo, err = los.Open(ctx, oid, pgx.LargeObjectModeWrite)
		if err != nil {
			return wrapRollback(ctx, tx, errors.Wrapf(err, "opening blob with oid %d", oid))
		}
		obj.Oid = oid
	}
	buf := make([]byte, 1024*1024)

	var totalRead int64
	for {
		nRead, err := reader.Read(buf)
		totalRead += int64(nRead)

		if nRead != 0 {
			if _, err := lo.Write(buf[:nRead]); err != nil {
				return wrapRollback(ctx, tx, errors.Wrap(err, "writing blob"))
			}
		}

		// nRead can be non-zero when err == io.EOF
		if err != nil {
			if err == io.EOF {
				break
			}
			return wrapRollback(ctx, tx, errors.Wrap(err, "reading buffer to write for blob"))
		}
	}
	if err := lo.Close(); err != nil {
		return wrapRollback(ctx, tx, errors.Wrap(err, "closing large object for blob"))
	}
	if totalRead != obj.GetLength() {
		return wrapRollback(ctx, tx, errors.Errorf("Blob metadata mismatch. Blob metadata shows %d in length, but data has length of %d", obj.GetLength(), totalRead))
	}

	if err := s.store.Upsert(ctx, obj); err != nil {
		return wrapRollback(ctx, tx, errors.Wrapf(err, "error upserting blob %q", obj.GetName()))
	}
	return tx.Commit(ctx)
}

// Get returns a blob from the database
func (s *storeImpl) Get(ctx context.Context, name string, writer io.Writer) (*storage.Blob, bool, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, false, err
	}
	ctx = pgPkg.ContextWithTx(ctx, tx)

	existingBlob, exists, err := s.store.Get(ctx, name)
	if err != nil || !exists {
		return nil, exists, wrapRollback(ctx, tx, err)
	}

	los := tx.LargeObjects()
	lo, err := los.Open(ctx, existingBlob.GetOid(), pgx.LargeObjectModeRead)
	if err != nil {
		err := errors.Wrapf(err, "error opening large object with oid %d", existingBlob.GetOid())
		return nil, false, wrapRollback(ctx, tx, err)
	}

	buf := make([]byte, 1024*1024)
	var totalRead int64
	for {
		nRead, err := lo.Read(buf)
		totalRead += int64(nRead)

		// nRead can be non-zero when err == io.EOF
		if nRead != 0 {
			if _, err := writer.Write(buf[:nRead]); err != nil {
				err := errors.Wrap(err, "error writing to output")
				return nil, false, wrapRollback(ctx, tx, err)
			}
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, false, wrapRollback(ctx, tx, errors.Wrap(err, "reading blob"))
		}
	}
	if err := lo.Close(); err != nil {
		err = errors.Wrap(err, "closing large object for blob")
		return nil, false, wrapRollback(ctx, tx, err)
	}

	if totalRead != existingBlob.GetLength() {
		return nil, false, wrapRollback(ctx, tx, errors.Errorf("Blob %s corrupted. Blob metadata shows %d in length, but data has length of %d", existingBlob.GetName(), existingBlob.GetLength(), totalRead))
	}

	return existingBlob, true, tx.Commit(ctx)
}

// Delete removes a blob from database if it exists
func (s *storeImpl) Delete(ctx context.Context, name string) error {
	if err := sac.VerifyAuthzOK(scopeChecker.WriteAllowed(ctx)); err != nil {
		return err
	}
	// Augment permission because we require read permission internally
	ctx = sac.WithGlobalAccessScopeChecker(ctx,
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Administration)))
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}

	ctx = pgPkg.ContextWithTx(ctx, tx)

	existingBlob, exists, err := s.store.Get(ctx, name)
	if err != nil || !exists {
		return wrapRollback(ctx, tx, err)
	}

	los := tx.LargeObjects()
	if err = los.Unlink(ctx, existingBlob.GetOid()); err != nil {
		return wrapRollback(ctx, tx, errors.Wrapf(err, "failed to remove large object with oid %d", existingBlob.GetOid()))
	}
	if err = s.store.Delete(ctx, name); err != nil {
		err = errors.Wrapf(err, "deleting large object %s", name)
		return wrapRollback(ctx, tx, err)
	}

	return tx.Commit(ctx)
}

// GetIDs all blob names
func (s *storeImpl) GetIDs(ctx context.Context) ([]string, error) {
	return s.store.GetIDs(ctx)
}

// GetMetadataByQuery get a list of Blobs by query.
func (s *storeImpl) GetMetadataByQuery(ctx context.Context, query *v1.Query) ([]*storage.Blob, error) {
	return s.store.GetByQuery(ctx, query)
}

// GetMetadata all blob names
func (s *storeImpl) GetMetadata(ctx context.Context, name string) (*storage.Blob, bool, error) {
	return s.store.Get(ctx, name)
}
