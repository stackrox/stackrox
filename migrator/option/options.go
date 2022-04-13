package option

import (
	"github.com/stackrox/stackrox/pkg/migrations"
)

// MigratorOptions contain all options for migrators to run.
var MigratorOptions = struct {
	DBPathBase string
}{
	DBPathBase: migrations.CurrentPath(),
}
