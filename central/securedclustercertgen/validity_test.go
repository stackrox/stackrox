package securedclustercertgen

import (
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/durationpb"
)

func TestClampRequestedValidity(t *testing.T) {
	cases := map[string]struct {
		requested *durationpb.Duration
		want      time.Duration
		wantErr   string
	}{
		"nil": {
			requested: nil,
			want:      0,
		},
		"zero": {
			requested: durationpb.New(0),
			want:      0,
		},
		"45 days": {
			requested: durationpb.New(45 * 24 * time.Hour),
			want:      45 * 24 * time.Hour,
		},
		"above max clamped": {
			requested: durationpb.New(2 * 365 * 24 * time.Hour),
			want:      mtls.CertLifetime(),
		},
		"below minimum": {
			requested: durationpb.New(time.Minute),
			wantErr:   "below the minimum",
		},
		"negative": {
			requested: durationpb.New(-time.Hour),
			wantErr:   "must not be negative",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := ClampRequestedValidity(tc.requested)
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}
