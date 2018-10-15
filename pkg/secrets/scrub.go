package secrets

import (
	"reflect"
	"strings"
)

// secretKeys lists keys that have secret values that should be scrubbed
// out of a config before returning it in the API.
type secretKeys map[string]struct{}

func newSecretKeys(keys []string) secretKeys {
	sk := make(secretKeys)
	for _, k := range keys {
		sk[strings.ToLower(k)] = struct{}{}
	}
	return sk
}

func (sk secretKeys) shouldScrub(key string) bool {
	_, present := sk[strings.ToLower(key)]
	return present
}

var scrubber = newSecretKeys([]string{
	"authToken",
	"oauthToken",
	"password",
	"secretKey",
	"serviceAccount",
	"AccessKeyId",
	"SecretAccessKey",
})

// ScrubSecretsFromStruct removes secret keys from an object
func ScrubSecretsFromStruct(obj interface{}) {
	value := reflect.ValueOf(obj)
	if value.Kind() == reflect.Ptr {
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct {
		return
	}
	for i := 0; i < value.NumField(); i++ {
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
		if scrubber.shouldScrub(value.Type().Field(i).Name) {
			value.Field(i).Set(reflect.ValueOf(""))
		}
	}
}
