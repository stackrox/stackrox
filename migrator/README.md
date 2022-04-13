# StackRox Database migration

## How to write new migration script

Script should correspond to single change. Script should be part of the same release as this change.
Here are the steps to write migration script:

1. Lookup current database version in `pkg/migrations/internal/seq_num.go` file
2. Under `migrations` folder create new folder with name
`m_{currentDBVersion}_to_m_{currentDBVersion+1}_{summary_of_migration}`
    1. Ensure that the `summary_of_migration` follows the naming convention of previous migrations, i.e., postfix `_policy` if it modifies policies
3. Create at least two files: `migration.go` and `migration_test.go`. These files should belong to package `m{currentDBVersion}tom{currentDBVersion+1}`
4. To better understand how to write these two files, look at existing examples: [#1](https://github.com/stackrox/stackrox/pull/8609) [#2](https://github.com/stackrox/stackrox/pull/7581) [#3](https://github.com/stackrox/stackrox/pull/7921) in `migrations` directory. Avoid depending on code that might change in the future as **migration should produce consistent results**.
5. Add to `migrator/runner/all.go` line

    ```go
    _ "github.com/stackrox/stackrox/migrator/migrations/m_{currentDBVersion}_to_m_{currentDBVersion+1}_{summary_of_migration}"
    ```

6. Increment the currentDBVersion to currentDBVersion+1

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
