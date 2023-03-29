package utils

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stackrox/rox/pkg/errox"
)

func VerifyReferencedResourceOrigin(referencing, referenced declarativeconfig.ResourceWithTraits, referencingName, referencedName string) error {
	if !IsDeclarativeOrigin(referencing.GetTraits().GetOrigin()) ||
		referenced.GetTraits().GetOrigin() != storage.Traits_IMPERATIVE {
		return nil
	}
	// referenced is imperative or default, while referencing is not
	return errox.InvalidArgs.Newf("imperative %s can't be referenced by non-imperative %s", referencedName, referencingName)
}

func IsDeclarativeOrigin(origin storage.Traits_Origin) bool {
	return origin == storage.Traits_DECLARATIVE || origin == storage.Traits_DECLARATIVE_ORPHANED
}
