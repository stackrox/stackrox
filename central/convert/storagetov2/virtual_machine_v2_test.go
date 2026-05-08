package storagetov2

import (
	"testing"

	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/require"
)

func TestVirtualMachineV2ToDetail_Notes(t *testing.T) {
	tests := map[string]struct {
		note     storage.VirtualMachineV2_Note
		expected v2.VMNote
	}{
		"should preserve legacy missing scan data note": {
			note:     storage.VirtualMachineV2_MISSING_SCAN_DATA,
			expected: v2.VMNote_VM_NOTE_MISSING_SCAN_DATA,
		},
		"should map missing scanner note": {
			note:     storage.VirtualMachineV2_MISSING_SCANNER,
			expected: v2.VMNote_VM_NOTE_MISSING_SCANNER,
		},
		"should map scan failed note": {
			note:     storage.VirtualMachineV2_SCAN_FAILED,
			expected: v2.VMNote_VM_NOTE_SCAN_FAILED,
		},
		"should preserve missing signature note": {
			note:     storage.VirtualMachineV2_MISSING_SIGNATURE,
			expected: v2.VMNote_VM_NOTE_MISSING_SIGNATURE,
		},
		"should preserve missing signature verification data note": {
			note:     storage.VirtualMachineV2_MISSING_SIGNATURE_VERIFICATION_DATA,
			expected: v2.VMNote_VM_NOTE_MISSING_SIGNATURE_VERIFICATION_DATA,
		},
		"should map missing metadata note": {
			note:     storage.VirtualMachineV2_MISSING_METADATA,
			expected: v2.VMNote_VM_NOTE_MISSING_METADATA,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			detail := VirtualMachineV2ToDetail(&storage.VirtualMachineV2{
				Id:    "vm-1",
				Name:  "vm-1",
				Notes: []storage.VirtualMachineV2_Note{tc.note},
			})
			require.Equal(t, []v2.VMNote{tc.expected}, detail.GetNotes())
		})
	}
}
