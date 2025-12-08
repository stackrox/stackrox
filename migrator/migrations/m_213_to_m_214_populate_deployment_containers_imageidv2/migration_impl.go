package m213tom214

import (
	"github.com/hashicorp/go-multierror"
	"github.com/jackc/pgx/v5"
	"github.com/stackrox/rox/migrator/migrations/m_213_to_m_214_populate_deployment_containers_imageidv2/schema"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/uuid"
)

var (
	log = logging.LoggerForModule()
)

var batchSize = 5000

func migrate(database *types.Databases) error {
	// Use databases.DBCtx to take advantage of the transaction wrapping present in the migration initiator
	pgutils.CreateTableFromModel(database.DBCtx, database.GormDB, schema.CreateTableDeploymentsStmt)
	log.Infof("Batch size is %d", batchSize)

	db := database.PostgresDB

	conn, err := db.Acquire(database.DBCtx)
	defer conn.Release()
	if err != nil {
		return err
	}
	updatedRows := 0
	for {
		batch := pgx.Batch{}
		// This will continue looping through the containers until there are no more containers that need to have their
		// image_idv2 field populated, in batches up to batchSize
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
			imageIdV2 := uuid.NewV5FromNonUUIDs(container.ImageNameFullName, container.ImageID).String()
			batch.Queue(updateStmt, imageIdV2, container.ImageNameFullName, container.ImageID)
		}
		batchResults := conn.SendBatch(database.DBCtx, &batch)
		var result *multierror.Error
		for i := 0; i < batch.Len(); i++ {
			_, err = batchResults.Exec()
			result = multierror.Append(result, err)
			if err == nil {
				updatedRows += 1
			}
		}
		if err = batchResults.Close(); err != nil {
			return err
		}
		if err = result.ErrorOrNil(); err != nil {
			return err
		}
		if len(containers) != batchSize {
			log.Infof("Populated the image_idv2 field in deployment containers. %d rows updated.", updatedRows)
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
