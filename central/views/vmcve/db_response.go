package vmcve

import (
	"time"

	"github.com/stackrox/rox/central/views/common"
)

type vmCVECoreResponse struct {
	CVE                        string     `db:"cve"`
	CVEIDs                     []string   `db:"cve_id"`
	VMsWithCriticalSeverity    int        `db:"critical_severity_count"`
	FixableVMsWithCriticalSev  int        `db:"fixable_critical_severity_count"`
	VMsWithImportantSeverity   int        `db:"important_severity_count"`
	FixableVMsWithImportantSev int        `db:"fixable_important_severity_count"`
	VMsWithModerateSeverity    int        `db:"moderate_severity_count"`
	FixableVMsWithModerateSev  int        `db:"fixable_moderate_severity_count"`
	VMsWithLowSeverity         int        `db:"low_severity_count"`
	FixableVMsWithLowSev       int        `db:"fixable_low_severity_count"`
	VMsWithUnknownSeverity     int        `db:"unknown_severity_count"`
	FixableVMsWithUnknownSev   int        `db:"fixable_unknown_severity_count"`
	TopCVSS                    *float32   `db:"cvss_max"`
	AffectedVMCount            int        `db:"virtual_machine_id_count"`
	FirstDiscoveredInSystem    *time.Time `db:"cve_created_time_min"`
	Published                  *time.Time `db:"cve_published_on_min"`
	EPSSProbabilityMax         *float32   `db:"epss_probability_max"`
}

func (c *vmCVECoreResponse) GetCVE() string {
	return c.CVE
}

func (c *vmCVECoreResponse) GetCVEIDs() []string {
	return c.CVEIDs
}

func (c *vmCVECoreResponse) GetVMsBySeverity() common.ResourceCountByCVESeverity {
	return &resourceCountByVMCVESeverity{
		CriticalSeverityCount:         c.VMsWithCriticalSeverity,
		FixableCriticalSeverityCount:  c.FixableVMsWithCriticalSev,
		ImportantSeverityCount:        c.VMsWithImportantSeverity,
		FixableImportantSeverityCount: c.FixableVMsWithImportantSev,
		ModerateSeverityCount:         c.VMsWithModerateSeverity,
		FixableModerateSeverityCount:  c.FixableVMsWithModerateSev,
		LowSeverityCount:              c.VMsWithLowSeverity,
		FixableLowSeverityCount:       c.FixableVMsWithLowSev,
		UnknownSeverityCount:          c.VMsWithUnknownSeverity,
		FixableUnknownSeverityCount:   c.FixableVMsWithUnknownSev,
	}
}

func (c *vmCVECoreResponse) GetTopCVSS() float32 {
	if c.TopCVSS == nil {
		return 0.0
	}
	return *c.TopCVSS
}

func (c *vmCVECoreResponse) GetAffectedVMCount() int {
	return c.AffectedVMCount
}

func (c *vmCVECoreResponse) GetFirstDiscoveredInSystem() *time.Time {
	return c.FirstDiscoveredInSystem
}

func (c *vmCVECoreResponse) GetPublishDate() *time.Time {
	return c.Published
}

func (c *vmCVECoreResponse) GetEPSSProbability() float32 {
	if c.EPSSProbabilityMax == nil {
		return 0.0
	}
	return *c.EPSSProbabilityMax
}

type vmCVECoreCount struct {
	CVECount int `db:"cve_count"`
}

type vmIDResponse struct {
	VMID string `db:"virtual_machine_id"`
}

// resourceCountByVMCVESeverity contains the counts of VMs by CVE severity.
type resourceCountByVMCVESeverity struct {
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

func (r *resourceCountByVMCVESeverity) GetCriticalSeverityCount() common.ResourceCountByFixability {
	return &resourceCountByFixability{total: r.CriticalSeverityCount, fixable: r.FixableCriticalSeverityCount}
}

func (r *resourceCountByVMCVESeverity) GetImportantSeverityCount() common.ResourceCountByFixability {
	return &resourceCountByFixability{total: r.ImportantSeverityCount, fixable: r.FixableImportantSeverityCount}
}

func (r *resourceCountByVMCVESeverity) GetModerateSeverityCount() common.ResourceCountByFixability {
	return &resourceCountByFixability{total: r.ModerateSeverityCount, fixable: r.FixableModerateSeverityCount}
}

func (r *resourceCountByVMCVESeverity) GetLowSeverityCount() common.ResourceCountByFixability {
	return &resourceCountByFixability{total: r.LowSeverityCount, fixable: r.FixableLowSeverityCount}
}

func (r *resourceCountByVMCVESeverity) GetUnknownSeverityCount() common.ResourceCountByFixability {
	return &resourceCountByFixability{total: r.UnknownSeverityCount, fixable: r.FixableUnknownSeverityCount}
}

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
