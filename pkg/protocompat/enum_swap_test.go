package protocompat

import (
	"testing"

	v1 "github.com/stackrox/rox/generated/test/enumswapv1"
	v2 "github.com/stackrox/rox/generated/test/enumswapv2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
)

// The protos under proto/test/enumswapv{1,2} model swapping the enum type of an existing
// repeated enum field while keeping the field name and field number. v1 uses a top-level
// ContainerType{REGULAR=0, INIT=1}; v2 uses a nested SkipContainerType{UNSET=0, INIT=1}.
//
// These tests pin down that the swap is wire-compatible: the encoding carries only the
// field number, the wire type and the numeric enum values, so blobs written by v1 are
// readable by v2 with no migration. The flip side is that compatibility is decided purely
// by the numbers, which is why REGULAR(0) silently becomes UNSET(0).

// TestEnumSwapWireLayout documents the exact bytes a repeated enum field occupies, so that
// a change to packing or field numbering shows up as a test failure rather than as
// unreadable rows.
func TestEnumSwapWireLayout(t *testing.T) {
	bytes, err := (&v1.EvaluationFilter{
		SkipContainerTypes: []v1.ContainerType{v1.ContainerType_REGULAR, v1.ContainerType_INIT},
	}).MarshalVT()
	require.NoError(t, err)

	// 0x0a = field 1, wire type 2 (length-delimited); 0x02 = payload length;
	// then the packed varints 0x00 (REGULAR) and 0x01 (INIT).
	assert.Equal(t, []byte{0x0a, 0x02, 0x00, 0x01}, bytes)
}

// TestEnumSwapProducesIdenticalBytes shows the two enum types are indistinguishable on the
// wire: the type name is never encoded, only the numeric value.
func TestEnumSwapProducesIdenticalBytes(t *testing.T) {
	before, err := (&v1.EvaluationFilter{
		SkipContainerTypes: []v1.ContainerType{v1.ContainerType_REGULAR, v1.ContainerType_INIT},
	}).MarshalVT()
	require.NoError(t, err)

	after, err := (&v2.EvaluationFilter{
		SkipContainerTypes: []v2.EvaluationFilter_SkipContainerType{v2.EvaluationFilter_UNSET, v2.EvaluationFilter_INIT},
	}).MarshalVT()
	require.NoError(t, err)

	assert.Equal(t, before, after)
}

// TestEnumSwapDecodesOldBlobs is the case that matters for already-serialized rows: encode
// with the old enum type, decode with the new one.
func TestEnumSwapDecodesOldBlobs(t *testing.T) {
	cases := map[string]struct {
		stored   []v1.ContainerType
		expected []v2.EvaluationFilter_SkipContainerType
	}{
		"empty list round-trips": {
			stored:   nil,
			expected: nil,
		},
		"INIT keeps its meaning": {
			stored:   []v1.ContainerType{v1.ContainerType_INIT},
			expected: []v2.EvaluationFilter_SkipContainerType{v2.EvaluationFilter_INIT},
		},
		"REGULAR degrades to UNSET because both are zero": {
			stored:   []v1.ContainerType{v1.ContainerType_REGULAR},
			expected: []v2.EvaluationFilter_SkipContainerType{v2.EvaluationFilter_UNSET},
		},
		"mixed list keeps order": {
			stored:   []v1.ContainerType{v1.ContainerType_REGULAR, v1.ContainerType_INIT},
			expected: []v2.EvaluationFilter_SkipContainerType{v2.EvaluationFilter_UNSET, v2.EvaluationFilter_INIT},
		},
		"duplicates are preserved": {
			stored:   []v1.ContainerType{v1.ContainerType_INIT, v1.ContainerType_INIT},
			expected: []v2.EvaluationFilter_SkipContainerType{v2.EvaluationFilter_INIT, v2.EvaluationFilter_INIT},
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			bytes, err := (&v1.EvaluationFilter{SkipContainerTypes: c.stored}).MarshalVT()
			require.NoError(t, err)

			var decoded v2.EvaluationFilter
			require.NoError(t, decoded.UnmarshalVT(bytes))
			assert.Equal(t, c.expected, decoded.GetSkipContainerTypes())
		})
	}
}

// TestEnumSwapDecodesNewBlobsWithOldType covers the downgrade direction, which matters if a
// rollback puts the previous binary back in front of rows written by the new one.
func TestEnumSwapDecodesNewBlobsWithOldType(t *testing.T) {
	bytes, err := (&v2.EvaluationFilter{
		SkipContainerTypes: []v2.EvaluationFilter_SkipContainerType{v2.EvaluationFilter_INIT},
	}).MarshalVT()
	require.NoError(t, err)

	var decoded v1.EvaluationFilter
	require.NoError(t, decoded.UnmarshalVT(bytes))
	assert.Equal(t, []v1.ContainerType{v1.ContainerType_INIT}, decoded.GetSkipContainerTypes())
}

// TestEnumSwapJSONNamesSurviveNesting guards the REST/CR contract. protojson encodes an
// enum by its value name, which is scoped to the enum itself, so nesting the enum inside a
// message changes the enum's full type name (and the generated Go identifier) but not what
// clients put on the wire. Callers keep sending "INIT", not "SkipContainerType_INIT".
func TestEnumSwapJSONNamesSurviveNesting(t *testing.T) {
	const payload = `{"skipContainerTypes":["INIT"]}`

	before, err := protojson.Marshal(&v1.EvaluationFilter{
		SkipContainerTypes: []v1.ContainerType{v1.ContainerType_INIT},
	})
	require.NoError(t, err)
	assert.JSONEq(t, payload, string(before))

	after, err := protojson.Marshal(&v2.EvaluationFilter{
		SkipContainerTypes: []v2.EvaluationFilter_SkipContainerType{v2.EvaluationFilter_INIT},
	})
	require.NoError(t, err)
	assert.JSONEq(t, payload, string(after))

	var decoded v2.EvaluationFilter
	require.NoError(t, protojson.Unmarshal([]byte(payload), &decoded))
	assert.Equal(t, []v2.EvaluationFilter_SkipContainerType{v2.EvaluationFilter_INIT}, decoded.GetSkipContainerTypes())

	// The Go-identifier spelling is not a wire name and must not be accepted, otherwise
	// we would be silently supporting a second spelling of the same value.
	assert.Error(t, protojson.Unmarshal([]byte(`{"skipContainerTypes":["SkipContainerType_INIT"]}`), &v2.EvaluationFilter{}))

	// Nesting does move the type name, which is what drives the swagger definition name.
	enum := (&v2.EvaluationFilter{}).ProtoReflect().Descriptor().Fields().Get(0).Enum()
	assert.Equal(t, "test.enumswapv2.EvaluationFilter.SkipContainerType", string(enum.FullName()))
}

// TestEnumSwapKeepsUnknownValues guards the assumption that makes the swap safe in the
// other direction too: proto3 enums are open, so a value the reader does not know about is
// retained as its raw number instead of being dropped or erroring.
func TestEnumSwapKeepsUnknownValues(t *testing.T) {
	// 0x0a 0x01 0x07 => field 1, packed, single value 7, which neither enum declares.
	var decoded v2.EvaluationFilter
	require.NoError(t, decoded.UnmarshalVT([]byte{0x0a, 0x01, 0x07}))
	assert.Equal(t, []v2.EvaluationFilter_SkipContainerType{7}, decoded.GetSkipContainerTypes())

	reencoded, err := decoded.MarshalVT()
	require.NoError(t, err)
	assert.Equal(t, []byte{0x0a, 0x01, 0x07}, reencoded)
}
