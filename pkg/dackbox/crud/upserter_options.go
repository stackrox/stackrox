package crud

import "github.com/stackrox/rox/pkg/features"

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

// WithUpserterCache adds the passed cache to the upserter
func WithUpserterCache(c *Cache) UpserterOption {
	return func(rc *upserterImpl) {
		rc.cache = c
	}
}

// AddToIndexIfFeatureEnabled indexes the object after insert if provided feature flag is enabled. It operates lazily, so things may not be in the index right away.
func AddToIndexIfFeatureEnabled(featureFlag features.FeatureFlag) UpserterOption {
	if !featureFlag.Enabled() {
		return func(rc *upserterImpl) {}
	}
	return func(rc *upserterImpl) {
		rc.addToIndex = true
	}
}
