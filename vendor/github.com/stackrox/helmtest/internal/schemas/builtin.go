package schemas

import (
	"embed"
	"io/fs"
	"sync"
)

//go:embed openapi-schemas/*
var builtinSchemasFS embed.FS

var (
	builtinSchemasRegistry     Registry
	builtinSchmeasRegistryInit sync.Once
)

// BuiltinSchemas returns a registry with built-in schemas.
func BuiltinSchemas() Registry {
	builtinSchmeasRegistryInit.Do(func() {
		subFS, err := fs.Sub(builtinSchemasFS, "openapi-schemas")
		if err != nil {
			panic(err)
		}
		builtinSchemasRegistry = newFSBasedRegistry(subFS)
	})
	return builtinSchemasRegistry
}
