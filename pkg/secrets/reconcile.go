package secrets

import (
	"reflect"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/protoreflect"
)

// ReconcileScrubbedStructWithExisting replaces scrub:always fields in updated with the corresponding field values in existing
func ReconcileScrubbedStructWithExisting(updated interface{}, existing interface{}) error {
	// walk updated first to verify scrub:always fields are empty/masked and scrub:dependent fields are equal to existing
	if err := reconcileScrubbedWithExisting(reflect.ValueOf(updated), reflect.ValueOf(existing), true, nil); err != nil {
		return err
	}
	// walk updated after verification and reconcile scrub:always fields with existing
	return reconcileScrubbedWithExisting(reflect.ValueOf(updated), reflect.ValueOf(existing), false, nil)
}

func reconcileScrubbedWithExisting(updated reflect.Value, existing reflect.Value, verifyOnly bool, path []string) error {
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

	skipDependentReconcile := false
	for i := 0; i < updatedType.NumField(); i++ {
		if protoreflect.IsProtoMessage(updatedType) && protoreflect.IsInternalGeneratorField(updatedType.Field(i)) {
			continue
		}

		updatedField := updated.Field(i)
		if updatedField.Kind() == reflect.Bool && updatedType.Field(i).Tag.Get(scrubStructTag) == scrubTagDisableDependentIfTrue {
			if updatedField.Bool() {
				skipDependentReconcile = true // skip because the field tagged as "disableDependentIfTrue" is true
			}
		}
	}

	path = append(path, updatedType.Name())
	for i := 0; i < updatedType.NumField(); i++ {
		if protoreflect.IsProtoMessage(updatedType) && protoreflect.IsInternalGeneratorField(updatedType.Field(i)) {
			continue
		}

		updatedField := updated.Field(i)
		existingField := existing.Field(i)
		switch updatedField.Kind() {
		case reflect.Struct:
			if err := reconcileScrubbedWithExisting(updatedField, existingField, verifyOnly, path); err != nil {
				return err
			}
		case reflect.Ptr, reflect.Interface:
			if updatedField.IsNil() && !existingField.IsNil() {
				return errors.Errorf("non-nil existing field '%s'",
					strings.Join(append(path, updatedType.Field(i).Name), "."))
			}
			if !updatedField.IsNil() {
				if err := reconcileScrubbedWithExisting(updatedField.Elem(), existingField.Elem(), verifyOnly, path); err != nil {
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
			if updatedField.Type() != existingField.Type() {
				return errors.Errorf("field type mismatch %s!=%s", updatedField.Type(), existingField.Type())
			}
			if updatedField.Kind() != reflect.String {
				return errors.Errorf("expected string kind, got %s", updatedField.Kind())
			}
			if !verifyOnly {
				updatedField.Set(reflect.ValueOf(existingField.String()))
			}
		case scrubTagDependent:
			if !skipDependentReconcile && !reflect.DeepEqual(updatedField.Interface(), existingField.Interface()) {
				return errors.Errorf("credentials required to update field '%s'",
					strings.Join(append(path, updatedType.Field(i).Name), "."))
			}
		}
	}
	return nil
}
