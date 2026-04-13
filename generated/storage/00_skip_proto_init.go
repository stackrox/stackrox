package storage

import (
	"os"
	"path/filepath"
)

// skipProtoInit skips proto type registry initialization for entrypoints
// that use vtprotobuf for all serialization (sensor, admission-control,
// config-controller). Central, roxctl, and migrator need the registry
// for protojson/reflection and run with full initialization.
// Saves ~10-15 MB of heap for secured cluster components.
var skipProtoInit = func() bool {
	name := filepath.Base(os.Args[0])
	switch name {
	case "central", "roxctl", "migrator":
		return false
	default:
		return true
	}
}()
