package redhat

import (
	"github.com/stackrox/scanner/ext/vulnmdsrc/types"
)

var (
	redhatAppender = &appender{}
)

// SingletonAppender returns the instance of the Red Hat appender.
func SingletonAppender() types.Appender {
	return redhatAppender
}
