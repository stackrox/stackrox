package waiter

import "github.com/google/uuid"

// IDGenerator represents a component that can generate unique IDs
type IDGenerator interface {
	// GenID should return a string for uniquely
	// identifying a request. Will be called multiple
	// times if there are collisions
	GenID() (string, error)
}

// UUIDGenerator is an IDGenerator that produces Version 4 UUIDs
type UUIDGenerator struct{}

var _ IDGenerator = (*UUIDGenerator)(nil)

// GenID generates a new Version 4 UUIDs on each invocation
func (g *UUIDGenerator) GenID() (string, error) {
	uid, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}

	return uid.String(), nil
}

// IDGeneratorFuncs defines an empty implementation of IDGenerator
// (useful for testing/mocking)
type IDGeneratorFuncs struct {
	GenIDFunc func() (string, error)
}

var _ IDGenerator = (*IDGeneratorFuncs)(nil)

// GenID invokes GenIDFunc to generate an ID
func (i IDGeneratorFuncs) GenID() (string, error) {
	return i.GenIDFunc()
}
