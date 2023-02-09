package testutils

import (
	"testing"

	"github.com/stackrox/rox/pkg/migrations/internal"
	"github.com/stackrox/rox/pkg/testutils/utils"
)

// SetCurrentDBSequenceNumber is used in unit test only
func SetCurrentDBSequenceNumber(t *testing.T, seqNum int) {
	utils.MustBeInTest(t)
	internal.CurrentDBVersionSeqNum = seqNum
	// This is to make sure unit tests can emulate earlier versions correctly.
	internal.LastRocksDBVersionSeqNum = seqNum
}

// SetDBMountPath is used for unit test only
func SetDBMountPath(t *testing.T, dbPath string) {
	utils.MustBeInTest(t)
	internal.DBMountPath = dbPath
}
