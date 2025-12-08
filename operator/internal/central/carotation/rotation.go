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
	// AddSecondaryAndPromote indicates that the CA has passed a threshold (e.g., 4/5 of its validity)
	// and a secondary CA should be generated and added, and then promoted to primary. This is doing
	// AddSecondary and PromoteSecondary in one step.
	AddSecondaryAndPromote
	// DeleteSecondary indicates that the secondary CA has expired and should be removed.
	DeleteSecondary
)

// DetermineAction selects a rotation action based on the current time, and the validity of the primary CA certificate.
// Important: The function assumes that input certificates have valid time ranges.
func DetermineAction(primary, secondary *x509.Certificate, current time.Time) Action {
	startTime := primary.NotBefore
	validityDuration := primary.NotAfter.Sub(primary.NotBefore)
	fifthOfValidityDuration := validityDuration / 5

	// Add secondary CA after 3/5 of the primary's validity period has elapsed.
	addSecondaryCATime := startTime.Add(3 * fifthOfValidityDuration)
	// Promote secondary to primary after 4/5 of the primary's validity period has elapsed.
	promoteSecondaryCATime := startTime.Add(4 * fifthOfValidityDuration)

	if secondary == nil {
		if current.After(promoteSecondaryCATime) {
			return AddSecondaryAndPromote
		}

		if current.After(addSecondaryCATime) {
			return AddSecondary
		}

		return NoAction
	}

	if current.After(promoteSecondaryCATime) && secondary.NotBefore.After(primary.NotBefore) {
		return PromoteSecondary
	}

	// Delete expired secondary
	if current.After(secondary.NotAfter) {
		return DeleteSecondary
	}

	return NoAction
}

// Handle applies the rotation action to the given file map.
func Handle(action Action, fileMap types.SecretDataMap) error {
	switch action {
	case AddSecondary:
		return addSecondaryCA(fileMap)

	case PromoteSecondary:
		certgen.PromoteSecondaryCA(fileMap)

	case AddSecondaryAndPromote:
		if err := addSecondaryCA(fileMap); err != nil {
			return err
		}
		certgen.PromoteSecondaryCA(fileMap)

	case DeleteSecondary:
		certgen.RemoveSecondaryCA(fileMap)
	}
	return nil
}

func addSecondaryCA(fileMap types.SecretDataMap) error {
	ca, err := certgen.GenerateCA()
	if err != nil {
		return errors.Wrap(err, "creating secondary CA failed")
	}
	certgen.AddSecondaryCAToFileMap(fileMap, ca)
	return nil
}
