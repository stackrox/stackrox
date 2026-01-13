package datastore

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/pkg/errors"
	baseImageStore "github.com/stackrox/rox/central/baseimage/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/uuid"
)

type datastoreImpl struct {
	storage baseImageStore.Store
	db      postgres.DB
}

const (
	baseImagesTable = "base_images"
	// The 'firstlayerdigest' column is indexed.
	listByFirstLayerQuery = "SELECT id FROM " + baseImagesTable + " WHERE firstlayerdigest = $1"
	// Query to find the ID by manifest digest
	getByManifestDigestQuery = "SELECT id FROM " + baseImagesTable + " WHERE manifestdigest = $1"
	upsertChunkSize          = 100
)

// New creates a new DataStore instance backed by PostgreSQL.
func New(store baseImageStore.Store, db postgres.DB) DataStore {
	return &datastoreImpl{
		storage: store,
		db:      db,
	}
}

func (ds *datastoreImpl) UpsertImage(ctx context.Context, image *storage.BaseImage, digests []string) error {
	if _, err := layers(image, digests); err != nil {
		return fmt.Errorf("upsert image %s: %w", image.GetId(), err)
	}
	return ds.storage.Upsert(ctx, image)
}

func (ds *datastoreImpl) UpsertImages(
	ctx context.Context,
	imagesWithLayer map[*storage.BaseImage][]string,
) error {
	batch := make([]*storage.BaseImage, 0, len(imagesWithLayer))
	for img, digests := range imagesWithLayer {
		if _, err := layers(img, digests); err != nil {
			return fmt.Errorf("prepare layers for image %s: %w", img.GetId(), err)
		}

		batch = append(batch, img)
	}

	if len(batch) <= upsertChunkSize {
		return ds.storage.UpsertMany(ctx, batch)
	}

	// Process in chunks. This reduces pressure on storage and avoids parameter/time limits.
	for start := 0; start < len(batch); start += upsertChunkSize {
		end := start + upsertChunkSize
		if end > len(batch) {
			end = len(batch)
		}

		if err := ds.storage.UpsertMany(ctx, batch[start:end]); err != nil {
			return fmt.Errorf("upsert images chunk [%d:%d]: %w", start, end, err)
		}
	}
	return nil
}

func layers(image *storage.BaseImage, digests []string) ([]*storage.BaseImageLayer, error) {
	if len(digests) == 0 {
		return nil, fmt.Errorf("layers: empty digests for image %s", image.GetId())
	}

	layers := make([]*storage.BaseImageLayer, 0, len(digests))

	for i, digest := range digests {
		layers = append(layers, &storage.BaseImageLayer{
			Id:          uuid.NewV4().String(),
			BaseImageId: image.GetId(),
			Index:       int32(i),
			LayerDigest: digest,
		})
	}

	image.FirstLayerDigest = digests[0]
	image.Layers = layers
	return layers, nil
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

	// Use the search framework to load full base image objects
	baseImages, _, err := ds.storage.GetMany(ctx, ids)
	if err != nil {
		return nil, err
	}

	return baseImages, nil
}

func (ds *datastoreImpl) GetBaseImage(ctx context.Context, manifestDigest string) (*storage.BaseImage, bool, error) {
	row := ds.db.QueryRow(ctx, getByManifestDigestQuery, manifestDigest)

	var id string
	if err := row.Scan(&id); err != nil {
		// Check if the error is "no rows found"
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return ds.storage.Get(ctx, id)
}
