package reconciliation

import (
	"errors"

	pkgErrors "github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/set"
)

var (
	log = logging.LoggerForModule()
)

// Perform factors out some of the common reconciliation logic to avoid duplication.
// It does the reconciliation logic, and closes the passed store at the end.
func Perform(store Store, existingIDs set.StringSet, resourceType string, removeFunc func(id string) error) error {
	idsToDelete := existingIDs.Difference(store.GetSet())
	defer store.Close(idsToDelete.Cardinality())
	if len(idsToDelete) == 0 {
		return nil
	}

	log.Infof("Deleting %d %s as a part of reconciliation", idsToDelete.Cardinality(), resourceType)

	var reconcileErrs error
	for id := range idsToDelete {
		reconcileErrs = errors.Join(reconcileErrs, removeFunc(id))
	}
	return pkgErrors.Wrapf(reconcileErrs, "%s reconciliation", resourceType)
}
