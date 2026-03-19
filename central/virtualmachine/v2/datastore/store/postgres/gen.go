package postgres

// NOTE: This store uses a custom implementation (store.go) instead of the
// auto-generated one. The go:generate directive below only generates the
// schema, not the store.
//go:generate pg-table-bindings-wrapper --type=storage.VirtualMachineV2 --search-category VIRTUAL_MACHINES_V2 --schema-only --search-scope VIRTUAL_MACHINE_VULNERABILITIES_V2,VIRTUAL_MACHINE_COMPONENTS_V2,VIRTUAL_MACHINE_SCANS_V2,VIRTUAL_MACHINES_V2,NAMESPACES,CLUSTERS
