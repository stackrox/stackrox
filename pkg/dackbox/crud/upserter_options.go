package crud

// UpserterOption is an option that modifies an Upserter.
type UpserterOption func(*upserterImpl)

// WithKeyFunction adds a ProtoKeyFunction to an Upserter.
func WithKeyFunction(kf ProtoKeyFunction) UpserterOption {
	return func(uc *upserterImpl) {
		uc.keyFunc = kf
	}
}

// AddToIndex indexes the object after insert. It operates lazily, so things may not be in the index right away.
func AddToIndex() UpserterOption {
	return func(rc *upserterImpl) {
		rc.addToIndex = true
	}
}
