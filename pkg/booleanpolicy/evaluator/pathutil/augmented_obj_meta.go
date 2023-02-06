package pathutil

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/stringutils"
)

// An AugmentedObjMeta represents metadata (ie, type information) about an augmented object.
// An augmented object is an object that is augmented to look like has certain extra fields that it doesn't.
// For example, if have something like
//
//	type A struct {
//	   Bs []B
//	}
//
//	type B struct {
//	   BVal string
//	}
//
//	type C struct {
//	  CVal float64
//	}
//
// Let's say you want to actually have a C inside each B.
// ie, you want to make it appear that the types are:
//
//	type A struct {
//	   Bs []B
//	}
//
//	type B struct {
//	   BVal string
//	   EmbeddedC C
//	}
//
//	type C struct {
//	  CVal float64
//	}
//
// Augmentation allows you to achieve this, by simply adding
// ("Bs.EmbeddedC", C) as an augment.
type AugmentedObjMeta struct {
	typ reflect.Type
	// The first key is the hash code of the path up to the
	// element containing the augmented object, and the second
	// key is the "name" of the augmented object.
	// For example, if we have a struct of the form:
	//  { A: { B: string, C: {D string}}
	// and we augment A.C with an E float64, so that the
	// augmented object looks like
	// { A: { B: string, C: {D string, E float64}}
	// then the keys in this map will be
	// "A.C" for the first map, and "E" for the second.
	augments map[string]map[string]*AugmentedObjMeta
}

// NewAugmentedObjMeta takes an object and creates an augmented obj.
// The object value is not important, only its type matters.
// Callers can use the AddObjectAt methods to add augments.
func NewAugmentedObjMeta(exampleObj interface{}) *AugmentedObjMeta {
	return &AugmentedObjMeta{typ: reflect.TypeOf(exampleObj)}
}

// RootType returns the root type of the underlying object (ie, the type of the base object, pre-augmentation).
func (o *AugmentedObjMeta) RootType() reflect.Type {
	return o.typ
}

// AddAugmentedObjectAt adds an augmented object at the given path.
// See the comment on the AugmentedObjMeta type for more details.
// It will typically be used in program initialization blocks.
// It returns itself for easy chaining.
// It will panic if path is empty.
func (o *AugmentedObjMeta) AddAugmentedObjectAt(path []string, childObjMeta *AugmentedObjMeta) *AugmentedObjMeta {
	if o.augments == nil {
		o.augments = make(map[string]map[string]*AugmentedObjMeta)
	}

	exceptLast, last := path[:len(path)-1], path[len(path)-1]
	mapKeyFromPath := strings.Join(exceptLast, ".")
	subMap := o.augments[mapKeyFromPath]
	if subMap == nil {
		subMap = make(map[string]*AugmentedObjMeta)
		o.augments[mapKeyFromPath] = subMap
	}

	subMap[last] = childObjMeta
	return o
}

// AddPlainObjectAt is like AddAugmentedObjectAt, but takes a plain, non-augmented child object.
// As with NewAugmentedObjMeta, the actual value of childObj does not matter, only its type does.
func (o *AugmentedObjMeta) AddPlainObjectAt(path []string, childObj interface{}) *AugmentedObjMeta {
	return o.AddAugmentedObjectAt(path, NewAugmentedObjMeta(childObj))
}

// MapSearchTagsToPaths returns a map from search tags to paths within this augmented object.
// It is NOT safe for concurrent use.
// Callers can call this after adding all the augmented objects they want.
func (o *AugmentedObjMeta) MapSearchTagsToPaths() (pathMap *FieldToMetaPathMap, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.Errorf("couldn't match search tags to path: %v", r)
		}
	}()
	pathMap = &FieldToMetaPathMap{underlying: make(map[string]metaPathAndMetadata)}
	o.doMapSearchTagsToPaths(MetaPath{}, pathMap)
	return pathMap, nil
}

func (o *AugmentedObjMeta) doMapSearchTagsToPaths(pathUntilThisObj MetaPath, outputMap *FieldToMetaPathMap) {
	seenAugmentKeys := set.NewStringSet()

	o.addPathsForSearchTags(nil, o.typ, pathUntilThisObj, MetaPath{}, outputMap, seenAugmentKeys)

	// Validate that we've actually seen all the augment paths given to us when traversing the object.
	// This is helpful to ensure that the passed paths don't match the actual paths in the object.
	// Note that this panic will not escape this package, it will be caught above and softened into an error.
	for augmentPathKey := range o.augments {
		if !seenAugmentKeys.Contains(augmentPathKey) {
			panic(fmt.Sprintf("augmented path %v/%v never encountered", pathUntilThisObj, augmentPathKey))
		}
	}
}

func (o *AugmentedObjMeta) addPathsForSearchTags(parentType, currentType reflect.Type, pathUntilThisObj, pathWithinThisObj MetaPath, outputMap *FieldToMetaPathMap, seenAugmentKeys set.StringSet) {
	switch currentType.Kind() {
	case reflect.Struct:
		o.addPathsForSearchTagsFromStruct(currentType, pathUntilThisObj, pathWithinThisObj, outputMap, seenAugmentKeys)
	case reflect.Ptr, reflect.Array, reflect.Slice:
		o.addPathsForSearchTags(currentType, currentType.Elem(), pathUntilThisObj, pathWithinThisObj, outputMap, seenAugmentKeys)
	case reflect.Interface:
		// assume that the interface type is a OneOf field, because everything else compiled from a proto will be a Ptr to a
		// concrete type.
		o.addPathsForSearchTagsFromInterface(parentType, currentType, pathUntilThisObj, pathWithinThisObj, outputMap, seenAugmentKeys)
	}
}

func (o *AugmentedObjMeta) addPathsForSearchTagsFromInterface(parentType, currentType reflect.Type, pathUntilThisObj, pathWithinThisObj MetaPath, outputMap *FieldToMetaPathMap, seenAugmentKeys set.StringSet) {
	ptrToParent := reflect.PtrTo(parentType)
	method, ok := ptrToParent.MethodByName("XXX_OneofWrappers")
	if !ok {
		panic(fmt.Sprintf("XXX_OneofWrappers should exist for all protobuf oneofs, not found for %s", parentType.Name()))
	}
	out := method.Func.Call([]reflect.Value{reflect.New(parentType)})
	actualOneOfFields := out[0].Interface().([]interface{})
	for _, f := range actualOneOfFields {
		typ := reflect.TypeOf(f)
		if typ.Implements(currentType) {
			o.addPathsForSearchTags(currentType, typ, pathUntilThisObj, pathWithinThisObj, outputMap, seenAugmentKeys)
		}
	}
}

func mapKeyFromMetaPath(path MetaPath) string {
	var out strings.Builder
	for i, step := range path {
		out.WriteString(step.FieldName)
		if i != len(path)-1 {
			out.WriteString(".")
		}
	}
	return out.String()
}

func (o *AugmentedObjMeta) addPathsForSearchTagsFromStruct(currentType reflect.Type, pathUntilThisObj, pathWithinThisObj MetaPath, outputMap *FieldToMetaPathMap, seenAugmentKeys set.StringSet) {
	// First, go over any augmented additional fields added in the struct.
	// These receive priority over any fields in the struct -- that is,
	// augments _are_ allowed to replace existing fields.
	key := mapKeyFromMetaPath(pathWithinThisObj)
	augmentedFields, augmentedFieldsFound := o.augments[key]
	if augmentedFieldsFound {
		seenAugmentKeys.Add(key)
		for augmentedFieldName, childObjMeta := range augmentedFields {
			newPath := pathWithNewStep(pathWithinThisObj, MetaStep{FieldName: augmentedFieldName, Type: childObjMeta.RootType()})
			childObjMeta.doMapSearchTagsToPaths(concatPaths(pathUntilThisObj, newPath), outputMap)
		}
	}

	// Next, go over the fields of the struct. These are the statically defined fields that exist in the struct.
	for i := 0; i < currentType.NumField(); i++ {
		field := currentType.Field(i)
		if _, inAugmented := augmentedFields[field.Name]; inAugmented {
			// Skip this field -- it has been clobbered by an augment.
			continue
		}

		// Get the search tags for the field.
		searchTag, _ := stringutils.Split2(field.Tag.Get("search"), ",")
		policyTag, shouldIgnore, shouldPreferParent, err := parsePolicyTag(field.Tag.Get("policy"))
		if err != nil {
			panic(err)
		}
		// End recursion here if it's ignored.
		if searchTag == "-" || shouldIgnore {
			continue
		}

		// Create a new path through this field.
		newPath := pathWithNewStep(pathWithinThisObj, MetaStep{FieldName: field.Name, Type: field.Type, StructFieldIndex: field.Index})
		actualTag := stringutils.OrDefault(policyTag, searchTag)
		if actualTag != "" {
			fullPath := concatPaths(pathUntilThisObj, newPath)
			if err := outputMap.add(actualTag, fullPath, shouldPreferParent); err != nil {
				// Panic here is okay, it will be caught.
				panic(err)
			}
		}

		o.addPathsForSearchTags(currentType, field.Type, pathUntilThisObj, newPath, outputMap, seenAugmentKeys)
	}
}

func parsePolicyTag(policyTag string) (tag string, shouldIgnore, shouldPreferParent bool, err error) {
	if policyTag == "" {
		return "", false, false, nil
	}
	parts := strings.Split(policyTag, ",")
	tag = parts[0]
	for _, extraPart := range parts[1:] {
		switch extraPart {
		case "ignore":
			shouldIgnore = true
		case "prefer-parent":
			shouldPreferParent = true
		default:
			return "", false, false, errors.Errorf("invalid policy tag %q: unknown field %q", policyTag, extraPart)
		}
	}
	return tag, shouldIgnore, shouldPreferParent, nil
}

// pathWithNewStep returns a new path that is comprised of the original path
// with step added to it. It does a full copy, so that the returned path
// does not use the same backing array.
func pathWithNewStep(path MetaPath, step MetaStep) MetaPath {
	out := make(MetaPath, 0, len(path)+1)
	out = append(out, path...)
	out = append(out, step)
	return out
}

func concatPaths(pathUntil, pathWithin MetaPath) MetaPath {
	fullPath := make(MetaPath, 0, len(pathUntil)+len(pathWithin))
	fullPath = append(fullPath, pathUntil...)
	fullPath = append(fullPath, pathWithin...)
	return fullPath
}
