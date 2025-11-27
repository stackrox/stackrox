package fake

import (
	"math"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	kubeVirtV1 "kubevirt.io/api/core/v1"
)

func TestToUnstructuredVMI(t *testing.T) {
	tests := map[string]struct {
		vsockCID         *uint32
		expectVSOCKCID   bool
		wantVSOCKCID     int64
		runtimeUser      uint64
		wantRuntimeUser  int64
		expectClampedRun bool
	}{
		"should normalize vsock CID and runtime user": {
			vsockCID:        pointerToUint32(1234),
			expectVSOCKCID:  true,
			wantVSOCKCID:    1234,
			runtimeUser:     42,
			wantRuntimeUser: 42,
		},
		"should leave vsock CID unset when absent": {
			vsockCID:        nil,
			expectVSOCKCID:  false,
			runtimeUser:     0,
			wantRuntimeUser: 0,
		},
		"should clamp runtime user when exceeding int64": {
			vsockCID:         nil,
			expectVSOCKCID:   false,
			runtimeUser:      uint64(math.MaxInt64) + 100,
			wantRuntimeUser:  math.MaxInt64,
			expectClampedRun: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			vmi := &kubeVirtV1.VirtualMachineInstance{
				TypeMeta: metav1.TypeMeta{
					Kind:       "VirtualMachineInstance",
					APIVersion: "kubevirt.io/v1",
				},
				Status: kubeVirtV1.VirtualMachineInstanceStatus{
					VSOCKCID:    tt.vsockCID,
					RuntimeUser: tt.runtimeUser,
				},
			}

			obj := toUnstructuredVMI(vmi)

			// Assert runtime user always present and normalized to int64
			runtimeValue, runtimeFound, err := unstructured.NestedInt64(obj.Object, "status", "runtimeUser")
			if err != nil {
				t.Fatalf("unexpected error reading runtimeUser: %v", err)
			}
			if !runtimeFound {
				t.Fatalf("expected runtimeUser field to be present")
			}
			if runtimeValue != tt.wantRuntimeUser {
				t.Fatalf("expected runtimeUser %d, got %d", tt.wantRuntimeUser, runtimeValue)
			}
			if tt.expectClampedRun && runtimeValue != math.MaxInt64 {
				t.Fatalf("expected runtimeUser to be clamped to MaxInt64 but got %d", runtimeValue)
			}

			// Assert vsock CID behavior
			value, found, err := unstructured.NestedInt64(obj.Object, "status", "vsockCID")
			if err != nil {
				t.Fatalf("unexpected error reading vsockCID: %v", err)
			}
			if tt.expectVSOCKCID && (!found || value != tt.wantVSOCKCID) {
				t.Fatalf("expected vsockCID %d, found=%t value=%d", tt.wantVSOCKCID, found, value)
			}
			if !tt.expectVSOCKCID && found {
				t.Fatalf("did not expect vsockCID but found value %d", value)
			}
		})
	}
}

func pointerToUint32(val uint32) *uint32 {
	return &val
}
