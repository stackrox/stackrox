package clusterid

import "testing"

// NewHandlerForTesting creates a new Handler for testing
func NewHandlerForTesting(_ *testing.T) *handlerImpl {
	return newClusterID()
}
