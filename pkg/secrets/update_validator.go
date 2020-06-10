package secrets

import (
	"reflect"
	"strings"

	"github.com/pkg/errors"
)

// ValidateUpdatedStruct checks that scrub:always fields are empty/masked and scrub:dependent fields are equal to existing
func ValidateUpdatedStruct(updated interface{}, existing interface{}) error {
	return validateUpdatedStruct(reflect.ValueOf(updated), reflect.ValueOf(existing), []string{})
}

func validateUpdatedStruct(updated reflect.Value, existing reflect.Value, path []string) error {
	updated = reflect.Indirect(updated)
	existing = reflect.Indirect(existing)
	if !updated.IsValid() || !existing.IsValid() {
		return errors.New("invalid input")
	}
	updatedType := updated.Type()
	existingType := existing.Type()
	if updatedType != existingType {
		return errors.Errorf("type not equal: '%s' != '%s'", updatedType, existingType)
	}
	if updated.Kind() != reflect.Struct {
		return errors.Errorf("expected struct, got %s", updated.Kind())
	}
	path = append(path, updatedType.Name())
	for i := 0; i < updatedType.NumField(); i++ {
		updatedField := updated.Field(i)
		existingField := existing.Field(i)
		switch updatedField.Kind() {
		case reflect.Struct:
			if err := validateUpdatedStruct(updatedField, existingField, path); err != nil {
				return err
			}
		case reflect.Ptr, reflect.Interface:
			if updatedField.IsNil() && !existingField.IsNil() {
				return errors.Errorf("non-nil existing field '%s'",
					strings.Join(append(path, updatedType.Field(i).Name), "."))
			}
			if !updatedField.IsNil() {
				if err := validateUpdatedStruct(updatedField.Elem(), existingField.Elem(), path); err != nil {
					return err
				}
			}
		}
		switch updatedType.Field(i).Tag.Get(scrubStructTag) {
		case scrubTagAlways:
			if !updatedField.IsZero() && updatedField.String() != ScrubReplacementStr {
				return errors.Errorf("non-zero or unmasked credential field '%s'",
					strings.Join(append(path, updatedType.Field(i).Name), "."))
			}
		case scrubTagDependent:
			if !reflect.DeepEqual(updatedField.Interface(), existingField.Interface()) {
				return errors.Errorf("credentials required to update field '%s'",
					strings.Join(append(path, updatedType.Field(i).Name), "."))
			}
		}
	}
	return nil
}
