package crud

// ReaderOption represents an option on a created Reader.
type ReaderOption func(*readerImpl)

// WithAllocFunction created a Reader with the input alloc function for allocating a space to serialize stored bytes.
func WithAllocFunction(alloc ProtoAllocFunction) ReaderOption {
	return func(rc *readerImpl) {
		rc.allocFunc = alloc
	}
}

// WithReaderCache adds the passed cache to the reader
func WithReaderCache(c *Cache) ReaderOption {
	return func(rc *readerImpl) {
		rc.cache = c
	}
}
