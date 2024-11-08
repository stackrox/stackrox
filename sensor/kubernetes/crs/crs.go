package crs

import (
	"github.com/pkg/errors"
)

// EnsureClusterRegistered initiates the CRS based cluster registration flow in case a
// CRS is found instead of regular service certificate.
func EnsureClusterRegistered() error {
	return errors.New("CRS is not currently implemented")
}
