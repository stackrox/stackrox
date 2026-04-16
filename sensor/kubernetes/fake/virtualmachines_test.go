package fake

import (
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/fixtures/vmindexreport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	kubeVirtV1 "kubevirt.io/api/core/v1"
)

func TestValidateVMWorkload(t *testing.T) {
	tests := map[string]struct {
		input                 VirtualMachineWorkload
		wantLifecycleDuration time.Duration
		wantUpdateInterval    time.Duration
		wantErr               string // empty string means no error expected
	}{
		"disabled workload (poolSize=0) should skip validation": {
			input: VirtualMachineWorkload{
				PoolSize:          0,
				LifecycleDuration: 0,
				UpdateInterval:    0,
			},
			wantLifecycleDuration: 0,
			wantUpdateInterval:    0,
			wantErr:               "",
		},
		"enabled workload with missing lifecycleDuration should default it": {
			input: VirtualMachineWorkload{
				PoolSize:          10,
				LifecycleDuration: 0,
				UpdateInterval:    10 * time.Second,
			},
			wantLifecycleDuration: defaultVMLifecycleDuration,
			wantUpdateInterval:    10 * time.Second,
			wantErr:               "virtualMachineWorkload.lifecycleDuration not set or <= 0; defaulting to 30m0s",
		},
		"enabled workload with missing updateInterval should default it": {
			input: VirtualMachineWorkload{
				PoolSize:          10,
				LifecycleDuration: 2 * time.Minute,
				UpdateInterval:    0,
			},
			wantLifecycleDuration: 2 * time.Minute,
			wantUpdateInterval:    defaultVMUpdateInterval,
			wantErr:               "virtualMachineWorkload.updateInterval not set or <= 0; defaulting to 3m0s",
		},
		"enabled workload with valid durations should keep them": {
			input: VirtualMachineWorkload{
				PoolSize:          5,
				LifecycleDuration: 2 * time.Minute,
				UpdateInterval:    20 * time.Second, // < lowerBound (1min), OK
			},
			wantLifecycleDuration: 2 * time.Minute,
			wantUpdateInterval:    20 * time.Second,
			wantErr:               "",
		},
		"enabled workload with negative lifecycleDuration should default it": {
			input: VirtualMachineWorkload{
				PoolSize:          5,
				LifecycleDuration: -1 * time.Second,
				UpdateInterval:    10 * time.Second,
			},
			wantLifecycleDuration: defaultVMLifecycleDuration,
			wantUpdateInterval:    10 * time.Second,
			wantErr:               "virtualMachineWorkload.lifecycleDuration not set or <= 0; defaulting to 30m0s",
		},
		"updateInterval in jitter range warns about potential missed updates": {
			input: VirtualMachineWorkload{
				PoolSize:          5,
				LifecycleDuration: 60 * time.Second, // bounds: 30s-90s
				UpdateInterval:    45 * time.Second, // in range, some VMs may miss
			},
			wantLifecycleDuration: 60 * time.Second,
			wantUpdateInterval:    45 * time.Second,
			wantErr: `The VM will live for a random duration between 30s and 1m30s. ` +
				`Setting "updateInterval"=45s may cause some VMs to never receive an update. ` +
				`Lower the value of "updateInterval" or increase the 'lifecycleDuration'.`,
		},
		"updateInterval below lower bound is OK": {
			input: VirtualMachineWorkload{
				PoolSize:          5,
				LifecycleDuration: 60 * time.Second, // bounds: 30s-90s
				UpdateInterval:    20 * time.Second, // < 30s lower bound, all VMs get updates
			},
			wantLifecycleDuration: 60 * time.Second,
			wantUpdateInterval:    20 * time.Second,
			wantErr:               "",
		},
		"updateInterval above upper bound causes none": {
			input: VirtualMachineWorkload{
				PoolSize:          5,
				LifecycleDuration: 60 * time.Second,  // bounds: 30s-90s
				UpdateInterval:    100 * time.Second, // > 90s upper bound, no VM gets updates
			},
			wantLifecycleDuration: 60 * time.Second,
			wantUpdateInterval:    100 * time.Second,
			wantErr: `The VM will live for a random duration between 30s and 1m30s. ` +
				`Setting "updateInterval"=1m40s causes none of the VMs to ever receive an update. ` +
				`Lower the value of "updateInterval" or increase the 'lifecycleDuration'.`,
		},
		"reportInterval in jitter range warns": {
			input: VirtualMachineWorkload{
				PoolSize:          5,
				LifecycleDuration: 60 * time.Second, // bounds: 30s-90s
				UpdateInterval:    20 * time.Second, // < lower bound, OK
				ReportInterval:    45 * time.Second, // in range
			},
			wantLifecycleDuration: 60 * time.Second,
			wantUpdateInterval:    20 * time.Second,
			wantErr: `The VM will live for a random duration between 30s and 1m30s. ` +
				`Setting "reportInterval"=45s may cause some VMs to never send any index reports. ` +
				`Lower the value of "reportInterval" or increase the 'lifecycleDuration'.`,
		},
		"initialReportDelay in jitter range warns": {
			input: VirtualMachineWorkload{
				PoolSize:           5,
				LifecycleDuration:  60 * time.Second, // bounds: 30s-90s
				UpdateInterval:     20 * time.Second, // < lower bound, OK
				ReportInterval:     20 * time.Second, // < lower bound, OK
				InitialReportDelay: 45 * time.Second, // in range
			},
			wantLifecycleDuration: 60 * time.Second,
			wantUpdateInterval:    20 * time.Second,
			wantErr: `The VM will live for a random duration between 30s and 1m30s. ` +
				`Setting "initialReportDelay"=45s may cause some VMs to never send any index reports. ` +
				`Lower the value of "initialReportDelay" or increase the 'lifecycleDuration'.`,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result, err := validateVMWorkload(tt.input)

			assert.Equal(t, tt.wantLifecycleDuration, result.LifecycleDuration, "lifecycleDuration mismatch")
			assert.Equal(t, tt.wantUpdateInterval, result.UpdateInterval, "updateInterval mismatch")
			assert.Equal(t, tt.input.PoolSize, result.PoolSize, "poolSize should not change")

			if tt.wantErr == "" {
				assert.NoError(t, err, "expected no error")
			} else {
				assert.EqualError(t, err, tt.wantErr)
			}
		})
	}
}

func TestGenerateFakeIndexReport(t *testing.T) {
	gen := vmindexreport.NewGeneratorWithSeed(10, 42) // 10 packages, seed=42 for reproducibility
	tests := map[string]uint32{
		"basic report": 1234,
		"different VM": 9999,
	}
	for name, vsockCID := range tests {
		t.Run(name, func(t *testing.T) {
			report := gen.GenerateV1IndexReport(vsockCID)

			// Verify vsockCID is set as string
			assert.Equal(t, fmt.Sprintf("%d", vsockCID), report.GetVsockCid(), "vsockCID mismatch")

			// Verify index report structure
			require.NotNil(t, report.GetIndexV4(), "IndexV4 should not be nil")
			assert.Equal(t, "IndexFinished", report.GetIndexV4().GetState(), "State mismatch")
			assert.True(t, report.GetIndexV4().GetSuccess(), "Success should be true")

			// Verify contents
			require.NotNil(t, report.GetIndexV4().GetContents(), "Contents should not be nil")
			assert.Len(t, report.GetIndexV4().GetContents().GetPackages(), 10, "expected 10 packages")
			assert.Len(t, report.GetIndexV4().GetContents().GetRepositories(), 3, "expected 3 real repositories")

			// Verify packages have valid CPEs (regression test for WFN error)
			for _, pkg := range report.GetIndexV4().GetContents().GetPackages() {
				assert.NotEmpty(t, pkg.GetCpe(), "package CPE should not be empty")
				assert.Contains(t, pkg.GetCpe(), "cpe:2.3:", "package CPE should be valid format")
				if pkg.GetSource() != nil {
					assert.NotEmpty(t, pkg.GetSource().GetCpe(), "source package CPE should not be empty")
				}
			}
		})
	}
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
