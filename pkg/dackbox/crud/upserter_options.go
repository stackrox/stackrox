package crud

// UpserterOption is an option that modifies an Upserter.
type UpserterOption func(*upserterImpl)

// WithKeyFunction adds a ProtoKeyFunction to an Upserter.
func WithKeyFunction(kf ProtoKeyFunction) UpserterOption {
	return func(uc *upserterImpl) {
		uc.keyFunc = kf
	}
}

// WithPartialUpserter adds a PartialUpserter to store some portion of an input objects data separately.
func WithPartialUpserter(partial PartialUpserter) UpserterOption {
	return func(rc *upserterImpl) {
		rc.partials = append(rc.partials, partial)
	}
}

// PartialUpserterOption is an option on a PartialUpserter.
type PartialUpserterOption func(*partialUpserterImpl)

// WithSplitFunc adds a ProtoSplitFunction that separates the data that will do to the partial upsert.
func WithSplitFunc(split ProtoSplitFunction) PartialUpserterOption {
	return func(pu *partialUpserterImpl) {
		pu.splitFunc = split
	}
}

// WithUpserter says where to upsert the partial data.
func WithUpserter(upserter Upserter) PartialUpserterOption {
	return func(pu *partialUpserterImpl) {
		pu.upserter = upserter
	}
}
