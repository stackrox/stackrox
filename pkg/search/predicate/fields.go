package predicate

import (
	"reflect"
	"strings"
)

func mapSearchTagsToFieldPaths(toWalk interface{}) map[string]FieldPath {
	ret := make(map[string]FieldPath)
	VisitFields(toWalk, func(fieldPath FieldPath) {
		currentField := fieldPath[len(fieldPath)-1]
		protoTag, oneofTag := getProtobufTags(currentField)
		if protoTag == "" && oneofTag == "" {
			// Skip non-protobuf fields.
			return
		}
		searchTag := getSearchTagForField(currentField)
		if searchTag == "-" || searchTag == "" {
			return
		}

		ret[searchTag] = fieldPath
	})
	return ret
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
