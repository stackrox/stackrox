//go:build darwin

package odirect

const boltWriteFlag = 0x0

// GetODirectFlag gets the value of O_DIRECT on the give os
func GetODirectFlag() int {
	return boltWriteFlag
}
