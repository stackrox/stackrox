package virtualmachine

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDisplayGuestOS(t *testing.T) {
	tests := map[string]struct {
		facts    map[string]string
		fallback string
		want     string
	}{
		"prefers detected guest OS": {
			facts: map[string]string{
				DetectedGuestOSKey: "Red Hat Enterprise Linux 9.2",
				GuestOSKey:         "Red Hat Enterprise Linux",
			},
			fallback: "Red Hat Enterprise Linux",
			want:     "Red Hat Enterprise Linux 9.2",
		},
		"falls back when detected is absent": {
			facts:    map[string]string{GuestOSKey: "Red Hat Enterprise Linux"},
			fallback: "Red Hat Enterprise Linux",
			want:     "Red Hat Enterprise Linux",
		},
		"falls back when detected is empty": {
			facts:    map[string]string{DetectedGuestOSKey: ""},
			fallback: "Red Hat Enterprise Linux",
			want:     "Red Hat Enterprise Linux",
		},
		"nil facts use fallback": {
			fallback: "unknown",
			want:     "unknown",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.want, DisplayGuestOS(tt.facts, tt.fallback))
		})
	}
}
