package fieldmap

import (
	"reflect"
	"strings"

	"github.com/stackrox/rox/pkg/protowalk"
)

// MapSearchTagsToFieldPaths creates a FieldMap, by walking the given object.
func MapSearchTagsToFieldPaths(toWalk interface{}) FieldMap {
	fieldMap := make(FieldMap)
	protowalk.WalkProto(reflect.TypeOf(toWalk), func(fp protowalk.FieldPath) bool {
		// Current field is the last field in the path.
		currentField := fp.Field()

		// Get the search tags for the field.
		searchTag := getSearchTagForField(currentField.StructField)
		if searchTag == "-" {
			return false
		}
		if searchTag != "" {
			fieldMap[strings.ToLower(searchTag)] = fp.StructFields()
		}
		return true
	})
	return fieldMap
}

func getSearchTagForField(field reflect.StructField) string {
	searchTags := strings.Split(field.Tag.Get("search"), ",")
	if len(searchTags) == 0 {
		return ""
	}
	return searchTags[0]
}
