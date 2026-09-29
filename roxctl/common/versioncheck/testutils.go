//go:build !release || test

package versioncheck

import "testing"

func ResetSuppressWarningForTesting(_ testing.TB) {
	suppressWarning.Store(false)
}
