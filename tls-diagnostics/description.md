Tool name: stackrox-tls-diagnostics

# Tech Stack

Standard Golang CLI tool.
Can depend on stackrox packages.

# Invocation

Integrate standard Cobra scaffolding.

Currently no subcommands available, but flags might be addedd soon.
e.g. --output=json.

# Operation

When invoked, the tool interacts with the current kubernetes cluster context and emits TLS diagnostics on stdout.

First feature implemented:

The tool needs to identify:
* is stackrox central installed on the cluster?
* is stackrox securedcluster installed on the cluster?
* Which namespaces?
* No need to support Helm/manifest installation, just operator (CRs).
-> identify the ACS setup: "just central", "just secured cluster", "both components, seperate namespaces", "both components, same namespace"

Iterate through all TLS secrets on the cluster and list -- in a human-friendly way -- their metadata, whatever makes sense: issuer, common names, alternative names, algorithm, key size, etc.
Use intermediate data structures for this, so that we can (later) support different output formats (human readable, json, etc.).

