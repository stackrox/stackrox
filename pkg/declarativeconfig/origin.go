package declarativeconfig

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
)

// VerifyReferencedResourceOrigin returns an error if resource is forbidden from referencing other resource.
func VerifyReferencedResourceOrigin(referenced, referencing ResourceWithTraits, referencedName, referencingName string) error {
	if !IsDeclarativeOrigin(referencing) || referenced.GetTraits().GetOrigin() != storage.Traits_IMPERATIVE {
		return nil
	}
	// referenced is imperative or default, while referencing is not
	return errox.InvalidArgs.Newf("imperative resource %s can't be referenced by non-imperative resource %s", referencedName, referencingName)
}

// IsDeclarativeOrigin returns whether origin of resource is declarative or not.
func IsDeclarativeOrigin(resource ResourceWithTraits) bool {
	return resource.GetTraits().GetOrigin() == storage.Traits_DECLARATIVE || resource.GetTraits().GetOrigin() == storage.Traits_DECLARATIVE_ORPHANED
}
