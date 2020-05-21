package fieldmap

import (
	"reflect"
	"strings"
)

// MapSearchTagsToFieldPaths creates a FieldMap, by walking the given object.
func MapSearchTagsToFieldPaths(toWalk interface{}) FieldMap {
	fieldMap := make(FieldMap)
	visitFields(toWalk, func(fieldPath FieldPath) bool {
		// Current field is the last field in the path.
		currentField := fieldPath[len(fieldPath)-1]

		// Get the proto tags for the field.
		protoTag, oneofTag := getProtobufTags(currentField)
		if protoTag == "" && oneofTag == "" {
			// Skip non-protobuf fields.
			return false
		}

		// Get the search tags for the field.
		searchTag := getSearchTagForField(currentField)
		if searchTag == "-" {
			return false
		}
		if searchTag != "" {
			fieldMap[strings.ToLower(searchTag)] = fieldPath
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

func getProtobufTags(field reflect.StructField) (string, string) {
	return field.Tag.Get("protobuf"), field.Tag.Get("protobuf_oneof")
}
