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
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, Facts(tc.input))
		})
	}
}

func TestSensorEvent(t *testing.T) {
	t.Run("nil Info returns nil", func(t *testing.T) {
		assert.Nil(t, SensorEvent(central.ResourceAction_UPDATE_RESOURCE, "cluster-1", nil))
	})

	cid := uint32(7)
	event := SensorEvent(central.ResourceAction_UPDATE_RESOURCE, "cluster-1", &Info{
		ID:        "vm-1",
		Name:      "rhel9",
		Namespace: "ns",
		Running:   true,
		VSOCKCID:  &cid,
		GuestOS:   "Red Hat Enterprise Linux 9",
	})
	assert.Equal(t, "vm-1", event.GetId())
	assert.Equal(t, central.ResourceAction_UPDATE_RESOURCE, event.GetAction())
	vm := event.GetVirtualMachine()
	assert.Equal(t, "cluster-1", vm.GetClusterId())
	assert.Equal(t, int32(7), vm.GetVsockCid())
	assert.True(t, vm.GetVsockCidSet())
	assert.Equal(t, v1.VirtualMachine_RUNNING, vm.GetState())
	assert.Equal(t, map[string]string{
		pkgVM.GuestOSKey: "Red Hat Enterprise Linux 9",
	}, vm.GetFacts())
}
