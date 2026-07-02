package virtualmachine

import (
	"testing"

	pkgVM "github.com/stackrox/rox/pkg/virtualmachine"
	"github.com/stretchr/testify/assert"
)

func TestFacts(t *testing.T) {
	cases := map[string]struct {
		input    *Info
		expected map[string]string
	}{
		"nil Info returns nil": {
			input:    nil,
			expected: nil,
		},
		"empty Info returns default guestOS": {
			input: &Info{},
			expected: map[string]string{
				pkgVM.GuestOSKey: pkgVM.UnknownGuestOS,
			},
		},
		"populated scalar fields": {
			input: &Info{
				GuestOS:     "Red Hat Enterprise Linux 9",
				Description: "web server",
				NodeName:    "node-1",
			},
			expected: map[string]string{
				pkgVM.GuestOSKey:     "Red Hat Enterprise Linux 9",
				pkgVM.DescriptionKey: "web server",
				pkgVM.NodeNameKey:    "node-1",
			},
		},
		"populated slice fields are joined": {
			input: &Info{
				IPAddresses: []string{"10.0.0.1", "10.0.0.2"},
				ActivePods:  []string{"pod-a=node-1"},
				BootOrder:   []string{"disk1=1", "disk2=2"},
				CDRomDisks:  []string{"cdrom0"},
			},
			expected: map[string]string{
				pkgVM.GuestOSKey:     pkgVM.UnknownGuestOS,
				pkgVM.IPAddressesKey: "10.0.0.1, 10.0.0.2",
				pkgVM.ActivePodsKey:  "pod-a=node-1",
				pkgVM.BootOrderKey:   "disk1=1, disk2=2",
				pkgVM.CDRomDisksKey:  "cdrom0",
			},
		},
		"AgentFacts are merged into result": {
			input: &Info{
				GuestOS: "RHEL 9",
				AgentFacts: map[string]string{
					pkgVM.ActivationStatusKey: pkgVM.ActivationStatusActive,
					pkgVM.DetectedGuestOSKey:  "Red Hat Enterprise Linux 9.2",
				},
			},
			expected: map[string]string{
				pkgVM.GuestOSKey:          "RHEL 9",
				pkgVM.ActivationStatusKey: pkgVM.ActivationStatusActive,
				pkgVM.DetectedGuestOSKey:  "Red Hat Enterprise Linux 9.2",
			},
		},
		"AgentFacts can override base keys": {
			input: &Info{
				GuestOS: "RHEL 9",
				AgentFacts: map[string]string{
					pkgVM.GuestOSKey: "overridden",
				},
			},
			expected: map[string]string{
				pkgVM.GuestOSKey: "overridden",
			},
		},
		"nil AgentFacts does not affect result": {
			input: &Info{
				GuestOS:    "Fedora",
				AgentFacts: nil,
			},
			expected: map[string]string{
				pkgVM.GuestOSKey: "Fedora",
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			actual := Facts(tc.input)
			assert.Equal(t, tc.expected, actual)
		})
	}
}
