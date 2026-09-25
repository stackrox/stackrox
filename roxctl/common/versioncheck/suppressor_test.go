package versioncheck

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVersionCheckerSuppressor(t *testing.T) {
	cases := map[string]struct {
		ctx                 context.Context
		expectedSuppression bool
	}{
		"background context should not suppress": {
			ctx:                 context.Background(),
			expectedSuppression: false,
		},
		"context with suppressor true should suppress": {
			ctx:                 context.WithValue(context.Background(), versionCheckerSuppressorKey{}, true),
			expectedSuppression: true,
		},
		"context with suppressor false should suppress": {
			ctx:                 context.WithValue(context.Background(), versionCheckerSuppressorKey{}, false),
			expectedSuppression: false,
		},
	}
	for tName, tCase := range cases {
		t.Run(tName, func(tt *testing.T) {
			assert.Equal(tt, tCase.expectedSuppression, ShouldSuppressVersionChecker(tCase.ctx))
		})
	}
}
