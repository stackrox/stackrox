package crud

// ReaderOption represents an option on a created Reader.
type ReaderOption func(*readerImpl)

// WithAllocFunction created a Reader with the input alloc function for allocating a space to serialize stored bytes.
func WithAllocFunction(alloc ProtoAllocFunction) ReaderOption {
	return func(rc *readerImpl) {
		rc.allocFunc = alloc
	}
}

// WithPartialReader is an option that causes the read function to read in some data stored under a separate key.
func WithPartialReader(partial PartialReader) ReaderOption {
	return func(rc *readerImpl) {
		rc.partials = append(rc.partials, partial)
	}
}

// PartialReaderOption is an option on a PartialReader.
type PartialReaderOption func(impl *partialReaderImpl)

// WithMergeFunction describes how the partial reader should combine data into the higher level output object, spit out
// by the parent reader.
func WithMergeFunction(merge ProtoMergeFunction) PartialReaderOption {
	return func(pr *partialReaderImpl) {
		pr.mergeFunc = merge
	}
}

// WithMatchFunction decides which children ids are routed to the partial reader.
func WithMatchFunction(match KeyMatchFunction) PartialReaderOption {
	return func(pr *partialReaderImpl) {
		pr.matchFunc = match
	}
}

// WithReader decides where and how to read partial data.
func WithReader(reader Reader) PartialReaderOption {
	return func(pr *partialReaderImpl) {
		pr.reader = reader
	}
}
