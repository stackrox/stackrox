package carotation

import (
	"testing"
	"time"

	"github.com/stackrox/rox/operator/internal/utils/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_DetermineAction(t *testing.T) {
	cases := map[string]struct {
		now                string
		primaryNotBefore   string
		primaryNotAfter    string
		secondaryNotBefore string
		secondaryNotAfter  string
		wantAction         Action
	}{
		"should return no action in first 3/5 of validity": {
			now:              "2026-06-01T00:00:00Z",
			primaryNotBefore: "2025-01-01T00:00:00Z",
			primaryNotAfter:  "2030-01-01T00:00:00Z",
			wantAction:       NoAction,
		},
		"should return no action just before 3/5 of validity": {
			now:              "2027-12-31T23:59:59Z",
			primaryNotBefore: "2025-01-01T00:00:00Z",
			primaryNotAfter:  "2030-01-01T00:00:00Z",
			wantAction:       NoAction,
		},
		"should add secondary after 3/5 of validity": {
			now:              "2028-01-02T00:00:00Z",
			primaryNotBefore: "2025-01-01T00:00:00Z",
			primaryNotAfter:  "2030-01-01T00:00:00Z",
			wantAction:       AddSecondary,
		},
		"should add secondary just before 4/5 of validity": {
			now:              "2028-12-31T23:59:59Z",
			primaryNotBefore: "2025-01-01T00:00:00Z",
			primaryNotAfter:  "2030-01-01T00:00:00Z",
			wantAction:       AddSecondary,
		},
		"should promote secondary after 4/5 of validity": {
			now:                "2029-01-02T00:00:00Z",
			primaryNotBefore:   "2025-01-01T00:00:00Z",
			primaryNotAfter:    "2030-01-01T00:00:00Z",
			secondaryNotBefore: "2028-01-01T00:00:00Z",
			secondaryNotAfter:  "2033-01-01T00:00:00Z",
			wantAction:         PromoteSecondary,
		},
		"should delete expired secondary": {
			now:                "2031-01-02T00:00:00Z",
			primaryNotBefore:   "2028-01-01T00:00:00Z",
			primaryNotAfter:    "2033-01-01T00:00:00Z",
			secondaryNotBefore: "2025-01-01T00:00:00Z",
			secondaryNotAfter:  "2030-01-01T00:00:00Z",
			wantAction:         DeleteSecondary,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			now, err := time.Parse(time.RFC3339, c.now)
			require.NoError(t, err)

			primary := testutils.GenerateTestCertWithValidity(t, c.primaryNotBefore, c.primaryNotAfter)
			secondary := testutils.GenerateTestCertWithValidity(t, c.secondaryNotBefore, c.secondaryNotAfter)

			action := DetermineAction(primary, secondary, now)
			assert.Equal(t, c.wantAction, action)
		})
	}
}
