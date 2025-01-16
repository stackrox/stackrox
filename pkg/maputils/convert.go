package maputils

// ConvertStringMapToBytes converts a map of the form map[string]string
// to a map of the form map[string][]byte.
func ConvertStringMapToBytes(stringMap map[string]string) map[string][]byte {
	bytesMap := make(map[string][]byte, len(stringMap))
	for k, v := range stringMap {
		bytesMap[k] = []byte(v)
	}
	return bytesMap
}

// ConvertBytesMapToStrings converts a map of the form map[string][]byte
// to a map of the form map[string]string.
func ConvertBytesMapToStrings(bytesMap map[string][]byte) map[string]string {
	stringMap := make(map[string]string, len(bytesMap))
	for k, v := range bytesMap {
		stringMap[k] = string(v)
	}
	return stringMap
}
