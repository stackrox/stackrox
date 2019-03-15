package reconciliation

import (
	"fmt"

	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/set"
)

var (
	log = logging.LoggerForModule()
)

// PerformDryRun performs reconciliation, but takes a dryRun flag.
func PerformDryRun(store Store, existingIDs set.StringSet, resourceType string, removeFunc func(id string) error, dryRun bool) error {
	defer store.Close()
	idsToDelete := existingIDs.Difference(store.GetSet()).AsSlice()
	if len(idsToDelete) == 0 {
		return nil
	}

	resources := fmt.Sprintf("%s %+v", resourceType, idsToDelete)
	if dryRun {
		log.Infof("Reconciliation in dry-run mode; would have deleted %s", resources)
		return nil
	}
	log.Infof("Deleting %s as a part of reconciliation", resources)

	errList := errorhelpers.NewErrorList("Network Policy reconciliation")
	for _, id := range idsToDelete {
		errList.AddError(removeFunc(id))
	}
	return errList.ToError()
}

// Perform factors out some of the common reconciliation logic to avoid duplication.
// It does the reconciliation logic, and closes the passed store at the end.
func Perform(store Store, existingIDs set.StringSet, resourceType string, removeFunc func(id string) error) error {
	return PerformDryRun(store, existingIDs, resourceType, removeFunc, false)
}
