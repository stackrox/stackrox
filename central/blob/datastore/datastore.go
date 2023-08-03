package datastore

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/stackrox/rox/central/blob/datastore/search"
	"github.com/stackrox/rox/central/blob/datastore/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	bufferedBlobDataLimitInBytes = 5 * 1024 * 1024
	adminSAC                     = sac.ForResource(resources.Administration)
)

// Datastore provides access to the blob store
type Datastore interface {
	GetIDs(ctx context.Context) ([]string, error)
	Upsert(ctx context.Context, obj *storage.Blob, reader io.Reader) error
	Get(ctx context.Context, name string, writer io.Writer) (*storage.Blob, bool, error)
	Delete(ctx context.Context, name string) error
	GetMetadata(ctx context.Context, name string) (*storage.Blob, bool, error)
	GetBlobWithDataInBuffer(ctx context.Context, name string) (*bytes.Buffer, *storage.Blob, bool, error)

	Search(ctx context.Context, query *v1.Query) ([]pkgSearch.Result, error)
	SearchIDs(ctx context.Context, q *v1.Query) ([]string, error)
	SearchMetadata(ctx context.Context, q *v1.Query) ([]*storage.Blob, error)
}

// NewDatastore creates a new Blob datastore
func NewDatastore(store store.Store, searcher search.Searcher) Datastore {
	return &datastoreImpl{
		store:    store,
		searcher: searcher,
	}
}

type datastoreImpl struct {
	store    store.Store
	searcher search.Searcher
}

// Upsert adds a new blob to the database
func (d *datastoreImpl) Upsert(ctx context.Context, obj *storage.Blob, reader io.Reader) error {
	return d.store.Upsert(ctx, obj, reader)
}

// Get retrieves a blob from the database
func (d *datastoreImpl) Get(ctx context.Context, name string, writer io.Writer) (*storage.Blob, bool, error) {
	return d.store.Get(ctx, name, writer)
}

// Delete removes a blob store from database
func (d *datastoreImpl) Delete(ctx context.Context, name string) error {
	return d.store.Delete(ctx, name)
}

// GetIDs return all blob ids
func (d *datastoreImpl) GetIDs(ctx context.Context) ([]string, error) {
	return d.store.GetIDs(ctx)
}

// GetMetadata returns blob metadata only
func (d *datastoreImpl) GetMetadata(ctx context.Context, name string) (*storage.Blob, bool, error) {
	return d.store.GetMetadata(ctx, name)
}

// GetBlobWithDataInBuffer returns the blob with data in a buffer with a size limit
func (d *datastoreImpl) GetBlobWithDataInBuffer(ctx context.Context, name string) (*bytes.Buffer, *storage.Blob, bool, error) {
	buf := bytes.NewBuffer(nil)

	blob, exists, err := d.store.Get(ctx, name, buf)
	if blob.GetLength() > int64(bufferedBlobDataLimitInBytes) {
		utils.Should(fmt.Errorf("blob %s has %d in length which is beyond buffer limit %d", name, blob.Size(), bufferedBlobDataLimitInBytes))
	}
	return buf, blob, exists, err
}

// Search blobs
func (d *datastoreImpl) Search(ctx context.Context, query *v1.Query) ([]pkgSearch.Result, error) {
	scopeChecker := adminSAC.ScopeChecker(ctx, storage.Access_READ_ACCESS)
	if !scopeChecker.IsAllowed() {
		return nil, errox.NotAuthorized
	}
	return d.searcher.Search(ctx, query)
}

// SearchIDs searches and return blob IDs
func (d *datastoreImpl) SearchIDs(ctx context.Context, q *v1.Query) ([]string, error) {
	scopeChecker := adminSAC.ScopeChecker(ctx, storage.Access_READ_ACCESS)
	if !scopeChecker.IsAllowed() {
		return nil, errox.NotAuthorized
	}
	return d.searcher.SearchIDs(ctx, q)
}

// SearchMetadata searches and return blob metadata only
func (d *datastoreImpl) SearchMetadata(ctx context.Context, q *v1.Query) ([]*storage.Blob, error) {
	scopeChecker := adminSAC.ScopeChecker(ctx, storage.Access_READ_ACCESS)
	if !scopeChecker.IsAllowed() {
		return nil, errox.NotAuthorized
	}
	return d.searcher.SearchMetadata(ctx, q)
}
