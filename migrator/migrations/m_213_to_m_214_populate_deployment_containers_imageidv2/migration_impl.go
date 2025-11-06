package m213tom214

import (
	"github.com/cloudflare/cfssl/log"
	"github.com/hashicorp/go-multierror"
	"github.com/jackc/pgx/v5"
	"github.com/stackrox/rox/migrator/migrations/m_213_to_m_214_populate_deployment_containers_imageidv2/schema"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/uuid"
)

var batchSize = 5000

func migrate(database *types.Databases) error {
	// Use databases.DBCtx to take advantage of the transaction wrapping present in the migration initiator
	pgutils.CreateTableFromModel(database.DBCtx, database.GormDB, schema.CreateTableDeploymentsStmt)

	db := database.PostgresDB

	conn, err := db.Acquire(database.DBCtx)
	defer conn.Release()
	if err != nil {
		return err
	}
	for {
		batch := pgx.Batch{}
		getStmt := `SELECT image_name_fullname, image_id FROM deployments_containers WHERE image_id is not null AND image_id != '' AND image_name_fullname is not null AND image_name_fullname != '' AND (image_idv2 is null OR image_idv2 = '') LIMIT $1`
		rows, err := db.Query(database.DBCtx, getStmt, batchSize)
		if err != nil {
			return err
		}
		defer rows.Close()

		containers, err := readRows(rows)
		if err != nil {
			return err
		}
		for _, container := range containers {
			updateStmt := `UPDATE deployments_containers SET image_idv2 = $1 WHERE image_name_fullname = $2 AND image_id = $3`
			batch.Queue(updateStmt, uuid.NewV5FromNonUUIDs(container.ImageNameFullName, container.ImageID).String(), container.ImageNameFullName, container.ImageID)
		}
		batchResults := conn.SendBatch(database.DBCtx, &batch)
		var result *multierror.Error
		for i := 0; i < batch.Len(); i++ {
			_, err = batchResults.Exec()
			result = multierror.Append(result, err)
		}
		if err = batchResults.Close(); err != nil {
			return err
		}
		if err = result.ErrorOrNil(); err != nil {
			return err
		}
		if len(containers) != batchSize {
			return nil
		}
	}
}

func readRows(rows *postgres.Rows) ([]*schema.DeploymentsContainers, error) {
	var containers []*schema.DeploymentsContainers

	for rows.Next() {
		var imageName string
		var imageId string

		if err := rows.Scan(&imageName, &imageId); err != nil {
			log.Errorf("Error scanning row: %v", err)
		}

		container := &schema.DeploymentsContainers{
			ImageID:           imageId,
			ImageNameFullName: imageName,
		}
		containers = append(containers, container)
	}

	log.Debugf("Read returned %d containers", len(containers))
	return containers, rows.Err()
}
