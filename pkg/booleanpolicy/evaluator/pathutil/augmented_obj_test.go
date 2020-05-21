package pathutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAugmentedObj(t *testing.T) {
	topLevelObj := &topLevel{A: 1, Nested: []nested{{B: "B0"}, {B: "B1"}}}
	intObj := &objWithInt{AugmentedVal: 3}
	stringObj := &objWithString{AugmentedVal: "AUGMENT"}
	nestedStringObj := &objWithString{AugmentedVal: "NESTEDAUGMENT"}
	o := NewAugmentedObj(topLevelObj)
	augmentedIntObj := NewAugmentedObj(intObj)
	require.NoError(t, augmentedIntObj.AddAugmentedObjAt(PathFromSteps(t, "StringObjWithinIntObj"), NewAugmentedObj(nestedStringObj)))
	require.NoError(t, o.AddAugmentedObjAt(
		PathFromSteps(t, "IntObj"), augmentedIntObj),
	)
	require.NoError(t, o.AddPlainObjAt(
		PathFromSteps(t, "Nested", 1, "StringObj"), stringObj),
	)

	value := o.Value()
	assert.Equal(t, topLevelObj, value.Underlying().Interface())

	intObjValue, found := value.TakeStep(MetaStep{FieldName: "IntObj"})
	assert.True(t, found)
	assert.Equal(t, intObj, intObjValue.Underlying().Interface())
	_, found = value.TakeStep(MetaStep{FieldName: "Nonexistent"})
	assert.False(t, found)

	// Test the error case.
	assert.Error(t, o.AddPlainObjAt(PathFromSteps(t, "IntObj"), stringObj))
}
