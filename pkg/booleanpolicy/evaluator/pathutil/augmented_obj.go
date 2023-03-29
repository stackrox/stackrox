package pathutil

import (
	"reflect"

	"github.com/pkg/errors"
)

// An augmentTree is a utility class used by AugmentedObj to efficiently store and retrieve
// augmented values that are added to the object by Path.
type augmentTree struct {
	value    *reflect.Value
	children map[Step]*augmentTree
}

func (t *augmentTree) takeStep(key Step) *augmentTree {
	if t == nil {
		return nil
	}
	return t.children[key]
}

func (t *augmentTree) getValue() *reflect.Value {
	if t == nil {
		return nil
	}
	return t.value
}

func addAugmentedObjToTreeAtPath(rootTree *augmentTree, path *Path, subObj *AugmentedObj) error {
	currentTree := rootTree
	for _, step := range path.steps {
		if currentTree.children == nil {
			currentTree.children = make(map[Step]*augmentTree)
		}
		subTree := currentTree.children[step]
		if subTree == nil {
			subTree = &augmentTree{}
			currentTree.children[step] = subTree
		}
		currentTree = subTree
	}

	if currentTree.children != nil {
		return errors.Errorf("cannot add subObj %v to tree %v: children exist at this path", subObj, currentTree.children)
	}
	currentTree.value = subObj.augmentTreeRoot.value
	currentTree.children = subObj.augmentTreeRoot.children
	return nil
}

// An AugmentedObj represents an object with some augments.
// Concretely, this means that it effectively consists of two parts:
// -> the core object itself
// -> a mapping of Paths to other (possibly augmented) objects.
// For example, given a struct like
//
//	type A struct {
//	   IntVal int
//	}
//
// and an object like A{IntVal: 1},
// you could augment it with "StringVal": "string".
// This makes it possible to treat the Augmented object _as if_
// it was A{IntVal: 1, StringVal: "string"}.
// This is a simple example -- it's possible to augment a value at an
// arbitrary path, traversing struct fields and slice indices, with an
// arbitrary object (which may, in turn, be an augmented object itself).
// It is a concrete realization of the object hierarchy described
// in an AugmentedObjMeta.
// Callers must use NewAugmentedObj to create one.
type AugmentedObj struct {
	augmentTreeRoot augmentTree
}

// NewAugmentedObj returns a ready-to-use instance of AugmentedObj, where the core
// object is the passed object.
// Callers can then call the AddObjAt methods to add augmented objects at various
// paths within this object.
func NewAugmentedObj(actualObj interface{}) *AugmentedObj {
	value := reflect.ValueOf(actualObj)
	return &AugmentedObj{augmentTreeRoot: augmentTree{value: &value}}
}

// AddAugmentedObjAt augments this object with the passed subObj, at the given path.
func (o *AugmentedObj) AddAugmentedObjAt(subObj *AugmentedObj, steps ...Step) error {
	return addAugmentedObjToTreeAtPath(&o.augmentTreeRoot, NewPath(steps...), subObj)
}

// AddPlainObjAt is a convenience wrapper around AddAugmentedObjAt for sub-objects
// that are not augmented.
func (o *AugmentedObj) AddPlainObjAt(subObj interface{}, steps ...Step) error {
	return o.AddAugmentedObjAt(NewAugmentedObj(subObj), steps...)
}

// Value returns an AugmentedValue, which starts off at the "root" of the augmented object.
func (o *AugmentedObj) Value() AugmentedValue {
	return &augmentedValue{underlying: *o.augmentTreeRoot.value, currentNode: &o.augmentTreeRoot}
}

// An AugmentedValue is a wrapper around a reflect.Value which can be traversed in a way
// that is augmentation-aware. It also keeps an internal record of the path traversed so far.
type AugmentedValue interface {
	Underlying() reflect.Value
	TakeStep(step MetaStep) (AugmentedValue, bool)
	// Elem is like calling .Elem on the underlying reflect.Value.
	// It panics if Elem() on the reflect.Value panics.
	Elem() AugmentedValue
	// Index is like calling .Index on the underlying reflect.Value.
	// It panics if Index(i) on the reflect.Value panics.
	Index(int) AugmentedValue
	PathFromRoot() *Path
}

type augmentedValue struct {
	parent       *augmentedValue
	edgeToParent Step
	depth        int

	currentNode *augmentTree
	underlying  reflect.Value
}

func (v *augmentedValue) Elem() AugmentedValue {
	return &augmentedValue{underlying: v.underlying.Elem(), currentNode: v.currentNode, parent: v.parent, edgeToParent: v.edgeToParent, depth: v.depth}
}

func (v *augmentedValue) Index(i int) AugmentedValue {
	step := IndexStep(i)
	return v.childValue(v.underlying.Index(i), v.currentNode.takeStep(step), step)
}

func (v *augmentedValue) Underlying() reflect.Value {
	return v.underlying
}

func (v *augmentedValue) TakeStep(metaStep MetaStep) (AugmentedValue, bool) {
	var newUnderlying reflect.Value
	var found bool

	step := FieldStep(metaStep.FieldName)
	nextNode := v.currentNode.takeStep(step)
	if metaStep.StructFieldIndex != nil {
		// This is a "static" struct -- traverse it directly.
		newUnderlying = v.underlying.FieldByIndex(metaStep.StructFieldIndex)
		found = true
	} else {
		// See if this is an augmented path.
		if value := nextNode.getValue(); value != nil {
			newUnderlying = *value
			found = true
		} else {
			// This specific case is hit when the field in the struct is an interface type,
			// in which case StructFieldIndex will not be present.
			if v.underlying.Kind() == reflect.Struct {
				newUnderlying = v.underlying.FieldByName(metaStep.FieldName)
				if newUnderlying.IsValid() {
					found = true
				}
			}
		}
	}
	if !found {
		return nil, false
	}
	return v.childValue(newUnderlying, nextNode, step), true
}

func (v *augmentedValue) childValue(newUnderlying reflect.Value, nextNode *augmentTree, edge Step) *augmentedValue {
	return &augmentedValue{
		parent:       v,
		edgeToParent: edge,
		depth:        v.depth + 1,
		underlying:   newUnderlying,
		currentNode:  nextNode,
	}
}

func (v *augmentedValue) PathFromRoot() *Path {
	p := &Path{steps: make([]Step, v.depth)}
	v.populateIntoSteps(&p.steps)
	return p
}

func (v *augmentedValue) populateIntoSteps(outSlice *[]Step) {
	if v.depth == 0 {
		return
	}
	(*outSlice)[v.depth-1] = v.edgeToParent
	v.parent.populateIntoSteps(outSlice)
}
