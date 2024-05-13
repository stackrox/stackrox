package nodecve

// NewEmptyResourceCountByCVESeverity creates a empty instance of type ResourceCountByCVESeverity.
func NewEmptyResourceCountByCVESeverity() ResourceCountByCVESeverity {
	return &emptyCountByNodeCVESeverity{}
}

type emptyCountByNodeCVESeverity struct{}

func (e *emptyCountByNodeCVESeverity) GetCriticalSeverityCount() int {
	return 0
}

func (e *emptyCountByNodeCVESeverity) GetImportantSeverityCount() int {
	return 0
}

func (e *emptyCountByNodeCVESeverity) GetModerateSeverityCount() int {
	return 0
}

func (e *emptyCountByNodeCVESeverity) GetLowSeverityCount() int {
	return 0
}
