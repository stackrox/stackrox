package common

type resourceCountByFixability struct {
	total   int
	fixable int
}

func (r *resourceCountByFixability) GetTotal() int {
	return r.total
}

func (r *resourceCountByFixability) GetFixable() int {
	return r.fixable
}

// ResourceCountByImageCVESeverity contains the counts of resources (Images or ImageCVEs) by CVE severity
type ResourceCountByImageCVESeverity struct {
	CriticalSeverityCount         int `db:"critical_severity_count"`
	FixableCriticalSeverityCount  int `db:"fixable_critical_severity_count"`
	ImportantSeverityCount        int `db:"important_severity_count"`
	FixableImportantSeverityCount int `db:"fixable_important_severity_count"`
	ModerateSeverityCount         int `db:"moderate_severity_count"`
	FixableModerateSeverityCount  int `db:"fixable_moderate_severity_count"`
	LowSeverityCount              int `db:"low_severity_count"`
	FixableLowSeverityCount       int `db:"fixable_low_severity_count"`
	UnknownSeverityCount          int `db:"unknown_severity_count"`
	FixableUnknownSeverityCount   int `db:"fixable_unknown_severity_count"`
}

func (r *ResourceCountByImageCVESeverity) GetCriticalSeverityCount() ResourceCountByFixability {
	return &resourceCountByFixability{
		total:   r.CriticalSeverityCount,
		fixable: r.FixableCriticalSeverityCount,
	}
}

func (r *ResourceCountByImageCVESeverity) GetImportantSeverityCount() ResourceCountByFixability {
	return &resourceCountByFixability{
		total:   r.ImportantSeverityCount,
		fixable: r.FixableImportantSeverityCount,
	}
}

func (r *ResourceCountByImageCVESeverity) GetModerateSeverityCount() ResourceCountByFixability {
	return &resourceCountByFixability{
		total:   r.ModerateSeverityCount,
		fixable: r.FixableModerateSeverityCount,
	}
}

func (r *ResourceCountByImageCVESeverity) GetLowSeverityCount() ResourceCountByFixability {
	return &resourceCountByFixability{
		total:   r.LowSeverityCount,
		fixable: r.FixableLowSeverityCount,
	}
}

func (r *ResourceCountByImageCVESeverity) GetUnknownSeverityCount() ResourceCountByFixability {
	return &resourceCountByFixability{
		total:   r.UnknownSeverityCount,
		fixable: r.FixableUnknownSeverityCount,
	}
}
