package predicate

import (
	"reflect"
	"strings"
)

func mapSearchTagsToFieldPaths(toWalk interface{}) map[string]FieldPath {
	fieldMap := make(map[string]FieldPath)
	VisitFields(toWalk, func(fieldPath FieldPath) {
		// Current field is the last field in the path.
		currentField := fieldPath[len(fieldPath)-1]

		// Get the proto tags for the field.
		protoTag, oneofTag := getProtobufTags(currentField)
		if protoTag == "" && oneofTag == "" {
			// Skip non-protobuf fields.
			return
		}

		// Get the search tags for the field.
		searchTag := getSearchTagForField(currentField)
		if searchTag == "-" || searchTag == "" {
			return
		}

		fieldMap[strings.ToLower(searchTag)] = fieldPath
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
