# Custom Route API specs

This directory includes manually created OpenAPI/Swagger specs for Central's custom routes defined in [/central/main.go](/central/main.go) (refer to `customRoutes()`)

These specs will roll up under the v1 API inside of Central as well as in the official docs on docs.redhat.com.

## Creating a new spec for a custom route

Creating a new spec for a custom route involves a few steps.

If you want to maintain the spec in YAML:

1. Create a YAML-based swagger spec from scratch in the file format `<serviceName>.swagger.yaml` in this directory.
2. Convert the YAML to JSON and save the JSON in the file format `<serviceName>.swagger.json` in this directory.
3. Add both files to git.

If you want to maintain the spec in JSON:

1. Create a JSON-based spec from scratch in the file format `<serviceName>.swagger.json` in this directory.
2. Add the file to git.

From there the swagger spec automation will pick up the new specs.
