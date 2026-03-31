package postgres

//go:generate pg-table-bindings-wrapper --type=storage.VirtualMachineScanV2 --search-category VIRTUAL_MACHINE_SCANS_V2 --no-copy-from --search-scope VIRTUAL_MACHINE_VULNERABILITIES_V2,VIRTUAL_MACHINE_COMPONENTS_V2,VIRTUAL_MACHINE_SCANS_V2,VIRTUAL_MACHINES_V2,NAMESPACES,CLUSTERS --references=storage.VirtualMachineV2
