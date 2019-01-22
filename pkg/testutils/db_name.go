package testutils

import (
	"strings"

	"github.com/stretchr/testify/suite"
)

// DBFileName returns an appropriate, unique, DB file name for the given suite.
// It works equally well whether used in SetupSuite or SetupTest.
func DBFileName(suite suite.Suite) string {
	return strings.Replace(suite.T().Name(), "/", "_", -1) + ".db"
}
