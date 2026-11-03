package m227tom228

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations/loghelper"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/sac"
)

var (
	log = loghelper.LogWrapper{}
)

func migrate(database *types.Databases) error {
	ctx := sac.WithAllAccess(context.Background())

	// manifest_digest is stored in the serialized BaseImageTag blob, not as its own column.
	// Clearing it makes the next base-image scan treat the tag as unseen and refetch history.
	rows, err := database.PostgresDB.Query(ctx, `SELECT id, serialized FROM base_image_tags`)
	if err != nil {
		return errors.Wrap(err, "querying base image tags")
	}
	defer rows.Close()

	type tagUpdate struct {
		id         string
		serialized []byte
	}
	var updates []tagUpdate
	for rows.Next() {
		var id string
		var serialized []byte
		if err := rows.Scan(&id, &serialized); err != nil {
			return errors.Wrap(err, "scanning base image tag")
		}
		tag := &storage.BaseImageTag{}
		if len(serialized) > 0 {
			if err := tag.UnmarshalVT(serialized); err != nil {
				return errors.Wrapf(err, "unmarshalling base image tag %s", id)
			}
		}
		if tag.GetManifestDigest() == "" {
			continue
		}
		tag.ManifestDigest = ""
		updated, err := tag.MarshalVT()
		if err != nil {
			return errors.Wrapf(err, "marshalling base image tag %s", id)
		}
		updates = append(updates, tagUpdate{id: id, serialized: updated})
	}
	if err := rows.Err(); err != nil {
		return errors.Wrap(err, "iterating base image tags")
	}
	// Release the query before updates. Close returns no error in this driver.
	rows.Close()

	for _, update := range updates {
		if _, err := database.PostgresDB.Exec(ctx, `UPDATE base_image_tags SET serialized = $1 WHERE id = $2::uuid`, update.serialized, update.id); err != nil {
			return errors.Wrapf(err, "clearing manifest digest for base image tag %s", update.id)
		}
	}
	log.WriteToStderrf("Cleared manifest digest for %d base image tags to force history refetch", len(updates))
	return nil
}
