package compliance

import (
	"os"
	"path/filepath"
)

// skipProtoInit skips proto type registry initialization for entrypoints
// that use vtprotobuf for all serialization. See generated/storage/00_skip_proto_init.go.
var skipProtoInit = func() bool {
	switch filepath.Base(os.Args[0]) {
	case "central", "roxctl", "migrator":
		return false
	default:
		return true
	}
}()
