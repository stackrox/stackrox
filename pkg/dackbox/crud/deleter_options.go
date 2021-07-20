package crud

// DeleterOption represents an option on a created Deleter.
type DeleterOption func(*deleterImpl)

// RemoveFromIndex removes the key from index after deletion. Happens lazily, so propagation may not be immediate.
func RemoveFromIndex() DeleterOption {
	return func(rc *deleterImpl) {
		rc.removeFromIndex = true
	}
}

// Shared causes the object to only be removed if all references to it in the graph have been removed.
func Shared() DeleterOption {
	return func(rc *deleterImpl) {
		rc.shared = true
	}
}
