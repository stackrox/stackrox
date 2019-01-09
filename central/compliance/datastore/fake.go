package datastore

import (
	"fmt"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

var (
	errNotFound = fmt.Errorf("not found")
)

type fake struct{}

// DataStore is the interface for accessing stored compliance data
type DataStore interface {
	QueryControlResults(query *v1.Query) ([]*storage.ComplianceControlResult, error)
}

// Fake is a factory function for the mocked up API
func Fake() DataStore {
	return &fake{}
}

func (f *fake) QueryControlResults(query *v1.Query) ([]*storage.ComplianceControlResult, error) {
	return nil, errNotFound
}
