package index

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/stretchr/testify/mock"
)

// MockIndexer is a mock implementation of the Indexer interface.
type MockIndexer struct {
	mock.Mock
}

// SecretAndRelationship is a mock implementation of SecretAndRelationship.
func (m *MockIndexer) SecretAndRelationship(sar *v1.SecretAndRelationship) error {
	args := m.Called(sar)
	return args.Error(0)
}
