package postgres

import "github.com/stackrox/rox/pkg/postgres"

type uncachedDB struct {
	postgres.DB
}

// WithoutStoreCache disables generic store caches for stores created with this DB.
// Other users of the underlying DB keep their existing cache policy.
// Additional DB wrappers must expose Unwrap() postgres.DB to preserve discovery
// of this policy regardless of wrapper order.
func WithoutStoreCache(db postgres.DB) postgres.DB {
	return &uncachedDB{DB: db}
}

// StoreCacheDisabled marks the database as requiring direct store access.
func (*uncachedDB) StoreCacheDisabled() bool {
	return true
}

// Unwrap preserves access to other database policies when wrappers are composed.
func (db *uncachedDB) Unwrap() postgres.DB {
	return db.DB
}

func storeCacheDisabled(db postgres.DB) bool {
	for db != nil {
		if policy, ok := db.(interface{ StoreCacheDisabled() bool }); ok && policy.StoreCacheDisabled() {
			return true
		}
		wrapper, ok := db.(interface{ Unwrap() postgres.DB })
		if !ok {
			return false
		}
		db = wrapper.Unwrap()
	}
	return false
}
