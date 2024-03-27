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
# If you have a central database running locally
$ psql -U postgres -h localhost -d central_data -t \
  -c 'SELECT serialized FROM integration_healths' | \
  go run ./tools/deserialize-proto --type storage.IntegrationHealth
```