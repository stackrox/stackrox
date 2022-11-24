package oomcheck

import "testing"

func TestItDoesNotPanic(t *testing.T) {
	checkUsageAndReport()
}
