//go:build !release || test

package versioncheck

import "testing"

func ResetSuppressVersionMismatchWarningForTesting(_ testing.TB) {
	suppressVersionMismatchWarning.Store(false)
}
