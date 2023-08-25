package postgres

import (
	"runtime/debug"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/utils"
)

// DeprecatedCall logs the caller of this function
// This helps trace where calls into deprecated features are coming from.
// e.g. calls into RocksDB and BoltDB initialization
func DeprecatedCall(name string) {
	utils.Should(errors.Errorf("unexpected call to legacy database %q", name))
	debug.PrintStack()
}
