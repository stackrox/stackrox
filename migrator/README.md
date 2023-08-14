# StackRox Database migration

## Purpose of database migrations

When changing the data model for stored objects, data conversions may be required. All migrations
are in the `migrations` subdirectory.

Migrations are organized with sequence numbers and executed in sequence. Each migration is provided
with a pointer to `types.Databases` where the pointed object contains all necessary database instances,
including `Bolt`, `RocksDB`, `Postgres` and `GormDB` (datamodel for postgres). Depending on the version
a migration is operating on, some database instances may be nil.

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

1. Run `DESCRIPTION="xxx" STORE_OBJECT="storage.Object0,storage.Object1" make bootstrap_migration` with a proper description of what the migration will do
   in the `DESCRIPTION` environment variable.
and OPTIONALLY with the objects being migrated to create `gen.go` files for each object being migrated
    in the optional `STORE_OBJECT` environment variable.

2. If using the optional `STORE_OBJECT` parameter, generate the migration stores with the following command:
    `gogen -run pg-table-bindings`

3. Determine if this change breaks a previous releases database.  If so increment the `MinimumSupportedDBVersionSeqNum` to
   the `CurrentDBVersionSeqNum` of the release immediately following the release that cannot tolerate the change. For
   example, in 4.2 a column `column_v2` is added to replace the `column_v1` column in 4.1.  All the code from 4.2
   onward will not reference `column_v1`.  At some point in the future a rollback to 4.1 will not longer be supported
   and we want to remove `column_v1`.  To do so, we will upgrade the schema to remove the column
   and update the `MinimumSupportedDBVersionSeqNum` to be the value of `CurrentDBVersionSeqNum` in 4.2
   as 4.1 will no longer be supported.  The migration process will inform the user of an error when trying to migrate
   to a software version that can no longer be supported by the database.

4. Write the migration code and associated tests in the generated `migration_impl.go` and `migration_test.go` files.
   The files contain a number of TODOs to help with the tasks to complete when writing the migration code itself.

5. To better understand how to write the `migration.go` and `migration_test.go` files, look at existing examples
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

This tool also generates conversion tools for schema, you may remove the 

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
    and it could be error-prone.
    We try to provide more convenient way to read and update the data.

2. Gorm

    Use Gorm to read small amount data. Gorm is light-weighted and comprehensive ORM allowing accessing databases
    in an object oriented way. You may have partial data access by trimming the gorm model.
    Check the [details](https://gorm.io/docs/) how to use Gorm.

    For example, to get all the image id and operating system from the image table, a Gorm model is needed first.
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

3. Stores

    Generate a version of the Postgres Store based off the migrator's Generic Store via the bootstrap process.
    The migrations with this method are closely associated with current release eg. search/delete with schema
    and the prototypes of the objects. The search schema and the prototypes of the objects must be backwards
    compatible.  The advantage of this approach is it would be more familiar for the engineers.

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