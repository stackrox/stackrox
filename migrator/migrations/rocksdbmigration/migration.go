package rocksdbmigration

var (
	separator = []byte("\x00")
)

// GetPrefixedKey returns fully-qualified key for rocksDB
func GetPrefixedKey(prefix []byte, id []byte) []byte {
	newKey := make([]byte, 0, len(id)+len(prefix)+len(separator))
	newKey = append(newKey, prefix...)
	newKey = append(newKey, separator...)
	newKey = append(newKey, id...)
	return newKey
}

// GetIDFromPrefixedKey removes prefix from key and returns it
func GetIDFromPrefixedKey(prefix []byte, key []byte) []byte {
	return key[len(prefix)+len(separator):]
}

// GetBucketPrefix returns the prefix from bucket name
func GetBucketPrefix(bucket []byte) []byte {
	prefix := make([]byte, 0, len(bucket)+len(separator))
	prefix = append(prefix, bucket...)
	prefix = append(prefix, separator...)
	return prefix
}
