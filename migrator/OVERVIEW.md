# StackRox Database Migration Overview

## IMPORTANT
All migrations must be backwards compatible in order to ensure a safe and successful rollback.
This includes schema changes to the Postgres database.  Additionally the migration should be 
idempotent because it will be executed again when rolling forward after a rollback.

The migrations are forward ONLY.  There is no concept of reverse migrations on the rollback as
our current deployment mechanisms make that a complicated endeavor.  So it is imperative that
we maintain database compatibility from version to version.

For example 4.1 -> 4.5 will run all migrations between 4.1 and 4.5.  4.4 -> 4.5 will only run the migrations between 4.4 and 4.5.
If an upgrade from 4.5 forced a rollback to 4.4, 4.4 needs to be able to operate on the 4.5 schema and data.
When the subsequent upgrade to 4.5.X occurs, all the migrations between 4.4 and 4.5.X will execute again.

As of 4.5 the database is backwards compatible back to 4.1.  There will arise a need to bump this to more
recent release as the product grows.  So in the event we need to update the oldest version the schema is compatible with we 
need to change the `MinimumSupportedDBVersionSeqNum` to be the first migration sequence number of the target 
version.  For example if we need to update that to 4.3 we would need to set the `MinimumSupportedDBVersionSeqNum`
to be 193.

### Migration Process
The migrator always executes on an upgrade of the ACS Central version whether a migration is required or not.
This is because some schema changes for new items may not require a data specific data migration and may just
add new schema elements.  For example, if a version of central added a new table that is new to that version.
Data would not need to be migrated by the schema updates would need to be applied.  The flow of the migrator
is as follows:

- Data migrations (if necessary)
    - Pull data from old Postgres schema if necessary
    - Apply Postgres schema updates with GORM AutoMigrate for table(s) where data is updated
    - Update the data
- Apply all Postgres schema updates with GORM AutoMigrate

It is important to note that we use GORM for managing the schemas.  As such we use the GORM auto migration feature
that ensures that no breaking changes are permitted to the schema.  For instance, GORM will not remove unused 
columns as that would break upon a rollback.  Additinally GORM will not make data type changes though it will
perform updates on precision and such as those are not breaking.
For more see [GORM Auto Migration](https://gorm.io/docs/migration.html#Auto-Migration)

### Test Backwards Compatibility
The `gke-upgrade-tests` provide broad stroke tests of the upgrade functionality. The upgrade test
starts with a 4.1.3 deployment and upgrades to the current release.  This will execute all the migrations
between 4.1.3 and the current release.  Additionally it verifies that the rollback to 4.1.3 still
succeeds.  This provides a general idea that the database is forward and backwards compatible.

Beyond that for any schema and/or persisted data changes, the engineer should test forwards and backwards 
compatiblity within the scope of their change.
To verify the change are compatible the engineer should follow this process:

- Deploy previous version
- Upgrade to current version
- Exercise the change, i.e. populate any necessary data.
- Rollback to the previous version
- Verify central is up and functioning normally
- Roll forward to the current version
- Verify any migrations were executed
- Exercise the change to ensure functionality remains.

Even if the upgrade test walked more possible versions, it still may miss the part of exercising the change.  
So it is imperative that the engineers also help to test and ensure their changes are compatible.
