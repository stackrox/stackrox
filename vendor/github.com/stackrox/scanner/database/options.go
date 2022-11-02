package database

// DatastoreOptions defines the options for reading from and writing to the database.
type DatastoreOptions struct {
	// WithFeatures is a read-only option.
	// `true` indicates the Features field should be filled.
	WithFeatures bool
	// WithVulnerabilities is a read-only option.
	// `true` means the Features field should be filled
	// and their AffectedBy fields should contain every vulnerability that
	// affect them.
	WithVulnerabilities bool
	// UncertifiedRHEL indicates the returned results should be for an
	// uncertified RHEL layer, if the layer's namespace is RHEL.
	UncertifiedRHEL bool
}

// GetWithFeatures returns "true" if WithFeatures is "true"; "false" otherwise.
func (o *DatastoreOptions) GetWithFeatures() bool {
	if o == nil {
		return false
	}
	return o.WithFeatures
}

// GetWithVulnerabilities returns "true" if WithVulnerabilities is "true"; "false" otherwise.
func (o *DatastoreOptions) GetWithVulnerabilities() bool {
	if o == nil {
		return false
	}
	return o.WithVulnerabilities
}

// GetUncertifiedRHEL returns "true" if UncertifiedRHEL is "true"; "false" otherwise.
func (o *DatastoreOptions) GetUncertifiedRHEL() bool {
	if o == nil {
		return false
	}
	return o.UncertifiedRHEL
}
