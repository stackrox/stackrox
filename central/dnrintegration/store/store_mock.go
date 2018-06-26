package store

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/stretchr/testify/mock"
)

// MockStore is a mock implementation of the Store interface.
type MockStore struct {
	mock.Mock
}

// AddDNRIntegration provides a mock function with given fields: integration
func (_m *MockStore) AddDNRIntegration(integration *v1.DNRIntegration) (string, error) {
	ret := _m.Called(integration)

	var r0 string
	if rf, ok := ret.Get(0).(func(*v1.DNRIntegration) string); ok {
		r0 = rf(integration)
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*v1.DNRIntegration) error); ok {
		r1 = rf(integration)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetDNRIntegration provides a mock function with given fields: id
func (_m *MockStore) GetDNRIntegration(id string) (*v1.DNRIntegration, bool, error) {
	ret := _m.Called(id)

	var r0 *v1.DNRIntegration
	if rf, ok := ret.Get(0).(func(string) *v1.DNRIntegration); ok {
		r0 = rf(id)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1.DNRIntegration)
		}
	}

	var r1 bool
	if rf, ok := ret.Get(1).(func(string) bool); ok {
		r1 = rf(id)
	} else {
		r1 = ret.Get(1).(bool)
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(string) error); ok {
		r2 = rf(id)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// GetDNRIntegrations provides a mock function with given fields: request
func (_m *MockStore) GetDNRIntegrations(request *v1.GetDNRIntegrationsRequest) ([]*v1.DNRIntegration, error) {
	ret := _m.Called(request)

	var r0 []*v1.DNRIntegration
	if rf, ok := ret.Get(0).(func(*v1.GetDNRIntegrationsRequest) []*v1.DNRIntegration); ok {
		r0 = rf(request)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*v1.DNRIntegration)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*v1.GetDNRIntegrationsRequest) error); ok {
		r1 = rf(request)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// RemoveDNRIntegration provides a mock function with given fields: id
func (_m *MockStore) RemoveDNRIntegration(id string) error {
	ret := _m.Called(id)

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(id)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// UpdateDNRIntegration provides a mock function with given fields: integration
func (_m *MockStore) UpdateDNRIntegration(integration *v1.DNRIntegration) error {
	ret := _m.Called(integration)

	var r0 error
	if rf, ok := ret.Get(0).(func(*v1.DNRIntegration) error); ok {
		r0 = rf(integration)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
