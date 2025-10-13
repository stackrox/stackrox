# deserialize-proto

Internal tool to deserialize central database objects for debugging purposes. 

## Usage

This tool reads from stdin, so you'll need to copy the deserialized value and pipe it in or run a psql command and pipe
in the stdout of that command.

Here are some examples:

```sh
# If you're on macOS and copied the deserialized object to your clipboard
$ pbpaste | go run ./tools/deserialize-proto --type storage.IntegrationHealth
```

```sh
# Read from stdin with a local database
$ psql -U postgres -h localhost -d central_data -t \
  -c 'SELECT serialized FROM integration_healths' | \
  go run ./tools/deserialize-proto --type storage.IntegrationHealth --stdin

# Connect automatically to a local database and print all entries.
$ go run tools/deserialize-proto/main.go --type storage.Policy

# Print entry by id.
$ go run tools/deserialize-proto/main.go --type storage.Policy --id <UUID>

# Connect to a configured database and run a custom where clause.
$ go run tools/deserialize-proto/main.go --type storage.Policy --where "name='OpenShift: Central Admin Secret Accessed'"

# Connect to a configured database.
$ POSTGRES_PASSWORD=password USER=postgres POSTGRES_PORT=5432 POSTGRES_HOST=localhost go run tools/deserialize-proto/main.go --type storage.Policy
```