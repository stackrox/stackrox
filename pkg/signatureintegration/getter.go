package signatureintegration

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
)

// Getter is a subset of the signature integration datastore, which is used by
// the image datastore to query signature integration for their name.
// The getter allows us to enrich the signature verification results with a user readable
// integration name when images are queried. The goal is to avoid the data denormalization
// that would otherwise occur if we were to set the name upon verification time. This is
// because integration names may change over time, but ids are immutable.
type Getter interface {
	GetSignatureIntegration(ctx context.Context, id string) (*storage.SignatureIntegration, bool, error)
}

type GetterFunc func() Getter

// GetVerifierName returns the signature integration name for a verification result.
func GetVerifierName(ctx context.Context, getter Getter, result *storage.ImageSignatureVerificationResult) (string, error) {
	verifierID := result.GetVerifierId()
	if verifierID == "" {
		return "", errors.New("empty verifier ID")
	}
	verifier, found, err := getter.GetSignatureIntegration(ctx, verifierID)
	if !found {
		return "", nil
	}
	if err != nil {
		return "", errors.Wrap(err, "getting signature integration")
	}

	return verifier.GetName(), nil
}
