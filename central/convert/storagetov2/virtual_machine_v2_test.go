package storagetov2

import (
	"testing"
	"time"

	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
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

func TestAgentStatusFromLastContact(t *testing.T) {
	now := time.Unix(1_700_000_000, 0).UTC()
	staleAfter := 12 * time.Hour

	tests := map[string]struct {
		ts       *timestamppb.Timestamp
		expected v2.AgentStatus
	}{
		"should return unknown when never scraped": {
			ts:       nil,
			expected: v2.AgentStatus_AGENT_STATUS_UNKNOWN,
		},
		"should return unknown when timestamp is invalid": {
			ts:       &timestamppb.Timestamp{Nanos: 2_000_000_000},
			expected: v2.AgentStatus_AGENT_STATUS_UNKNOWN,
		},
		"should return active when last scrape is still inside the window": {
			ts:       timestamppb.New(now.Add(-time.Hour)),
			expected: v2.AgentStatus_AGENT_STATUS_ACTIVE,
		},
		"should return inactive when last scrape is exactly the window": {
			ts:       timestamppb.New(now.Add(-staleAfter)),
			expected: v2.AgentStatus_AGENT_STATUS_INACTIVE,
		},
		"should return inactive when last scrape is older than the window": {
			ts:       timestamppb.New(now.Add(-13 * time.Hour)),
			expected: v2.AgentStatus_AGENT_STATUS_INACTIVE,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, tc.expected, AgentStatusFromLastContact(tc.ts, now, staleAfter))
		})
	}
}
