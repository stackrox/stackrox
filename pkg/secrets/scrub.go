package secrets

import (
	"reflect"
)

const (
	scrubStructTag = "scrub"
	scrubTagAlways = "always"
	// ReplacementStr is a string format of a masked secret
	ReplacementStr = "******"
)

// ScrubSecretsFromStruct removes secret keys from an object
func ScrubSecretsFromStruct(obj interface{}) {
	value := reflect.ValueOf(obj)
	if value.Kind() == reflect.Ptr {
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct {
		return
	}

	valueType := value.Type()
	for i := 0; i < value.NumField(); i++ {
		typeField := valueType.Field(i)
		fieldKind := value.Field(i).Kind()
		switch fieldKind {
		case reflect.Struct:
			in := value.Field(i).Interface()
			ScrubSecretsFromStruct(&in)
		case reflect.Ptr, reflect.Interface:
			if !value.Field(i).IsNil() {
				ScrubSecretsFromStruct(value.Field(i).Interface())
			}
		}
		switch typeField.Tag.Get(scrubStructTag) {
		case scrubTagAlways:
			value.Field(i).Set(reflect.ValueOf(""))
		}
	}
}

// ScrubSecretsFromStructWithReplacement hides secret keys from an object with given replacement
func ScrubSecretsFromStructWithReplacement(obj interface{}, replacement string) {
	value := reflect.ValueOf(obj)
	if value.Kind() == reflect.Ptr {
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct {
		return
	}

	valueType := value.Type()
	for i := 0; i < value.NumField(); i++ {
		typeField := valueType.Field(i)
		fieldKind := value.Field(i).Kind()
		switch fieldKind {
		case reflect.Struct:
			in := value.Field(i).Interface()
			ScrubSecretsFromStructWithReplacement(&in, replacement)
		case reflect.Ptr, reflect.Interface:
			if !value.Field(i).IsNil() {
				ScrubSecretsFromStructWithReplacement(value.Field(i).Interface(), replacement)
			}
		}
		switch typeField.Tag.Get(scrubStructTag) {
		case scrubTagAlways:
			if value.Field(i).String() != "" {
				value.Field(i).Set(reflect.ValueOf(replacement))
			}
		}
	}
}
