package detection

import "github.com/stackrox/rox/generated/storage"

// FilterOption will be called for each policy to be applied.  A FilterOption should return true if a policy should be
// run and false if a policy should not be run.
type FilterOption func(*storage.Policy) bool
