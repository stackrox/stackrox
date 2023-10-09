# StackRox Database migration

## IMPORTANT
All migrations must be backwards compatible in order to ensure a safe and successful rollback.

## Purpose of database migrations

When changing the data model for stored objects, data conversions may be required. All migrations
are in the `migrations` subdirectory.

Migrations are organized with sequence numbers and executed in sequence. Each migration is provided
with a pointer to `types.Databases` where the pointed object contains all necessary database instances,
including `Bolt`, `RocksDB`, `Postgres` and `GormDB` (datamodel for postgres) as well as a context `DBCtx`. 
The context allows for the migrations to be wrapped in a transaction so they can be committed as they 
are processed.  (Migrations moving data with `GormDB` will not be part of the outer transaction, 
as such care should be taken when using `GormDB` to move data.)  Depending on the version a migration 
is operating on, some database instances may be nil.

A migration can read from any of the databases, make changes to the data or to the datamodel
(database schema when working with postgres), then persist these changes to the database.

## History of the datastores

1. Before release 3.73, the migrator was targeting internal key-value stores `BoltDB` and `RocksDB`.

   All migrations were of the form `m_{currentDBVersion}_to_m_{currentDBVersion+1}_{summary_of_migrations}` .

2. Release 3.73 and 3.74 brought the database `Postgres` as Technical preview. This had the consequence that
`Postgres` was a new potential datastore targeted by the migrator.

   In these versions, two sets of data migrations were possible: key-value store migrations, and data move migrations
   (from `BoltDB` and `RocksDB` to `Postgres`). The migration sequence is the following: first all key-value
   data migrations are applied, then all data move migrations are applied.

    - Key-value store data migrations have the form `m_{currentDBVersion}_to_m_{currentDBVersion+1}_{summary_of_migrations}` .
    - Data move migrations have the form `n_{postgresSchemaVersion}_to_n_{postgresSchemaVersion+1}_{moved_data_type}` .
    
3. After 4.0, the key-value stores are deprecated and `Postgres` becomes the only data store.

   The migration returns to the old scheme, restarting from the database version after all data moves to `Postgres`.

   All migrations again have the form `m_{currentDBVersion}_to_m_{currentDBVersion+1}_{summary_of_migrations}`.

4. Starting in 4.2, all migrations MUST be backwards compatible, at least while previous releases are supported.  

   Rollbacks from 4.2 to 4.1 will NO LONGER use the `central_previous` database.  These rollbacks will use `central_active` and as such all schema changes and migrations must be backwards compatible.  This means no destructive changes like deleting columns or fields and ensuring that any data changes either work with previous versions or occur within a new field.  When a breaking change is made the `MinimumSupportedDBVersionSeqNum` will need to be updated to the minimum database version that will work with the breaking change.

## How to write new migration script

Script should correspond to single change. Script should be part of the same release as this change.
Here are the steps to write migration script:

1. Run `DESCRIPTION="xxx" make bootstrap_migration` with a proper description of what the migration will do
   in the `DESCRIPTION` environment variable.

2. Determine if this change breaks a previous releases database.  If so increment the `MinimumSupportedDBVersionSeqNum` 
   to the `CurrentDBVersionSeqNum` of the release immediately following the release that cannot tolerate the change. 
   For example, in 4.2 a column `column_v2` is added to replace the `column_v1` column in 4.1.  All the code from 4.2
   onward will not reference `column_v1`.  At some point in the future a rollback to 4.1 will not longer be supported
   and we want to remove `column_v1`.  To do so, we will upgrade the schema to remove the column
   and update the `MinimumSupportedDBVersionSeqNum` to be the value of `CurrentDBVersionSeqNum` in 4.2
   as 4.1 will no longer be supported.  The migration process will inform the user of an error when trying to migrate
   to a software version that can no longer be supported by the database.

3. Write the migration code and associated tests in the generated `migration_impl.go` and `migration_test.go` files.
   The files contain a number of TODOs to help with the tasks to complete when writing the migration code itself.

4. To better understand how to write the `migration.go` and `migration_test.go` files, look at existing examples
in `migrations` directory, or at the examples listed below.

    - [#1](https://github.com/stackrox/rox/pull/8609)
    - [#2](https://github.com/stackrox/rox/pull/7581)
    - [#3](https://github.com/stackrox/rox/pull/7921)

    Avoid depending on code that might change in the future as **migration should produce consistent results**.

## How to test migration on locally deployed cluster

1. Create PR with migration files to build image in CI
2. Checkout **before** commit with migration files and `make clean image`
3. `export STORAGE=pvc`
4. `teardown && ./deploy/k8s/deploy-local.sh`
5. `./scripts/k8s/local-port-forward.sh`
6. Create all necessary testing data via central UI and REST endpoints
7. Checkout **at the same commit** your PR currently pointing to
8. `kubectl -n stackrox set image deploy/central central=stackrox/main:$(make tag)`
9. You can ensure migration script was executed by looking into Central logs. You should see next log messages:

    ```bigquery
    Migrator: <timestamp> log.go:18: Info: Found DB at version <currentDBVersion>, which is less than what we expect (<currentDBVersion+1>). Running migrations...
    Migrator: <timestamp> log.go:18: Info: Successfully updated DB from version <currentDBVersion> to <currentDBVersion+1>
    ```

10. Re-run `./scripts/k8s/local-port-forward.sh`
11. Verify that migration worked correctly

## Writing postgres migration tests

Follow the TODOs listed in `migration_test.go`.  This includes a recommended test to verify the pre-migration SQL statements provide
the expected results against the post-migration database in order to verify backwards compatiblity.

### Migrator limitations

Migrator upgrades the data from a previous datamodel to the current one. In the case of data manipulation migrations,
the previous and current datamodels can be identical. Now the previous and current datamodels can differ. When loading
data, the migrator needs to access the legacy schema. Each migration may apply a different schema change, therefore
the current datastore or schema code cannot be used.

Each migration is responsible for looking into the databases (there can be multiple ones) it needs to load the data
it manipulates and for converting it.

### Possible migration steps

#### Freeze current schema as needed.

A migration shall not access current schema under `pkg/postgres/schema`. The migration access data in the format
of its creation time and bring the format of the next migration. It is associated with a specific sequence number(aka version)
and hence it does not evolve with the latest version. The frozen schema helps to keep migration separated
from Central and keep the codes of migrations stable.

To freeze a schema, you can use the following tool to generate a frozen schema, make sure use the exact same parameters
to generate current schema which can be find in each Postgres store.

```shell
pg-schema-migration-helper --type=<prototype> --search-category ...
```

This tool also generates conversion tools for schema, you may remove them. 

#### Create or upgrade the schema of a table.

For the initial data push to postgres, the generated schemas under `migrator/migrations/frozenschema/v73` were used.
The auto-generation scripts are removed so the schemas are frozen after 3.73.

In 3.74, snapshots of the schemas are taken with an on-demand basis.

Starting from release 4.0, we recommend to keep frozen schemas inside a new migration:
- If the migration does not change the schema but need to access the data of a table, it needs to freeze its schema in the migration.
- If the migration changes the schema of a table, it may need to freeze two versions of its schema before and after the schema change.

If your new migration need to change the Postgres schema, use the following statement to apply frozen schema in a migration.

```go
pgutils.CreateTableFromModel(context.Background(), gormDB, frozenSchema.CreateTableCollectionsStmt)
```

#### Access data

Migration can access the legacy and Postgres databases but it does not have access to the central datastores.
In migrator, there are a multiple ways to access data.

1. Raw SQL commands

    Raw SQL commands are always available to databases and it has good isolation from current release. It is used frequently in
    migrations before Postgres. Migrations with raw SQL command needs less maintenance but it may not be convenient
    and it could be error-prone.  This model supports the transaction passed via the databases.DBCtx.
    We try to provide more convenient way to read and update the data.

2. Gorm

    Use Gorm to read small amount data. Gorm is light-weighted and comprehensive ORM allowing accessing databases
    in an object oriented way. You may have partial data access by trimming the gorm model.
    Check the [details](https://gorm.io/docs/) how to use Gorm.

    The example below, illustrates how to walk the object to populate a field that was promoted to a column.  Note that
    we must explicitly narrow the fields we select.  If we select the whole object, gorm will default to a `select *`
    and in that case a subsquent migration modifying the structure of the same table will fail because the statement
    cach will be invalid
    
    ```go
    func migrate(database *types.Databases) error {
        // We are simply promoting a field to a column so the serialized object is unchanged.  Thus, we
        // have no need to worry about the old schema and can simply perform all our work on the new one.
        db := database.GormDB
        pgutils.CreateTableFromModel(database.DBCtx, db, schema.CreateTableListeningEndpointsStmt)
        db = db.WithContext(database.DBCtx).Table(schema.ListeningEndpointsTableName)
        query := db.WithContext(database.DBCtx).Table(schema.ListeningEndpointsTableName).Select("serialized")

        rows, err := query.Rows()
        if err != nil {
            return errors.Wrapf(err, "failed to iterate table %s", schema.ListeningEndpointsTableName)
        }
        defer func() { _ = rows.Close() }()

        var convertedPLOPs []*schema.ListeningEndpoints
        var count int
        for rows.Next() {
            var plop *schema.ListeningEndpoints
            if err = query.ScanRows(rows, &plop); err != nil {
                return errors.Wrap(err, "failed to scan rows")
            }

            plopProto, err := schema.ConvertProcessListeningOnPortStorageToProto(plop)
            if err != nil {
                return errors.Wrapf(err, "failed to convert %+v to proto", plop)
            }

            converted, err := schema.ConvertProcessListeningOnPortStorageFromProto(plopProto)
            if err != nil {
                return errors.Wrapf(err, "failed to convert from proto %+v", plopProto)
            }
            convertedPLOPs = append(convertedPLOPs, converted)
            count++

            if len(convertedPLOPs) == batchSize {
                // Upsert converted blobs
                if err = db.Clauses(clause.OnConflict{UpdateAll: true}).Model(schema.CreateTableListeningEndpointsStmt.GormModel).Create(&convertedPLOPs).Error; err != nil {
                    return errors.Wrapf(err, "failed to upsert converted %d objects after %d upserted", len(convertedPLOPs), count-len(convertedPLOPs))
                }
                convertedPLOPs = convertedPLOPs[:0]
            }
        }

        if err := rows.Err(); err != nil {
            return errors.Wrapf(err, "failed to get rows for %s", schema.ListeningEndpointsTableName)
        }

        if len(convertedPLOPs) > 0 {
            if err = db.Clauses(clause.OnConflict{UpdateAll: true}).Model(schema.CreateTableListeningEndpointsStmt.GormModel).Create(&convertedPLOPs).Error; err != nil {
                return errors.Wrapf(err, "failed to upsert last %d objects", len(convertedPLOPs))
            }
        }
        log.Infof("Converted %d plop records", count)

        return nil
    }
    ```

    Another example, to get all the image id and operating system from the image table, a Gorm model is needed first.
    As not all data is needed, a trimmed model can be used. All Gorm models can be read from `pkg/postgres/schema`,
    copied and trimmed.

   ```go
   type imageIDAndOperatingSystem struct {
       ID                  string `gorm:"column:id;type:varchar;primaryKey"`
       ScanOperatingSystem string `gorm:"column:scan_operatingsystem;type:varchar"`
   }
   ```
   
    The trimmed Gorm model can be used to read from database.

   ```go
   imageTable := gormDB.Table(pkgSchema.ImagesSchema.Table).Model(pkgSchema.CreateTableImagesStmt.GormModel)
   var imageCount int64
   if err := imageTable.Count(&imageCount).Error; err != nil {
       return err
   }
   imageBuf := make([]imageIDAndOs, 0, imageBatchSize)
   imageToOsMap := make(map[string]string, imageCount)
   result := imageTable.FindInBatches(&imageBuf, imageBatchSize, func(_ *gorm.DB, batch int) error {
       for _, sub := range imageBuf {
         imageToOsMap[sub.ID] = sub.ScanOperatingSystem
       }
       return nil
   })
   ```

    Most protobuf object fields are wrapped in the serialized field in Postgres tables. The fields from the Gorm model
    need to be deserialized/serialized to read/update the objects. In case a field has to be updated,
    the protobuf object has to be serialized. To address this issue, a tool is provided to convert a protobuf object
    to/from a Gorm model.

   ```go
   type ClusterHealthStatuses struct {
       Id         string    `gorm:"column:id;type:varchar;primaryKey"`
       Serialized []byte    `gorm:"column:serialized;type:bytea"`
   }
   ```
   Gorm cannot participate in the transaction passed via the context.  The risk and possible ramifications of that
   should be considered before deciding to use Gorm to move the data.

3. Duplicate the Postgres Store
   This method is used in version 73 and 74 to migrate all tables from RocksDB to Postgres. In addition to frozen schema,
   the store to access the data are also frozen for migration. The migrations with this method are closely associated
   with current release eg. search/delete with schema and the prototypes of the objects. This method is NOT recommended for
   4.0 and beyond.
   This model supports the transaction passed via the databases.DBCtx.

#### Conversion tool

All tables in central store protobuf objects in serialized form. The `serialized` column needs to be updated with
the other columns.

Doing the conversion for each migration is cumbersome and not convenient. A tool exists to generate the conversion 
functions. The conversion functions can be generated and then tailored to be used in the migrations.

The following shows an example of the conversion functions.

The tool is `pg-schema-migration-helper`, it can be used as follows.

```shell
pg-schema-migration-helper --type=storage.VulnerabilityRequest --search-category VULN_REQUEST
```

`pg-schema-migration-helper` uses the same elements as `pg-table-bindings-wrapper` (the code generator for the postgres
stores in central). These elements are the protobuf types and search-categories. It uses these elements to generate the
schema and conversion functions.

More examples with test protobuf objects are available in `migrator/migrations/postgreshelper/schema`
```go
func convertVulnerabilityRequestFromProto(obj *storage.VulnerabilityRequest) (*schema.VulnerabilityRequests, error) {
        serialized, err := obj.Marshal()
        if err != nil {
                return nil, err
        }
        model := &schema.VulnerabilityRequests{
                Id:                                obj.GetId(),
                TargetState:                       obj.GetTargetState(),
                Status:                            obj.GetStatus(),
                Expired:                           obj.GetExpired(),
                RequestorName:                     obj.GetRequestor().GetName(),
                CreatedAt:                         pgutils.NilOrTime(obj.GetCreatedAt()),
                LastUpdated:                       pgutils.NilOrTime(obj.GetLastUpdated()),
                DeferralReqExpiryExpiresWhenFixed: obj.GetDeferralReq().GetExpiry().GetExpiresWhenFixed(),
                DeferralReqExpiryExpiresOn:        pgutils.NilOrTime(obj.GetDeferralReq().GetExpiry().GetExpiresOn()),
                CvesIds:                           pq.Array(obj.GetCves().GetIds()).(*pq.StringArray),
                Serialized:                        serialized,
        }
        return model, nil
}
 
func convertSlimUserFromProto(obj *storage.SlimUser, idx int, vulnerability_requests_Id string) (*schema.VulnerabilityRequestsApprovers, error) {
        model := &schema.VulnerabilityRequestsApprovers{
                VulnerabilityRequestsId: vulnerability_requests_Id,
                Idx:                     idx,
                Name:                    obj.GetName(),
        }
        return model, nil
}
 
func convertRequestCommentFromProto(obj *storage.RequestComment, idx int, vulnerability_requests_Id string) (*schema.VulnerabilityRequestsComments, error) {
        model := &schema.VulnerabilityRequestsComments{
                VulnerabilityRequestsId: vulnerability_requests_Id,
                Idx:                     idx,
                UserName:                obj.GetUser().GetName(),
        }
        return model, nil
}
```