package utils

// CopyKeys returns a copy of a list of keys.
func CopyKeys(keys [][]byte) [][]byte {
	ret := make([][]byte, len(keys))
	for i := 0; i < len(keys); i++ {
		ret[i] = append([]byte{}, keys[i]...)
	}
	return ret
}
