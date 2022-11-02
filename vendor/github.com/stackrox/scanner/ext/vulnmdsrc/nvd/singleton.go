package nvd

import (
	"github.com/stackrox/scanner/ext/vulnmdsrc/types"
)

var (
	nvdAppender = &appender{}
)

// SingletonAppender returns the instance of the NVD appender.
func SingletonAppender() types.Appender {
	return nvdAppender
}
