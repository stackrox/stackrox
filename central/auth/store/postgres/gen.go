package postgres

//go:generate pg-table-bindings-wrapper --type=storage.AuthMachineToMachineConfig --references=storage.Role --get-all-func
