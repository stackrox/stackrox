package datastore

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/pkg/errors"
	baseImageStore "github.com/stackrox/rox/central/baseimage/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/uuid"
)

type datastoreImpl struct {
	storage baseImageStore.Store
	db      postgres.DB
}

var (
	log       = logging.LoggerForModule()
	imagesSAC = sac.ForResource(resources.ImageAdministration)
)

const (
	baseImagesTable = "base_images"
	// The 'firstlayerdigest' column is indexed.
	listByFirstLayerQuery = "SELECT id FROM " + baseImagesTable + " WHERE firstlayerdigest = $1"

	// Query to find the ID by manifest digest
	getByManifestDigestQuery = "SELECT id FROM " + baseImagesTable + " WHERE manifestdigest = $1"
)

// New creates a new DataStore instance backed by PostgreSQL.
func New(store baseImageStore.Store, db postgres.DB) DataStore {
	return &datastoreImpl{
		storage: store,
		db:      db,
	}
}

func (ds *datastoreImpl) UpsertImage(ctx context.Context, image *storage.BaseImage, digests []string) error {
	ok, err := imagesSAC.WriteAllowed(ctx)
	if err != nil {
		return err
	}
	if !ok {
		return sac.ErrResourceAccessDenied
	}

	layers(image, digests)
	return ds.storage.Upsert(ctx, image)
}

func (ds *datastoreImpl) UpsertImages(
	ctx context.Context, imagesWithLayer map[*storage.BaseImage][]string,
) error {
	ok, err := imagesSAC.WriteAllowed(ctx)
	if err != nil {
		return err
	}
	if !ok {
		return sac.ErrResourceAccessDenied
	}

	batch := make([]*storage.BaseImage, 0, len(imagesWithLayer))
	for img, digests := range imagesWithLayer {
		layers(img, digests)
		batch = append(batch, img)
	}

	return ds.storage.UpsertMany(ctx, batch)
}

func layers(image *storage.BaseImage, digests []string) []*storage.BaseImageLayer {
	layers := make([]*storage.BaseImageLayer, 0, len(digests))

	for i, digest := range digests {
		layers = append(layers, &storage.BaseImageLayer{
			Id:          uuid.NewV4().String(),
			BaseImageId: image.GetId(),
			Index:       int32(i),
			LayerDigest: digest,
		})
	}

	if len(digests) > 0 && image.GetFirstLayerDigest() != digests[0] {
		log.Warnf(
			"FirstLayerDigest mismatch for image %s: claims %s but first layer is %s.",
			image.GetId(), image.GetFirstLayerDigest(), digests[0],
		)
		image.FirstLayerDigest = digests[0]
	}

	image.Layers = layers
	return layers
}

func (ds *datastoreImpl) ListCandidateBaseImages(ctx context.Context, firstLayer string) ([]*storage.BaseImage, error) {
	if ok, err := imagesSAC.ReadAllowed(ctx); err != nil {
		return nil, err
	} else if !ok {
		return nil, sac.ErrResourceAccessDenied
	}
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

func (ds *datastoreImpl) GetBaseImage(ctx context.Context, manifestDigest string) (*storage.BaseImage, bool, error) {
	if ok, err := imagesSAC.ReadAllowed(ctx); err != nil {
		return nil, false, err
	} else if !ok {
		return nil, false, sac.ErrResourceAccessDenied
	}
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
