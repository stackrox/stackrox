package endpoints

import (
	"fmt"
	"net/url"
	"reflect"

	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/pkg/urlfmt"
	"github.com/stackrox/stackrox/pkg/utils"
)

const (
	// validateStructTag field types are used to indicate endpoints that need to be validated
	validateStructTag = "validate"
	// validateTagNoLocalEndpoint is a validate tag type used to indicate the endpoint must not reference localhost or cluster metadata svc
	validateTagNoLocalEndpoint = "nolocalendpoint"
)

// ValidateEndpoints validates endpoints to ensure they do not reference localhost or local metadata service
func ValidateEndpoints(obj interface{}) error {
	validator := func(field reflect.Value, validateTag string) error {
		switch validateTag {
		case validateTagNoLocalEndpoint:
			// appending http:// to ensure url.Parse works, because it requires a scheme
			url, err := url.Parse(fmt.Sprintf("http://%s", urlfmt.TrimHTTPPrefixes(field.String())))
			if err != nil {
				return err
			}
			if field.Kind() != reflect.String {
				utils.CrashOnError(errors.Errorf("expected string kind, got %s", field.Kind()))
			}
			if field.String() != "" {
				if err := validate(url.Hostname()); err != nil {
					return err
				}
			}
		}
		return nil
	}
	return visitStructTags(reflect.ValueOf(obj), validator)
}

func visitStructTags(value reflect.Value, visitor func(field reflect.Value, tag string) error) error {
	if value.Kind() == reflect.Ptr {
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct {
		return nil
	}
	valueType := value.Type()
	for i := 0; i < value.NumField(); i++ {
		var err error
		fieldValue := value.Field(i)
		switch fieldValue.Kind() {
		case reflect.Struct:
			err = visitStructTags(fieldValue, visitor)
		case reflect.Ptr, reflect.Interface:
			if !fieldValue.IsNil() {
				err = visitStructTags(fieldValue.Elem(), visitor)
			}
		}
		if err != nil {
			return err
		}
		err = visitor(fieldValue, valueType.Field(i).Tag.Get(validateStructTag))
		if err != nil {
			return err
		}
	}
	return nil
}

func validate(hostname string) error {
	if hostname == "127.0.0.1" || hostname == "localhost" {
		return errors.New("endpoint cannot reference localhost")
	}
	if hostname == "169.254.169.254" || hostname == "metadata.google.internal" {
		return errors.New("endpoint cannot reference the cluster metadata service")
	}
	return nil
}
