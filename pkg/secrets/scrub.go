package secrets

import (
	"reflect"
)

const (
	// scrubStructTag field types are used to indicate credentials or credential dependent fields
	scrubStructTag = "scrub"
	// scrubTagAlways is a scrub tag type used to indicate a field is a credential
	scrubTagAlways = "always"
	// scrubTagDependent is a scrub tag type used to indicate a field is dependent on credentials and could be used to exfiltrate credentials
	scrubTagDependent = "dependent"
	// ScrubReplacementStr is a string format of a masked credential
	ScrubReplacementStr = "******"
)

// ScrubSecretsFromStructWithReplacement hides secret keys from an object with given replacement
func ScrubSecretsFromStructWithReplacement(obj interface{}, replacement string) {
	scrubSecretsFromStructWithReplacement(reflect.ValueOf(obj), replacement)
}

func scrubSecretsFromStructWithReplacement(value reflect.Value, replacement string) {
	if value.Kind() == reflect.Ptr {
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct {
		return
	}
	valueType := value.Type()
	for i := 0; i < value.NumField(); i++ {
		fieldValue := value.Field(i)
		switch fieldValue.Kind() {
		case reflect.Struct:
			scrubSecretsFromStructWithReplacement(fieldValue, replacement)
		case reflect.Ptr, reflect.Interface:
			if !fieldValue.IsNil() {
				scrubSecretsFromStructWithReplacement(fieldValue.Elem(), replacement)
			}
		}
		switch valueType.Field(i).Tag.Get(scrubStructTag) {
		case scrubTagAlways:
			if fieldValue.String() != "" {
				fieldValue.Set(reflect.ValueOf(replacement))
			}
		}
	}
}
