package storecache

// Cache offers a generic interface for cache implementations.  Structs implementing this interface are expected to be
// thread safe.  Structs implementing this interface must have the following properties:
// The cache must never update an existing key with a new value that has an older version
// The cache must never write to any empty key with a version lower than the last Removed version
type Cache interface {
	Add(key, value interface{}, version uint64)
	Get(key interface{}) interface{}
	Remove(key interface{}, version uint64) bool
}
