package common

// NewEmptyResourceCountByCVESeverity creates a empty instance of type ResourceCountByCVESeverity.
func NewEmptyResourceCountByCVESeverity() ResourceCountByCVESeverity {
	return &emptyResourceCountByCVESeverity{}
}

type emptyResourceCountByCVESeverity struct{}

func (r *emptyResourceCountByCVESeverity) GetCriticalSeverityCount() ResourceCountByFixability {
	return NewEmptyResourceCountByFixability()
}

func (r *emptyResourceCountByCVESeverity) GetImportantSeverityCount() ResourceCountByFixability {
	return NewEmptyResourceCountByFixability()
}

func (r *emptyResourceCountByCVESeverity) GetModerateSeverityCount() ResourceCountByFixability {
	return NewEmptyResourceCountByFixability()
}

func (r *emptyResourceCountByCVESeverity) GetLowSeverityCount() ResourceCountByFixability {
	return NewEmptyResourceCountByFixability()
}

// NewEmptyResourceCountByFixability creates a empty instance of type ResourceCountByFixability.
func NewEmptyResourceCountByFixability() ResourceCountByFixability {
	return &emptyResourceCountByFixability{}
}

type emptyResourceCountByFixability struct{}

func (r *emptyResourceCountByFixability) GetTotal() int {
	return 0
}

func (r *emptyResourceCountByFixability) GetFixable() int {
	return 0
}
