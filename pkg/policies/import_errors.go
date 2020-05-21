package policies

const (
	// ErrImportUnknown is used by policy import for all errors not in a more specific category
	ErrImportUnknown = "unknown_err"
	// ErrImportDuplicateID is used by policy import when an imported policy has the same ID as an existing policy and "overwrite" has not been set
	ErrImportDuplicateID = "duplicate_id"
	// ErrImportDuplicateName is used by policy import when an imported policy has the same name an existing policy and "overwrite" has not been set
	ErrImportDuplicateName = "duplicate_name"
	// ErrImportClustersOrNotifiersRemoved is used by policy import when the notifiers or cluster scopes were stripped out of an imported policy.  Not strictly an error, more like a warning
	ErrImportClustersOrNotifiersRemoved = "removed_clusters_or_notifiers"
	// ErrImportValidation is used by policy import when an imported policy does not pass validation
	ErrImportValidation = `invalid_policy`
)
