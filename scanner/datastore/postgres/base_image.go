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

// GetBaseImageCandidates retrieves a map of base images that have a matching final layer digest.
// The map key is formatted as "registry/repository:tag" and the value is a slice of all
// layer hashes, ordered by their level.
func (i indexerBaseImageStore) GetBaseImageCandidates(ctx context.Context, digest string) (map[string][]string, error) {
	// This SQL query joins the base_images and base_image_layer tables.
	// It filters for base images where the final_layer matches the provided digest.
	// The results are grouped by base image, and the array_agg function collects
	// all layer hashes for each base image, ordered by level.
	sql := `
		SELECT
			bi.registry,
			bi.repository,
			bi.tag,
			array_agg(bil.layer_hash ORDER BY bil.level ASC) AS layer_hashes_ordered_by_level
		FROM
			base_images AS bi
		INNER JOIN
			base_image_layer AS bil ON bi.id = bil.iid
		WHERE
			bi.final_layer = $1
		GROUP BY
			bi.id, bi.registry, bi.repository, bi.tag;
	`

	rows, err := i.pool.Query(ctx, sql, digest)
	if err != nil {
		return nil, fmt.Errorf("failed to query for base image candidates: %w", err)
	}
	defer rows.Close()

	candidates := make(map[string][]string)

	for rows.Next() {
		var registry, repository string
		var tag *string // Use a pointer for the nullable 'tag' column
		var layerHashes []string

		if err := rows.Scan(&registry, &repository, &tag, &layerHashes); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Build the map key as "registry/repository:tag".
		// Handle the case where the tag is NULL.
		var keyBuilder strings.Builder
		keyBuilder.WriteString(registry)
		keyBuilder.WriteString("/")
		keyBuilder.WriteString(repository)
		if tag != nil {
			keyBuilder.WriteString(":")
			keyBuilder.WriteString(*tag)
		}

		// Add the ordered layer hashes to the map.
		candidates[keyBuilder.String()] = layerHashes
	}

	// Check for any errors that occurred during rows.Next() or rows.Close().
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during row iteration: %w", err)
	}

	return candidates, nil
}
