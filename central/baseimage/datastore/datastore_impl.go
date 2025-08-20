package datastore

import (
	"context"

	"github.com/pkg/errors"
	biStr "github.com/stackrox/rox/central/baseimage/store/postgres"
	bilStr "github.com/stackrox/rox/central/baseimagelayer/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres"
)

var (
	log = logging.LoggerForModule()
)

const (
	baseImageLayersTable = "base_image_layers"
	baseImagesTable      = "base_images"
)

type datastoreImpl struct {
	db                  postgres.DB
	baseImageStore      biStr.Store
	baseImageLayerStore bilStr.Store
}

func (d datastoreImpl) GetCandidateLayers(ctx context.Context, layerSHA string) ([]*storage.BaseImageLayer, bool, error) {
	query := "SELECT T1.serialized FROM " + baseImageLayersTable + " AS T1 JOIN " + baseImagesTable + " AS T2 ON T1.layerdigest = T2.firstlayerdigest WHERE T1.layerdigest = $1;"

	rows, err := d.db.Query(ctx, query, layerSHA)
	if err != nil {
		return nil, false, errors.Wrap(err, "could not execute query to get candidate layers")
	}
	defer rows.Close()

	var results []*storage.BaseImageLayer

	for rows.Next() {
		var serialized []byte
		if err := rows.Scan(&serialized); err != nil {
			return nil, false, errors.Wrap(err, "could not convert row into serialized bytes")
		}

		// Unmarshal
		var baseImageLayer storage.BaseImageLayer
		if err := baseImageLayer.UnmarshalVT(serialized); err != nil {
			return nil, false, errors.Wrap(err, "could not unmarshal serialized data")
		}
		results = append(results, &baseImageLayer)
	}

	if err := rows.Err(); err != nil {
		return nil, false, errors.Wrap(err, "error after iterating through rows")
	}

	// This is a common debugging pattern.
	log.Infof("Found %d candidate base images for digest %s\n", len(results), layerSHA)

	return results, true, nil
}
