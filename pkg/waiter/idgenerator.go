package waiter

import "github.com/stackrox/rox/pkg/uuid"

// IDGenerator generates unique IDs.
type IDGenerator interface {
	// GenID should return a string for uniquely
	// identifying a request. Will be called multiple
	// times if there are collisions.
	GenID() (string, error)
}

// UUIDGenerator generates Version 4 UUIDs.
type UUIDGenerator struct{}

var _ IDGenerator = (*UUIDGenerator)(nil)

// GenID generates a new Version 4 UUIDs on each invocation.
func (g *UUIDGenerator) GenID() (string, error) {
	return uuid.NewV4().String(), nil
}
