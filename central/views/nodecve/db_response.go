package nodecve

import (
	"time"

	"github.com/stackrox/rox/central/views/common"
)

type nodeCVECoreResponse struct {
	CVE                               string     `db:"cve"`
	CVEIDs                            []string   `db:"cve_id"`
	TopCVSS                           float32    `db:"cvss_max"`
	NodeCount                         int        `db:"node_id_count"`
	NodesWithCriticalSeverity         int        `db:"critical_severity_count"`
	FixableNodesWithCriticalSeverity  int        `db:"fixable_critical_severity_count"`
	NodesWithImportantSeverity        int        `db:"important_severity_count"`
	FixableNodesWithImportantSeverity int        `db:"fixable_important_severity_count"`
	NodesWithModerateSeverity         int        `db:"moderate_severity_count"`
	FixableNodesWithModerateSeverity  int        `db:"fixable_moderate_severity_count"`
	NodesWithLowSeverity              int        `db:"low_severity_count"`
	FixableNodesWithLowSeverity       int        `db:"fixable_low_severity_count"`
	ImagesWithUnknownSeverity         int        `db:"unknown_severity_count"`
	FixableImagesWithUnknownSeverity  int        `db:"fixable_unknown_severity_count"`
	NodeIDs                           []string   `db:"node_id"`
	OperatingSystemCount              int        `db:"operating_system_count"`
	FirstDiscoveredInSystem           *time.Time `db:"cve_created_time_min"`
}

// GetCVE returns the CVE identifier
func (c *nodeCVECoreResponse) GetCVE() string {
	return c.CVE
}

// GetCVEIDs returns the unique primary key IDs associated with the node CVE
func (c *nodeCVECoreResponse) GetCVEIDs() []string {
	return c.CVEIDs
}

// GetTopCVSS returns the maximum CVSS score of the node CVE
func (c *nodeCVECoreResponse) GetTopCVSS() float32 {
	return c.TopCVSS
}

// GetNodeCount returns the number of nodes affected by the node CVE
func (c *nodeCVECoreResponse) GetNodeCount() int {
	return c.NodeCount
}

// GetNodeCountBySeverity returns the number of nodeMap of each severity level
func (c *nodeCVECoreResponse) GetNodeCountBySeverity() common.ResourceCountByCVESeverity {
	return &countByNodeCVESeverity{
		CriticalSeverityCount:         c.NodesWithCriticalSeverity,
		FixableCriticalSeverityCount:  c.FixableNodesWithCriticalSeverity,
		ImportantSeverityCount:        c.NodesWithImportantSeverity,
		FixableImportantSeverityCount: c.FixableNodesWithImportantSeverity,
		ModerateSeverityCount:         c.NodesWithModerateSeverity,
		FixableModerateSeverityCount:  c.FixableNodesWithModerateSeverity,
		LowSeverityCount:              c.NodesWithLowSeverity,
		FixableLowSeverityCount:       c.FixableNodesWithLowSeverity,
		UnknownSeverityCount:          c.ImagesWithUnknownSeverity,
		FixableUnknownSeverityCount:   c.FixableImagesWithUnknownSeverity,
	}
}

// GetNodeIDs returns the node ids affected by the node CVE
func (c *nodeCVECoreResponse) GetNodeIDs() []string {
	return c.NodeIDs
}

// GetFirstDiscoveredInSystem returns the first time the node CVE was discovered in the system
func (c *nodeCVECoreResponse) GetFirstDiscoveredInSystem() *time.Time {
	return c.FirstDiscoveredInSystem
}

// GetOperatingSystemCount returns the number of operating systems of nodeMap affected by the node CVE
func (c *nodeCVECoreResponse) GetOperatingSystemCount() int {
	return c.OperatingSystemCount
}

type nodeCVECoreCount struct {
	CVECount int `db:"cve_count"`
}

type countByNodeCVESeverity struct {
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

// NewCountByNodeCVESeverity creates and returns a node resource count by CVE severity.
func NewCountByNodeCVESeverity(
	critical, fixableCritical,
	important, fixableImportant,
	moderate, fixableModerate,
	low, fixableLow, unknown, fixableUnknown int) common.ResourceCountByCVESeverity {
	return &countByNodeCVESeverity{
		CriticalSeverityCount:         critical,
		FixableCriticalSeverityCount:  fixableCritical,
		ImportantSeverityCount:        important,
		FixableImportantSeverityCount: fixableImportant,
		ModerateSeverityCount:         moderate,
		FixableModerateSeverityCount:  fixableModerate,
		LowSeverityCount:              low,
		FixableLowSeverityCount:       fixableLow,
		UnknownSeverityCount:          unknown,
		FixableUnknownSeverityCount:   fixableUnknown,
	}
}

func (c *countByNodeCVESeverity) GetCriticalSeverityCount() common.ResourceCountByFixability {
	return &resourceCountByFixability{
		total:   c.CriticalSeverityCount,
		fixable: c.FixableCriticalSeverityCount,
	}
}

func (c *countByNodeCVESeverity) GetImportantSeverityCount() common.ResourceCountByFixability {
	return &resourceCountByFixability{
		total:   c.ImportantSeverityCount,
		fixable: c.FixableImportantSeverityCount,
	}
}

func (c *countByNodeCVESeverity) GetModerateSeverityCount() common.ResourceCountByFixability {
	return &resourceCountByFixability{
		total:   c.ModerateSeverityCount,
		fixable: c.FixableModerateSeverityCount,
	}
}

func (c *countByNodeCVESeverity) GetLowSeverityCount() common.ResourceCountByFixability {
	return &resourceCountByFixability{
		total:   c.LowSeverityCount,
		fixable: c.FixableLowSeverityCount,
	}
}

func (c *countByNodeCVESeverity) GetUnknownSeverityCount() common.ResourceCountByFixability {
	return &resourceCountByFixability{
		total:   c.UnknownSeverityCount,
		fixable: c.FixableUnknownSeverityCount,
	}
}

type resourceCountByFixability struct {
	total   int
	fixable int
}

func (c *resourceCountByFixability) GetTotal() int {
	return c.total
}

func (c *resourceCountByFixability) GetFixable() int {
	return c.fixable
}

type nodeResponse struct {
	NodeID string `db:"node_id"`
}

func (r *nodeResponse) GetNodeID() string {
	return r.NodeID
}
