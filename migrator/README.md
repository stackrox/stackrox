# StackRox Database migration

## Purpose of database migrations

When changing the data model for stored objects, data conversions may be required. All migrations
are in the `migrations` subdirectory.

Migrations are organized with sequence numbers and executed in sequence. Each migration is provided
with a pointer to `types.Databases` where the pointed object contains all necessary database instances,
including `Bolt`, `RocksDB`, `Postgres` and `GormDB` (datamodel for postgres).

A migration can read from any of the databases, make changes to the data or to the datamodel
(database schema when working with postgres), then persist these changes to the database.

## How to write new migration script

Script should correspond to single change. Script should be part of the same release as this change.
Here are the steps to write migration script:

1. Determine which database group will be targeted. The options should be `BoltDB or RocksDB`,
`Postgres and Gorm` or a migration from the former to the latter.

    Note: the rule of thumb to determine the target group is based on the first release where the migration should be
    applied.
    1. Before 3.73, the only target group should be `BoltDB or RocksDB`.
    2. For 3.73, there should be two sets of migrations. One for the data changes, targeting the `BoltDB or RocksDB`
        group, and a set of migrations loading the data from that group and pushing it to `Postgres`.
    3. After 3.73, the only target group should be `Postgres`.

2. Lookup current database version in `pkg/migrations/internal/seq_num.go` file. 

    Use `PostgresDBVersionPlus` if the target database is `Postgres`, `CurrentDBVersionSeqNum` otherwise.

    The selected variable will be referred to as `currentDBVersion` in the next steps.
 
3. Under `migrations` folder create new folder with name
`{prefix}_{currentDBVersion}_to_{prefix}_{currentDBVersion+1}_{summary_of_migration}`

    2. Use the prefix `n` if `Postgres` is in the target database group, `m` otherwise.
    3. Ensure that the `summary_of_migration` follows the naming convention of previous migrations,
        i.e., postfix `_policy` if it modifies policies

4. Create at least two files: `migration.go` and `migration_test.go`. These files should belong to package
`{prefix}{currentDBVersion}to{prefix}{currentDBVersion+1}`

5. The `migration.go` file should contain at least the following elements:
    ```go
   import (
       "github.com/stackrox/rox/migrator/migrations"
       "github.com/stackrox/rox/migrator/types"
       // If needed for the sequence number management.
       pkgMigrations "github.com/stackrox/rox/pkg/migrations"
   )
   
   var (
       startSeqNum = // see below
       /*
        * If the migration writes to Postgres, startSeqNum should be:
        *     pkgMigrations.CurrentDBVersionSeqNumWithougPostgres() + PostgresDBVersionPlus
        * Otherwise:
        *     CurrentDBVersionSeqNum
        * The values for CurrentDBVersionSeqNum and PostgresDBVersionPlus are extracted from
        * pkg/migrations/internal/seq_num.go earlier.
        */ 
       migration = types.Migration{
           StartingSeqNum: startSeqNum,
           VersionAfter: startSeqNum+1,
           Run: func(database *types.Databases) error {
               // Migration code ..
           },
       }
   )
   
   func init() {
       migrations.MustRegisterMigration(migration)
   }
   
   // Additional code to support the migration code
    ```
6. Add to `migrator/runner/all.go` line

    ```go
    _ "github.com/stackrox/rox/migrator/migrations/m_{currentDBVersion}_to_m_{currentDBVersion+1}_{summary_of_migration}"
    ```

7. Increment the `currentDBVersion` sequence number variable used from `pkg/migrations/internal/seq_num.go` by one.

8. To better understand how to write the `migration.go` and `migration_test.go` files, look at existing examples
in `migrations` directory, or at the examples listed below.

    - [#1](https://github.com/stackrox/rox/pull/8609)
    - [#2](https://github.com/stackrox/rox/pull/7581)
    - [#3](https://github.com/stackrox/rox/pull/7921)

    Avoid depending on code that might change in the future as **migration should produce consistent results**.

## How to test migration on locally deployed cluster

1. Create PR with migration files to build image in CircleCI
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

#### Create or upgrade the schema of a table.

```go
pkgSchema.ApplySchemaForTable(context.Background(), databases.PostgresDB, <schema>)
```

Note: the schema should be the Postgres schema at the version of migration. It does not evolve with the latest version
and is associated with a specific sequence number (aka version).

For the initial data push to postgres, the auto-generated schemas under `pkg/postgres/schema` were used.

Starting with release 3.73, snapshots of the schemas will be taken for each release of Postgres Datastore.

#### Access data

Migration can access the legacy and Postgres databases but it does not have access to the central datastores.
In migrator, there are a multiple ways to access data.

1. Raw SQL commands

    Raw SQL commands are always available to databases. But it may not be convenient and it could be error-prone.
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