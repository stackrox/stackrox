package fake

import (
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	kubeVirtV1 "kubevirt.io/api/core/v1"
)

func TestValidateVMWorkload(t *testing.T) {
	tests := map[string]struct {
		input                    VirtualMachineWorkload
		wantLifecycleDuration    time.Duration
		wantUpdateInterval       time.Duration
		expectLifecycleDefaulted bool
		expectUpdateIntervalDef  bool
	}{
		"disabled workload (poolSize=0) should skip validation": {
			input: VirtualMachineWorkload{
				PoolSize:          0,
				LifecycleDuration: 0,
				UpdateInterval:    0,
			},
			wantLifecycleDuration: 0, // stays 0, not defaulted
			wantUpdateInterval:    0, // stays 0, not defaulted
		},
		"enabled workload with missing durations should apply defaults": {
			input: VirtualMachineWorkload{
				PoolSize:          10,
				LifecycleDuration: 0,
				UpdateInterval:    0,
			},
			wantLifecycleDuration:    defaultVMLifecycleDuration,
			wantUpdateInterval:       defaultVMUpdateInterval,
			expectLifecycleDefaulted: true,
			expectUpdateIntervalDef:  true,
		},
		"enabled workload with valid durations should keep them": {
			input: VirtualMachineWorkload{
				PoolSize:          5,
				LifecycleDuration: 2 * time.Minute,
				UpdateInterval:    30 * time.Second,
			},
			wantLifecycleDuration: 2 * time.Minute,
			wantUpdateInterval:    30 * time.Second,
		},
		"enabled workload with negative durations should apply defaults": {
			input: VirtualMachineWorkload{
				PoolSize:          5,
				LifecycleDuration: -1 * time.Second,
				UpdateInterval:    -1 * time.Second,
			},
			wantLifecycleDuration:    defaultVMLifecycleDuration,
			wantUpdateInterval:       defaultVMUpdateInterval,
			expectLifecycleDefaulted: true,
			expectUpdateIntervalDef:  true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := validateVMWorkload(tt.input)

			assert.Equal(t, tt.wantLifecycleDuration, result.LifecycleDuration, "lifecycleDuration mismatch")
			assert.Equal(t, tt.wantUpdateInterval, result.UpdateInterval, "updateInterval mismatch")
			// PoolSize should never change
			assert.Equal(t, tt.input.PoolSize, result.PoolSize, "poolSize should not change")
		})
	}
}

func TestNewVMTemplates(t *testing.T) {
	tests := map[string]struct {
		poolSize     int
		guestOSPool  []string
		vsockBaseCID uint32
	}{
		"single template": {
			poolSize:     1,
			guestOSPool:  []string{"linux"},
			vsockBaseCID: 100,
		},
		"multiple templates": {
			poolSize:     5,
			guestOSPool:  []string{"linux", "windows", "fedora"},
			vsockBaseCID: 1000,
		},
		"large pool": {
			poolSize:     100,
			guestOSPool:  []string{"rhel"},
			vsockBaseCID: 3,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			templates := newVMTemplates(tt.poolSize, tt.guestOSPool, tt.vsockBaseCID)

			require.Len(t, templates, tt.poolSize, "wrong number of templates")

			for i, tmpl := range templates {
				assert.Equal(t, i, tmpl.index, "template index mismatch")
				assert.Equal(t, fmt.Sprintf("vm-%d", i), tmpl.baseName, "baseName mismatch")
				assert.Equal(t, "default", tmpl.baseNamespace, "baseNamespace mismatch")
				assert.Equal(t, tt.vsockBaseCID+uint32(i), tmpl.vsockCID, "vsockCID mismatch")
				assert.Contains(t, tt.guestOSPool, tmpl.guestOS, "guestOS not from pool")
			}

			// Verify vsockCIDs are unique
			seenCIDs := make(map[uint32]bool)
			for _, tmpl := range templates {
				assert.False(t, seenCIDs[tmpl.vsockCID], "duplicate vsockCID: %d", tmpl.vsockCID)
				seenCIDs[tmpl.vsockCID] = true
			}
		})
	}
}

func TestVMTemplateInstantiate(t *testing.T) {
	template := &vmTemplate{
		index:         42,
		baseName:      "test-vm",
		baseNamespace: "test-ns",
		vsockCID:      12345,
		guestOS:       "Red Hat Enterprise Linux",
	}

	tests := map[string]struct {
		iteration   int
		wantVMName  string
		wantVMIName string
		wantVMUID   types.UID
		wantVMIUID  types.UID
	}{
		"first iteration": {
			iteration:   0,
			wantVMName:  "test-vm-0",
			wantVMIName: "test-vm-0-vmi",
			wantVMUID:   types.UID("00000000-0000-4000-8000-000000000042"),
			wantVMIUID:  types.UID("00000000-0000-4000-9000-000042000000"),
		},
		"second iteration": {
			iteration:   1,
			wantVMName:  "test-vm-1",
			wantVMIName: "test-vm-1-vmi",
			wantVMUID:   types.UID("00000000-0000-4000-8000-000000000042"), // same VM UID (based on index)
			wantVMIUID:  types.UID("00000000-0000-4000-9000-000042000001"),
		},
		"high iteration": {
			iteration:   999,
			wantVMName:  "test-vm-999",
			wantVMIName: "test-vm-999-vmi",
			wantVMUID:   types.UID("00000000-0000-4000-8000-000000000042"),
			wantVMIUID:  types.UID("00000000-0000-4000-9000-000042000999"),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			vm, vmi := template.instantiate(tt.iteration)

			// Verify VM properties
			assert.Equal(t, tt.wantVMName, vm.GetName(), "VM name mismatch")
			assert.Equal(t, "test-ns", vm.GetNamespace(), "VM namespace mismatch")
			assert.Equal(t, tt.wantVMUID, vm.GetUID(), "VM UID mismatch")
			assert.Equal(t, "VirtualMachine", vm.GetKind(), "VM kind mismatch")
			assert.Equal(t, "kubevirt.io/v1", vm.GetAPIVersion(), "VM apiVersion mismatch")

			// Verify VMI properties
			assert.Equal(t, tt.wantVMIName, vmi.GetName(), "VMI name mismatch")
			assert.Equal(t, "test-ns", vmi.GetNamespace(), "VMI namespace mismatch")
			assert.Equal(t, tt.wantVMIUID, vmi.GetUID(), "VMI UID mismatch")
			assert.Equal(t, "VirtualMachineInstance", vmi.GetKind(), "VMI kind mismatch")

			// Verify VMI owner reference points to VM
			ownerRefs := vmi.GetOwnerReferences()
			require.Len(t, ownerRefs, 1, "expected exactly one owner reference")
			assert.Equal(t, tt.wantVMUID, ownerRefs[0].UID, "owner reference UID mismatch")
			assert.Equal(t, tt.wantVMName, ownerRefs[0].Name, "owner reference name mismatch")
			assert.Equal(t, "VirtualMachine", ownerRefs[0].Kind, "owner reference kind mismatch")

			// Verify VMI has vsockCID
			vsockCID, found, err := unstructured.NestedInt64(vmi.Object, "status", "vsockCID")
			require.NoError(t, err)
			require.True(t, found, "vsockCID not found in VMI status")
			assert.Equal(t, int64(12345), vsockCID, "vsockCID mismatch")

			// Verify VMI has guestOS
			guestOS, found, err := unstructured.NestedString(vmi.Object, "status", "guestOSInfo", "name")
			require.NoError(t, err)
			require.True(t, found, "guestOSInfo.name not found")
			assert.Equal(t, "Red Hat Enterprise Linux", guestOS)
		})
	}
}

func TestGenerateFakeIndexReport(t *testing.T) {
	gen := newReportGenerator(10, 3) // 10 packages, 3 repos

	tests := map[string]struct {
		vsockCID uint32
		vmID     string
	}{
		"basic report": {
			vsockCID: 1234,
			vmID:     "test-vm-id-1",
		},
		"different VM": {
			vsockCID: 9999,
			vmID:     "00000000-0000-4000-8000-000000000042",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			report := generateFakeIndexReport(gen, tt.vsockCID, tt.vmID)

			// Verify vsockCID is set as string
			assert.Equal(t, fmt.Sprintf("%d", tt.vsockCID), report.VsockCid, "vsockCID mismatch")

			// Verify index report structure
			require.NotNil(t, report.IndexV4, "IndexV4 should not be nil")
			assert.Equal(t, fmt.Sprintf("hash-%s", tt.vmID), report.IndexV4.HashId, "HashId mismatch")
			assert.Equal(t, "IndexFinished", report.IndexV4.State, "State mismatch")
			assert.True(t, report.IndexV4.Success, "Success should be true")

			// Verify contents
			require.NotNil(t, report.IndexV4.Contents, "Contents should not be nil")
			assert.Len(t, report.IndexV4.Contents.Packages, 10, "expected 10 packages")
			assert.Len(t, report.IndexV4.Contents.Repositories, 3, "expected 3 repositories")

			// Verify packages have valid CPEs (regression test for WFN error)
			for _, pkg := range report.IndexV4.Contents.Packages {
				assert.NotEmpty(t, pkg.Cpe, "package CPE should not be empty")
				assert.Contains(t, pkg.Cpe, "cpe:2.3:", "package CPE should be valid format")
				if pkg.Source != nil {
					assert.NotEmpty(t, pkg.Source.Cpe, "source package CPE should not be empty")
				}
			}
		})
	}
}

func TestGenerateFakeIndexReport_TemplateRotation(t *testing.T) {
	gen := newReportGenerator(5, 2)

	// Generate multiple reports and verify template rotation
	var hashIDs []string
	numReports := precomputedReportVariants * 2

	for i := 0; i < numReports; i++ {
		report := generateFakeIndexReport(gen, uint32(i), fmt.Sprintf("vm-%d", i))
		// Extract a package ID to identify which template was used
		for pkgID := range report.IndexV4.Contents.Packages {
			hashIDs = append(hashIDs, pkgID)
			break
		}
	}

	// Verify we see multiple different templates (at least 2 different ones)
	uniqueTemplates := make(map[string]bool)
	for _, id := range hashIDs {
		// Extract template variant from package ID (format: "pkg-template-{variant}-{index}")
		uniqueTemplates[id[:len("pkg-template-X")]] = true
	}
	assert.GreaterOrEqual(t, len(uniqueTemplates), 2, "expected multiple templates to be used")
}

func TestJitteredInterval(t *testing.T) {
	tests := map[string]struct {
		interval      time.Duration
		jitterPercent float64
	}{
		"60s with 5% jitter": {
			interval:      60 * time.Second,
			jitterPercent: 0.05,
		},
		"1m with 20% jitter": {
			interval:      time.Minute,
			jitterPercent: 0.20,
		},
		"100ms with 10% jitter": {
			interval:      100 * time.Millisecond,
			jitterPercent: 0.10,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			minExpected := time.Duration(float64(tt.interval) * (1 - tt.jitterPercent))
			maxExpected := time.Duration(float64(tt.interval) * (1 + tt.jitterPercent))

			// Run multiple times to verify randomness stays within bounds
			for i := 0; i < 100; i++ {
				result := jitteredInterval(tt.interval, tt.jitterPercent)
				assert.GreaterOrEqual(t, result, minExpected, "jittered interval below minimum")
				assert.LessOrEqual(t, result, maxExpected, "jittered interval above maximum")
			}
		})
	}
}

func TestJitteredInterval_ZeroJitter(t *testing.T) {
	interval := 60 * time.Second
	result := jitteredInterval(interval, 0)
	assert.Equal(t, interval, result, "zero jitter should return exact interval")
}

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
			require.NoError(t, err, "unexpected error reading runtimeUser")
			require.True(t, runtimeFound, "expected runtimeUser field to be present")
			assert.Equal(t, tt.wantRuntimeUser, runtimeValue, "runtimeUser mismatch")
			if tt.expectClampedRun {
				assert.Equal(t, int64(math.MaxInt64), runtimeValue, "expected runtimeUser to be clamped to MaxInt64")
			}

			// Assert vsock CID behavior
			value, found, err := unstructured.NestedInt64(obj.Object, "status", "vsockCID")
			require.NoError(t, err, "unexpected error reading vsockCID")
			if tt.expectVSOCKCID {
				assert.True(t, found, "expected vsockCID to be present")
				assert.Equal(t, tt.wantVSOCKCID, value, "vsockCID mismatch")
			} else {
				assert.False(t, found, "did not expect vsockCID but found value %d", value)
			}
		})
	}
}

func pointerToUint32(val uint32) *uint32 {
	return &val
}
