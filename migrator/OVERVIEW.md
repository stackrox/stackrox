# StackRox Database Migration Overview

## IMPORTANT
All migrations must be backwards compatible in order to ensure a safe and successful rollback.
This includes schema changes to the Postgres database.

### Migration Process
The migrator always executes on an upgrade of the ACS Central version whether a migration is required or not.
This is because some schema changes for new items may not require a data specific data migration and may just
add new schema elements.  For example, if a version of central added a new table that is new to that version.
Data would not need to be migrated by the schema updates would need to be applied.  The flow of the migrator
is as follows:

- Data migrations (if necessary)
    - Pull data from old Postgres schema if necessary
    - Apply Postgres schema updates for table(s) where data is updated
    - Update the data
- Apply all Postgres schema updates

It is important to note that we use GORM for managing the schemas.  As such we use the GORM auto migration feature
that ensures that no breaking changes are permitted to the schema.  
For more see [GORM Auto Migration](https://gorm.io/docs/migration.html#Auto-Migration)






### Test Backwards Compatibility