package testutils

import (
	"testing"

	"github.com/stackrox/stackrox/pkg/migrations/internal"
	"github.com/stackrox/stackrox/pkg/testutils"
)

// SetCurrentDBSequenceNumber is used in unit test only
func SetCurrentDBSequenceNumber(t *testing.T, seqNum int) {
	testutils.MustBeInTest(t)
	internal.CurrentDBVersionSeqNum = seqNum
}

// SetDBMountPath is used for unit test only
func SetDBMountPath(t *testing.T, dbPath string) {
	testutils.MustBeInTest(t)
	internal.DBMountPath = dbPath
}
