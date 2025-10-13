# How to execute tests requiring a running Postgres

1. Set a Postgres password:
    ```
    export POSTGRES_PASSWORD=mysecret
    ```
1. Run Postgres:
    ```
    $ docker run --rm -d --name my-postgres -e POSTGRES_USER="${USER}" -e POSTGRES_DB=postgres -e POSTGRES_PASSWORD="${POSTGRES_PASSWORD}" -p 5432:5432 postgres
    ```
1. Execute tests, specifying the `sql_integration` tag. For example:
    ```
    $ go test -tags sql_integration -v -count 1 ./... -run TestClusterInitBackend
    ```
