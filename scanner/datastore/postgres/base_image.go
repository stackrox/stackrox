package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/stackrox/rox/scanner/baseimage"
)

// AddBaseImage adds a new base image and its associated layers to the database within a transaction.
func (i indexerBaseImageStore) AddBaseImage(ctx context.Context, baseImage baseimage.AddBaseImageInput) error {
	// Start a transaction
	tx, err := i.pool.Begin(ctx)
	if err != nil {
		return err
	}
	// Defer a rollback in case of error. If the transaction commits, this will be a no-op.
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	// Insert the BaseImage into the base_images table
	insertBaseImageSQL := `
		INSERT INTO base_images (registry, repository, tag, digest, config_digest, active, final_layer)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id;
	`
	var baseImageID int64
	err = tx.QueryRow(
		ctx,
		insertBaseImageSQL,
		baseImage.BaseImage.Registry,
		baseImage.BaseImage.Repository,
		baseImage.BaseImage.Tag,
		baseImage.BaseImage.Digest,
		baseImage.BaseImage.ConfigDigest,
		baseImage.BaseImage.Active,
		baseImage.BaseImage.FinalLayer,
	).Scan(&baseImageID)
	if err != nil {
		return err
	}

	// Bulk Insert BaseImageLayers into the base_image_layer table
	if len(baseImage.Layers) > 0 {
		valueStrings := make([]string, 0, len(baseImage.Layers))
		valueArgs := make([]interface{}, 0, len(baseImage.Layers)*3) // 3 arguments per layer (iid, layer_hash, level)

		for idx, layer := range baseImage.Layers {
			valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d, $%d)",
				idx*3+1, idx*3+2, idx*3+3))
			valueArgs = append(valueArgs, baseImageID, layer.LayerHash, layer.Level)
		}

		insertLayersSQL := fmt.Sprintf(`
			INSERT INTO base_image_layer (iid, layer_hash, level)
			VALUES %s;
		`, strings.Join(valueStrings, ","))

		_, err := tx.Exec(ctx, insertLayersSQL, valueArgs...)
		if err != nil {
			return err // If bulk layer insertion fails, the transaction will be rolled back.
		}
	}

	// Commit if all succeeded
	return tx.Commit(ctx)
}
