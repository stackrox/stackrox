package enumregistry

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestAddValues(t *testing.T) {
	tests := map[string]struct {
		path   string
		values map[string]int32
		verify func(t *testing.T)
	}{
		"basic enum values": {
			path: "test.field",
			values: map[string]int32{
				"LOW":  1,
				"HIGH": 2,
			},
			verify: func(t *testing.T) {
				assert.Equal(t, []int32{1}, Get("test.field", "low"))
				assert.Equal(t, []int32{2}, Get("test.field", "high"))
				assert.Equal(t, "LOW", Lookup("test.field", 1))
				assert.Equal(t, "HIGH", Lookup("test.field", 2))
				assert.True(t, IsEnum("test.field"))
			},
		},
		"multiple values with mixed case": {
			path: "severity.level",
			values: map[string]int32{
				"UNSET":    0,
				"CRITICAL": 1,
				"HIGH":     2,
				"MEDIUM":   3,
				"LOW":      4,
			},
			verify: func(t *testing.T) {
				assert.Equal(t, []int32{0}, Get("severity.level", "unset"))
				assert.Equal(t, []int32{1}, Get("severity.level", "critical"))
				assert.Equal(t, "UNSET", Lookup("severity.level", 0))
				assert.Equal(t, "CRITICAL", Lookup("severity.level", 1))
			},
		},
		"multiple calls to same path append values": {
			path: "status",
			values: map[string]int32{
				"PENDING": 1,
			},
			verify: func(t *testing.T) {
				// First call adds PENDING
				assert.Equal(t, []int32{1}, Get("status", "pending"))

				// Second call to same path adds more values
				AddValues("status", map[string]int32{
					"COMPLETE": 2,
				})
				assert.Equal(t, []int32{1}, Get("status", "pending"))
				assert.Equal(t, []int32{2}, Get("status", "complete"))
				assert.Equal(t, "PENDING", Lookup("status", 1))
				assert.Equal(t, "COMPLETE", Lookup("status", 2))
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Reset registry for each test
			enumMap = make(map[string]map[string]int32)
			reverseEnumMap = make(map[string]map[int32]string)

			AddValues(tc.path, tc.values)
			tc.verify(t)
		})
	}
}

func TestSnapshot(t *testing.T) {
	tests := map[string]struct {
		setup  func()
		verify func(t *testing.T, snapshot map[string]map[string]int32)
	}{
		"empty registry": {
			setup: func() {
				// Registry is already empty from init
			},
			verify: func(t *testing.T, snapshot map[string]map[string]int32) {
				assert.Empty(t, snapshot)
			},
		},
		"single path with values": {
			setup: func() {
				AddValues("test.field", map[string]int32{
					"LOW":  1,
					"HIGH": 2,
				})
			},
			verify: func(t *testing.T, snapshot map[string]map[string]int32) {
				assert.Len(t, snapshot, 1)
				assert.Contains(t, snapshot, "test.field")
				assert.Equal(t, int32(1), snapshot["test.field"]["LOW"])
				assert.Equal(t, int32(2), snapshot["test.field"]["HIGH"])
			},
		},
		"multiple paths": {
			setup: func() {
				AddValues("severity", map[string]int32{
					"LOW":  1,
					"HIGH": 2,
				})
				AddValues("status", map[string]int32{
					"PENDING":  0,
					"COMPLETE": 1,
				})
			},
			verify: func(t *testing.T, snapshot map[string]map[string]int32) {
				assert.Len(t, snapshot, 2)
				assert.Contains(t, snapshot, "severity")
				assert.Contains(t, snapshot, "status")
				assert.Equal(t, int32(1), snapshot["severity"]["LOW"])
				assert.Equal(t, int32(0), snapshot["status"]["PENDING"])
			},
		},
		"deep copy - mutations don't affect registry": {
			setup: func() {
				AddValues("test.field", map[string]int32{
					"VALUE": 1,
				})
			},
			verify: func(t *testing.T, snapshot map[string]map[string]int32) {
				// Mutate the snapshot
				snapshot["test.field"]["VALUE"] = 999
				snapshot["test.field"]["NEW"] = 2
				snapshot["new.path"] = map[string]int32{"other": 3}

				// Original registry should be unchanged
				assert.Equal(t, []int32{1}, Get("test.field", "value"))
				assert.Empty(t, Get("test.field", "new"))
				assert.False(t, IsEnum("new.path"))
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Reset registry for each test
			enumMap = make(map[string]map[string]int32)
			reverseEnumMap = make(map[string]map[int32]string)

			tc.setup()
			snapshot := Snapshot()
			tc.verify(t, snapshot)
		})
	}
}

func TestAddValuesRoundTrip(t *testing.T) {
	tests := map[string]struct {
		descriptor *descriptorpb.EnumDescriptorProto
		path       string
	}{
		"simple enum": {
			path: "test.severity",
			descriptor: &descriptorpb.EnumDescriptorProto{
				Value: []*descriptorpb.EnumValueDescriptorProto{
					{Name: proto.String("UNSET"), Number: proto.Int32(0)},
					{Name: proto.String("LOW"), Number: proto.Int32(1)},
					{Name: proto.String("MEDIUM"), Number: proto.Int32(2)},
					{Name: proto.String("HIGH"), Number: proto.Int32(3)},
					{Name: proto.String("CRITICAL"), Number: proto.Int32(4)},
				},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Reset registry
			enumMap = make(map[string]map[string]int32)
			reverseEnumMap = make(map[string]map[int32]string)

			// Add using descriptor
			Add(tc.path, tc.descriptor)

			// Capture the state
			snapshot := Snapshot()

			// Store expected Get/Lookup results
			var expectedGetResults []struct {
				query  string
				result []int32
			}
			var expectedLookupResults []struct {
				value  int32
				result string
			}

			for _, val := range tc.descriptor.GetValue() {
				expectedGetResults = append(expectedGetResults, struct {
					query  string
					result []int32
				}{
					query:  val.GetName(),
					result: Get(tc.path, val.GetName()),
				})
				expectedLookupResults = append(expectedLookupResults, struct {
					value  int32
					result string
				}{
					value:  val.GetNumber(),
					result: Lookup(tc.path, val.GetNumber()),
				})
			}

			// Clear the registry
			enumMap = make(map[string]map[string]int32)
			reverseEnumMap = make(map[string]map[int32]string)

			// Restore using AddValues with snapshot data
			for path, values := range snapshot {
				// Convert lowercased keys back to original case for AddValues
				// We need to get original case from reverseEnumMap before clearing
				// So we'll use the snapshot differently
				AddValues(path, values)
			}

			// Verify Get/Lookup produce same results
			for _, expected := range expectedGetResults {
				actual := Get(tc.path, expected.query)
				assert.Equal(t, expected.result, actual, "Get(%q, %q) should match", tc.path, expected.query)
			}

			for _, expected := range expectedLookupResults {
				actual := Lookup(tc.path, expected.value)
				assert.Equal(t, expected.result, actual, "Lookup(%q, %d) should match", tc.path, expected.value)
			}
		})
	}
}
