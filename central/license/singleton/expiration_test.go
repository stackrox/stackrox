package singleton

import (
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/license/validator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	minUntilExpiration = (366 + 21) * 24 * time.Hour // 366 days (~1 year) + 21 days sprint-duration
)

func TestActiveProdKeyExpiration(t *testing.T) {
	require.NotNil(t, activeProdKey, "there is no active production license key")

	var activeProdKeyRestrictions *validator.SigningKeyRestrictions

	for _, reg := range validatorRegistrations {
		if reg.keyAndAlgo.Equal(activeProdKey) {
			restrs := reg.restrictionsFunc()
			activeProdKeyRestrictions = &restrs
			break
		}
	}

	require.NotNil(t, activeProdKeyRestrictions, "failed to determine active prod license key")
	assert.Truef(t, time.Until(activeProdKeyRestrictions.LatestNotValidBefore) > minUntilExpiration, "expiration time is less than %v in the future; active key needs to be rotated", minUntilExpiration)
}
