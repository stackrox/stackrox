package option

import "github.com/stackrox/rox/pkg/migrations"

// CentralOptions contain all the central options.
var CentralOptions = struct {
	DBPathBase string
}{
	DBPathBase: migrations.CurrentPath(),
}
