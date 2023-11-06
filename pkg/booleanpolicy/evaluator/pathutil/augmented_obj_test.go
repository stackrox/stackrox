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
	mapObj := &objWithMap{AugmentedMap: map[string]string{"mapKey": "augment"}}
	o := NewAugmentedObj(topLevelObj)

	fullValue, err := o.GetFullValue()
	require.NoError(t, err)
	assert.Equal(t, map[string]interface{}{
		"A": 1,
		"Nested": []interface{}{
			map[string]interface{}{"B": "B0"},
			map[string]interface{}{"B": "B1"},
		},
	}, fullValue)

	augmentedIntObj := NewAugmentedObj(intObj)
	require.NoError(t, augmentedIntObj.AddAugmentedObjAt(NewAugmentedObj(nestedStringObj), FieldStep("StringObjWithinIntObj")))
	require.NoError(t, o.AddAugmentedObjAt(augmentedIntObj, FieldStep("IntObj")))
	require.NoError(t, o.AddPlainObjAt(stringObj, FieldStep("Nested"), IndexStep(1), FieldStep("StringObj")))
	require.NoError(t, o.AddPlainObjAt(mapObj, FieldStep("MapObj")))

	value := o.Value()
	assert.Equal(t, topLevelObj, value.Underlying().Interface())

	intObjValue, found := value.TakeStep(MetaStep{FieldName: "IntObj"})
	assert.True(t, found)
	assert.Equal(t, intObj, intObjValue.Underlying().Interface())
	_, found = value.TakeStep(MetaStep{FieldName: "Nonexistent"})
	assert.False(t, found)

	// Test the error case.
	assert.Error(t, o.AddPlainObjAt(stringObj, FieldStep("IntObj")))

	fullValue, err = o.GetFullValue()
	require.NoError(t, err)
	assert.Equal(t, map[string]interface{}{
		"A": 1,
		"Nested": []interface{}{
			map[string]interface{}{"B": "B0"},
			map[string]interface{}{
				"B":         "B1",
				"StringObj": map[string]interface{}{"AugmentedVal": "AUGMENT"},
			},
		},
		"IntObj": map[string]interface{}{
			"AugmentedVal": 3,
			"StringObjWithinIntObj": map[string]interface{}{
				"AugmentedVal": "NESTEDAUGMENT",
			},
		},
		"MapObj": map[string]interface{}{
			"AugmentedMap": map[string]interface{}{"mapKey": "augment"},
		},
	}, fullValue)
}
