//go:build !release || test

package versioncheck

import "testing"

func UnsuppressVersionMismatchWarningForTesting(_ testing.TB) {
	suppressVersionMismatchWarning.Store(false)
}
