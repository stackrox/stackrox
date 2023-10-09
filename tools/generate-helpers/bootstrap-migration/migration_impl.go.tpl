{{- define "TODO"}}TODO(do{{- /**/ -}}nt-merge){{end -}}
package {{.packageName}}

import (
	"github.com/stackrox/rox/migrator/types"
)

// {{template "TODO"}}: generate/write and import any store required for the migration (skip any unnecessary step):
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

// {{template "TODO"}}: Determine if this change breaks a previous releases database.
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
	_ = database // {{template "TODO"}}: remove this line, it is there to make the compiler happy while the migration code is being written.
	// Use databases.DBCtx to take advantage of the transaction wrapping present in the migration initiator

	// {{template "TODO"}}: Migration code comes here
	// {{template "TODO"}}: When using gorm, make sure you use a separate handle for the updates and the query.  Such as:
	// db = db.WithContext(database.DBCtx).Table(schema.ListeningEndpointsTableName)
	// query := db.WithContext(database.DBCtx).Table(schema.ListeningEndpointsTableName).Select("serialized")
	// See README for more details

	return nil
}

// {{template "TODO"}}: Write the additional code to support the migration

// {{template "TODO"}}: remove any pending TODO
