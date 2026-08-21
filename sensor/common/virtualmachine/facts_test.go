package virtualmachine

import (
	"testing"

	"github.com/stackrox/rox/generated/internalapi/central"
	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
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
			assert.Equal(t, tc.expected, Facts(tc.input))
		})
	}
}

func TestAgentFactsFromResponseFacts(t *testing.T) {
	cases := map[string]struct {
		input    map[string]string
		expected map[string]string
	}{
		"nil map returns nil": {
			input:    nil,
			expected: nil,
		},
		"empty map returns nil": {
			input:    map[string]string{},
			expected: nil,
		},
		"RHEL with version and statuses": {
			input: map[string]string{
				"detected_os":         v1.DetectedOS_RHEL.String(),
				"os_version":          "9.2",
				"activation_status":   v1.ActivationStatus_INACTIVE.String(),
				"dnf_metadata_status": v1.DnfMetadataStatus_UNAVAILABLE.String(),
			},
			expected: map[string]string{
				pkgVM.DetectedGuestOSKey:   "Red Hat Enterprise Linux 9.2",
				pkgVM.ActivationStatusKey:  pkgVM.ActivationStatusInactive,
				pkgVM.DNFMetadataStatusKey: pkgVM.DNFMetadataStatusUnavailable,
			},
		},
		"RHEL without version": {
			input: map[string]string{
				"detected_os": v1.DetectedOS_RHEL.String(),
			},
			expected: map[string]string{
				pkgVM.DetectedGuestOSKey: "Red Hat Enterprise Linux",
			},
		},
		"unspecified enums are omitted": {
			input: map[string]string{
				"detected_os":         v1.DetectedOS_UNKNOWN.String(),
				"activation_status":   v1.ActivationStatus_ACTIVATION_UNSPECIFIED.String(),
				"dnf_metadata_status": v1.DnfMetadataStatus_DNF_METADATA_UNSPECIFIED.String(),
			},
			expected: nil,
		},
		"active and available": {
			input: map[string]string{
				"activation_status":   v1.ActivationStatus_ACTIVE.String(),
				"dnf_metadata_status": v1.DnfMetadataStatus_AVAILABLE.String(),
			},
			expected: map[string]string{
				pkgVM.ActivationStatusKey:  pkgVM.ActivationStatusActive,
				pkgVM.DNFMetadataStatusKey: pkgVM.DNFMetadataStatusAvailable,
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, AgentFactsFromResponseFacts(tc.input))
		})
	}
}

func TestSensorEvent(t *testing.T) {
	t.Run("nil Info returns nil", func(t *testing.T) {
		assert.Nil(t, SensorEvent(central.ResourceAction_UPDATE_RESOURCE, "cluster-1", nil))
	})

	cid := uint32(7)
	event := SensorEvent(central.ResourceAction_UPDATE_RESOURCE, "cluster-1", &Info{
		ID:         "vm-1",
		Name:       "rhel9",
		Namespace:  "ns",
		Running:    true,
		VSOCKCID:   &cid,
		GuestOS:    "Red Hat Enterprise Linux 9",
		AgentFacts: map[string]string{pkgVM.ActivationStatusKey: pkgVM.ActivationStatusActive},
	})
	assert.Equal(t, "vm-1", event.GetId())
	assert.Equal(t, central.ResourceAction_UPDATE_RESOURCE, event.GetAction())
	vm := event.GetVirtualMachine()
	assert.Equal(t, "cluster-1", vm.GetClusterId())
	assert.Equal(t, int32(7), vm.GetVsockCid())
	assert.True(t, vm.GetVsockCidSet())
	assert.Equal(t, v1.VirtualMachine_RUNNING, vm.GetState())
	assert.Equal(t, map[string]string{
		pkgVM.GuestOSKey:          "Red Hat Enterprise Linux 9",
		pkgVM.ActivationStatusKey: pkgVM.ActivationStatusActive,
	}, vm.GetFacts())
}
