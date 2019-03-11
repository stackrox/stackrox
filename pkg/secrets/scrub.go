package secrets

import (
	"reflect"
)

const (
	scrubStructTag = "scrub"
	scrubTagAlways = "always"
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
