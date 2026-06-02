package securedclustercertgen

import (
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/mtls"
	"google.golang.org/protobuf/types/known/durationpb"
)

const (
	// minRequestedCertValidity is the absolute floor for requested certificate validity.
	minRequestedCertValidity = 5 * time.Minute
)

// ClampRequestedValidity validates and clamps the requested certificate validity duration.
// When requested is nil or zero, returns zero duration (caller should use default 1-year signing profile).
func ClampRequestedValidity(requested *durationpb.Duration) (time.Duration, error) {
	if requested == nil {
		return 0, nil
	}
	d := requested.AsDuration()
	if d == 0 {
		return 0, nil
	}
	if d < 0 {
		return 0, errors.New("requested validity must not be negative")
	}
	if d < minRequestedCertValidity {
		return 0, errors.Errorf("requested validity %s is below the minimum %s", d, minRequestedCertValidity)
	}
	maxValidity := mtls.CertLifetime()
	if d > maxValidity {
		d = maxValidity
	}
	return d, nil
}
