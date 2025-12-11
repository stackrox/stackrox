package datastore

import (
	"context"

	baseImageStore "github.com/stackrox/rox/central/baseimage/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
)

type datastoreImpl struct {
	storage baseImageStore.Store
	db      postgres.DB
}

const (
	baseImagesTable = "base_images"
	// The 'firstlayerdigest' column is indexed.
	listByFirstLayerQuery = "SELECT id FROM " + baseImagesTable + " WHERE firstlayerdigest = $1"
)

// New creates a new DataStore instance backed by PostgreSQL.
func New(store baseImageStore.Store, db postgres.DB) DataStore {
	return &datastoreImpl{
		storage: store,
		db:      db,
	}
}
func (ds *datastoreImpl) ListCandidateBaseImages(ctx context.Context, firstLayer string) ([]*storage.BaseImage, error) {
	rows, err := ds.db.Query(ctx, listByFirstLayerQuery, firstLayer)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// If no IDs match, return empty to save a DB call.
	if len(ids) == 0 {
		return nil, nil
	}

	baseImages, _, err := ds.storage.GetMany(ctx, ids)
	if err != nil {
		return nil, err
	}

	return baseImages, nil
}
