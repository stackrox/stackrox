package idmap

import "github.com/stackrox/stackrox/generated/storage"

// Storage stores information about
type Storage interface {
	OnNamespaceAdd(nss ...*storage.NamespaceMetadata)
	OnNamespaceRemove(nsIDs ...string)

	// Get returns the current ID map. The result is safe to use for an arbitrary period of time, without
	// further locking.
	Get() *IDMap
}
