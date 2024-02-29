# Operator Tools

Subdirectories contain various tools used by the operator.

Each tool must have its own directory and go module. 
Otherwise, there might be conflicting dependencies between tools and versions.
e.g. The dreaded `gnostic` dependency.

# Adding a tool

1. Create a new directory for the tool, as well as a `go.mod` file. The module should be `github.com/stackrox/rox/operator/tools/<tool>`
2. Copy the `noop.go` file to the directory. Check [here](./yq/noop.go) for an example.
3. Add a `tool.go` file to the directory. Add a `import _ "<tool-module>"` statement to the file.
4. Add a `go-tool` statement in the `Makefile`. For example, `$(call go-tool, YQ, github.com/mikefarah/yq/v4, tools/yq)`
5. Add a [dependabot configuration](./../../.github/dependabot.yaml) for the tool.
6. Run `make <tool>` to install the tool locally.
7. Profit