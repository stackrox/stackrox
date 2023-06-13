//go:build linux

package odirect

import "syscall"

const boltWriteFlag = syscall.O_DIRECT

// GetODirectFlag gets the value of O_DIRECT on the give os
func GetODirectFlag() int {
	return boltWriteFlag
}
