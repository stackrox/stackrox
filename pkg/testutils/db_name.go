package testutils

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
)

// DBFileName returns an appropriate, unique, DB file name for the given suite.
// It works equally well whether used in SetupSuite or SetupTest.
func DBFileName(suite suite.Suite) string {
	return DBFileNameForT(suite.T())
}

// DBFileNameForT returns an appropriate, unique, DB file name for the given test.
func DBFileNameForT(t *testing.T) string {
	return strings.Replace(t.Name(), "/", "_", -1) + ".db"
}
