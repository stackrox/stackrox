package crud

// DeleterOption represents an option on a created Deleter.
type DeleterOption func(*deleterImpl)

// WithPartialDeleter adds a PartialDeleter to delete dependent keys.
func WithPartialDeleter(partial PartialDeleter) DeleterOption {
	return func(rc *deleterImpl) {
		rc.partials = append(rc.partials, partial)
	}
}

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

// PartialDeleterOption is an option on a PartialDeleter.
type PartialDeleterOption func(impl *partialDeleterImpl)

// WithDeleterMatchFunction decides which children ids are routed to the partial reader.
func WithDeleterMatchFunction(match KeyMatchFunction) PartialDeleterOption {
	return func(pr *partialDeleterImpl) {
		pr.matchFunction = match
	}
}

// WithDeleter decides which children ids are routed to the partial reader.
func WithDeleter(deleter Deleter) PartialDeleterOption {
	return func(pr *partialDeleterImpl) {
		pr.deleter = deleter
	}
}
