package idmap

import "context"

// FromContext retrieves the active ID map for this context.
func FromContext(_ context.Context) *IDMap {
	// TODO: this is not actually stored in the context. However, since an ID map has a validity only for a short
	// period (i.e., single request), there is no harm in requiring a context to access it.
	// We might want to revisit if we actually do want to store this object in the context in the future..
	return StorageSingleton().Get()
}
