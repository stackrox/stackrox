package secrets

import (
	"reflect"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	// scrubStructTag field types are used to indicate credentials or credential dependent fields
	scrubStructTag = "scrub"
	// scrubTagAlways is a scrub tag type used to indicate a field is a credential
	scrubTagAlways = "always"
	// scrubTagMapValues is a scrub tag type used to indicate a map[string]string field that contain keys with secrets.
	// List of scrubbed keys are listed in mapKeysToReplace.
	scrubTagMapValues = "map-values"
	// scrubTagDependent is a scrub tag type used to indicate a field is dependent on credentials and could be used to exfiltrate credentials
	scrubTagDependent = "dependent"
	// ScrubReplacementStr is a string format of a masked credential
	ScrubReplacementStr = "******"
)

var (
	mapKeysToReplace = []string{"client_secret"}
)

// ScrubSecretsFromStructWithReplacement hides secret keys from an object with given replacement
func ScrubSecretsFromStructWithReplacement(obj interface{}, replacement string) {
	scrubber := func(field reflect.Value, scrubTag string) {
		switch scrubTag {
		case scrubTagAlways:
			if field.Kind() != reflect.String {
				utils.CrashOnError(errors.Errorf("expected string kind, got %s", field.Kind()))
			}
			if field.Type() != reflect.TypeOf(replacement) {
				utils.CrashOnError(errors.Errorf("field type mismatch %s!=%s", field.Type(), reflect.TypeOf(replacement)))
			}
			field.Set(reflect.ValueOf(replacement))
		case scrubTagMapValues:
			if field.Kind() != reflect.Map {
				utils.CrashOnError(errors.Errorf("expected map kind, got %s", field.Kind()))
			}
			if field.Type() != reflect.TypeOf(map[string]string{}) {
				utils.CrashOnError(errors.Errorf("field type mismatch %s!=%s", field.Type(), reflect.TypeOf(map[string]string{})))
			}

			if field.IsNil() {
				return
			}
			for _, scrubKey := range mapKeysToReplace {
				if val := field.MapIndex(reflect.ValueOf(scrubKey)); !val.IsValid() {
					continue
				}

				field.SetMapIndex(reflect.ValueOf(scrubKey), reflect.ValueOf(replacement))
			}
		}
	}

	visitStructTags(reflect.ValueOf(obj), scrubber)
}

func visitStructTags(value reflect.Value, visitor func(field reflect.Value, tag string)) {
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
			visitStructTags(fieldValue, visitor)
		case reflect.Ptr, reflect.Interface:
			if !fieldValue.IsNil() {
				visitStructTags(fieldValue.Elem(), visitor)
			}
		}
		visitor(fieldValue, valueType.Field(i).Tag.Get(scrubStructTag))
	}
}
