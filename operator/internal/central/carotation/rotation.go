package carotation

import (
	"crypto/x509"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/operator/internal/types"
	"github.com/stackrox/rox/pkg/certgen"
)

// Action represents the possible actions to take during CA rotation.
type Action int

const (
	// NoAction indicates that no action needs to be taken at this time.
	NoAction Action = iota
	// AddSecondary indicates that the CA has passed a threshold (e.g., 3/5 of its validity)
	// and a secondary CA should be generated and added.
	AddSecondary
	// PromoteSecondary indicates that the secondary CA should become the new primary CA,
	// typically when the primary is nearing the end of its validity (e.g., last year).
	PromoteSecondary
	// DeleteSecondary indicates that the secondary CA has expired and should be removed.
	DeleteSecondary
)

// DetermineAction selects a rotation action based on the current time, and the validity of the primary CA certificate.
// Important: The function assumes that input certificates have valid time ranges.
func DetermineAction(primary, secondary *x509.Certificate, current time.Time) Action {
	startTime := primary.NotBefore

	// HACK: add secondary CA after 5 minutes (considering the 5 minute backdating)
	addSecondaryCATime := startTime.Add(10 * time.Minute)
	if current.After(addSecondaryCATime) && secondary == nil {
		return AddSecondary
	}

	// HACK: promote secondary CA to primary after 10 minutes
	promoteSecondaryCATime := startTime.Add(15 * time.Minute)
	if current.After(promoteSecondaryCATime) {
		return PromoteSecondary
	}

	// Delete expired secondary
	if secondary != nil && current.After(secondary.NotAfter) {
		return DeleteSecondary
	}

	return NoAction
}

// Handle applies the rotation action to the given file map.
func Handle(action Action, fileMap types.SecretDataMap) error {
	switch action {
	case AddSecondary:
		ca, err := certgen.GenerateCA()
		if err != nil {
			return errors.Wrap(err, "creating secondary CA failed")
		}
		certgen.AddSecondaryCAToFileMap(fileMap, ca)

	case DeleteSecondary:
		certgen.RemoveSecondaryCA(fileMap)

	case PromoteSecondary:
		certgen.PromoteSecondaryCA(fileMap)
	}

	return nil
}
