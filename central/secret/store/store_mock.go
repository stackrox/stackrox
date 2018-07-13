package store

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/stretchr/testify/mock"
)

// MockStore is a mock implementation of the Store interface.
type MockStore struct {
	mock.Mock
}

// GetAllSecrets is a mock implementation of GetAllSecrets.
func (m *MockStore) GetAllSecrets() ([]*v1.Secret, error) {
	args := m.Called()
	return args.Get(0).([]*v1.Secret), args.Error(1)
}

// GetSecret is a mock implementation of GetSecret.
func (m *MockStore) GetSecret(id string) (secret *v1.Secret, exists bool, err error) {
	args := m.Called(id)
	return args.Get(0).(*v1.Secret), args.Bool(1), args.Error(2)
}

// GetSecretsBatch is a mock implementation of GetSecretsBatch.
func (m *MockStore) GetSecretsBatch(ids []string) ([]*v1.Secret, error) {
	args := m.Called(ids)
	return args.Get(0).([]*v1.Secret), args.Error(1)
}

// UpsertSecret is a mock implementation of UpsertSecret.
func (m *MockStore) UpsertSecret(secret *v1.Secret) error {
	args := m.Called(secret)
	return args.Error(0)
}

// GetRelationship is a mock implementation of GetRelationship.
func (m *MockStore) GetRelationship(id string) (relationships *v1.SecretRelationship, exists bool, err error) {
	args := m.Called(id)
	return args.Get(0).(*v1.SecretRelationship), args.Bool(1), args.Error(2)
}

// GetRelationshipBatch is a mock implementation of GetRelationshipBatch.
func (m *MockStore) GetRelationshipBatch(ids []string) ([]*v1.SecretRelationship, error) {
	args := m.Called(ids)
	return args.Get(0).([]*v1.SecretRelationship), args.Error(1)
}

// UpsertRelationship is a mock implementation of UpsertRelationship.
func (m *MockStore) UpsertRelationship(relationship *v1.SecretRelationship) error {
	args := m.Called(relationship)
	return args.Error(0)
}
