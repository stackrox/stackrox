package postgres

//go:generate pg-table-bindings-wrapper --type=storage.VirtualMachineComponentV2 --search-category VIRTUAL_MACHINE_COMPONENTS_V2 --schema-only --search-scope VIRTUAL_MACHINE_VULNERABILITIES_V2,VIRTUAL_MACHINE_COMPONENTS_V2,VIRTUAL_MACHINE_SCANS_V2,VIRTUAL_MACHINES_V2,NAMESPACES,CLUSTERS --references=storage.VirtualMachineScanV2
