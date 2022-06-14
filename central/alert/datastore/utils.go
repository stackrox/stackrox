package datastore

import "github.com/stackrox/rox/generated/storage"

func fillSortHelperFields(listAlert *storage.ListAlert) *storage.ListAlert {
	if listAlert.GetPolicy() == nil {
		return listAlert
	}
	listAlert.Policy.DeveloperInternalFields = &storage.ListAlertPolicy_DevFields{
		SORTName: listAlert.GetPolicy().GetName(),
	}
	return listAlert
}
