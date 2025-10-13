package globaldb

const (
	// DefaultDataStorePoolSize is the size of the pool of mutexes that guard datastores.
	// This is a shared global constant across datastores, just for the sake of DRY.
	// Individual datastores are free to use another value.
	DefaultDataStorePoolSize = 16
)
