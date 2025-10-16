package m213tom214

import (
	"github.com/cloudflare/cfssl/log"
	"github.com/stackrox/rox/migrator/migrations/m_213_to_m_214_populate_deployment_containers_imageidv2/schema"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/uuid"
)

// TODO(dont-merge): generate/write and import any store required for the migration (skip any unnecessary step):
//  - create a schema subdirectory
//  - create a schema/old subdirectory
//  - create a schema/new subdirectory
//  - create a stores subdirectory
//  - create a stores/previous subdirectory
//  - create a stores/updated subdirectory
//  - copy the old schemas from pkg/postgres/schema to schema/old
//  - copy the old stores from their location in central to appropriate subdirectories in stores/previous
//  - generate the new schemas in pkg/postgres/schema and the new stores where they belong
//  - copy the newly generated schemas from pkg/postgres/schema to schema/new
//  - remove the calls to GetSchemaForTable and to RegisterTable from the copied schema files
//  - remove the xxxTableName constant from the copied schema files
//  - copy the newly generated stores from their location in central to appropriate subdirectories in stores/updated
//  - remove any unused function from the copied store files (the minimum for the public API should contain Walk, UpsertMany, DeleteMany)
//  - remove the scoped access control code from the copied store files
//  - remove the metrics collection code from the copied store files

// TODO(dont-merge): Determine if this change breaks a previous releases database.
// If so increment the `MinimumSupportedDBVersionSeqNum` to the `CurrentDBVersionSeqNum` of the release immediately
// following the release that cannot tolerate the change in pkg/migrations/internal/fallback_seq_num.go.
//
// For example, in 4.2 a column `column_v2` is added to replace the `column_v1` column in 4.1.
// All the code from 4.2 onward will not reference `column_v1`. At some point in the future a rollback to 4.1
// will not longer be supported and we want to remove `column_v1`. To do so, we will upgrade the schema to remove
// the column and update the `MinimumSupportedDBVersionSeqNum` to be the value of `CurrentDBVersionSeqNum` in 4.2
// as 4.1 will no longer be supported. The migration process will inform the user of an error when trying to migrate
// to a software version that can no longer be supported by the database.

func migrate(database *types.Databases) error {
	// Use databases.DBCtx to take advantage of the transaction wrapping present in the migration initiator
	pgutils.CreateTableFromModel(database.DBCtx, database.GormDB, schema.CreateTableDeploymentsStmt)

	db := database.PostgresDB

	getStmt := `SELECT image_name_fullname, image_id FROM deployments_containers WHERE image_id is not null AND image_id != '' AND image_name_fullname is not null AND image_name_fullname != ''`
	rows, err := db.Query(database.DBCtx, getStmt)
	if err != nil {
		return err
	}

	containers, err := readRows(rows)
	if err != nil {
		return err
	}
	for _, container := range containers {
		updateStmt := `UPDATE deployments_containers SET image_idv2 = $1 WHERE image_name_fullname = $2 AND image_id = $3`
		_, err = db.Exec(database.DBCtx, updateStmt, uuid.NewV5FromNonUUIDs(container.ImageNameFullName, container.ImageID).String(), container.ImageNameFullName, container.ImageID)
		if err != nil {
			return err
		}
	}

	return nil
}

func readRows(rows *postgres.Rows) ([]*schema.DeploymentsContainers, error) {
	var containers []*schema.DeploymentsContainers

	for rows.Next() {
		var imageName string
		var imageId string

		if err := rows.Scan(&imageName, &imageId); err != nil {
			return nil, pgutils.ErrNilIfNoRows(err)
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

// TODO(dont-merge): Write the additional code to support the migration

// TODO(dont-merge): remove any pending TODO
