package search

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/stretchr/testify/mock"
)

// MockSearcher is a mock implementation of the Searcher interface.
type MockSearcher struct {
	mock.Mock
}

// SearchSecrets is a mock implementation of SearchSecrets.
func (m *MockSearcher) SearchSecrets(rawQuery *v1.RawQuery) ([]*v1.SearchResult, error) {
	args := m.Called(rawQuery)
	return args.Get(0).([]*v1.SearchResult), args.Error(1)
}

// SearchRawSecrets is a mock implementation of SearchRawSecrets.
func (m *MockSearcher) SearchRawSecrets(rawQuery *v1.RawQuery) ([]*v1.SecretAndRelationship, error) {
	args := m.Called(rawQuery)
	return args.Get(0).([]*v1.SecretAndRelationship), args.Error(1)
}
