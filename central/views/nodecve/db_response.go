package nodecve

import (
	"time"
)

type nodeCVECoreResponse struct {
	CVE                        string     `db:"cve"`
	CVEIDs                     []string   `db:"cve_id"`
	TopCVSS                    float32    `db:"cvss_max"`
	NodeCount                  int        `db:"node_id_count"`
	NodesWithCriticalSeverity  int        `db:"critical_severity_count"`
	NodesWithImportantSeverity int        `db:"important_severity_count"`
	NodesWithModerateSeverity  int        `db:"moderate_severity_count"`
	NodesWithLowSeverity       int        `db:"low_severity_count"`
	NodeIDs                    []string   `db:"node_id"`
	FirstDiscoveredTime        *time.Time `db:"cve_created_time"`
	OperatingSystemCount       int        `db:"operating_system"`
}

// GetCVE returns the CVE identifier
func (c *nodeCVECoreResponse) GetCVE() string {
	return c.CVE
}

// GetCVEID returns the unique primary key ID associated with the node CVE
func (c *nodeCVECoreResponse) GetCVEIDs() []string {
	return c.CVEIDs
}

// GetCVSS returns the CVSS score of the node CVE
func (c *nodeCVECoreResponse) GetTopCVSS() float32 {
	return c.TopCVSS
}

// GetNodeCount returns the number of nodes affected by the node CVE
func (c *nodeCVECoreResponse) GetNodeCount() int {
	return c.NodeCount
}

// GetNodeCountBySeverity returns the number of nodes of each severity level
func (n *nodeCVECoreResponse) GetNodeCountBySeverity() ResourceCountByCVESeverity {
	return &countByNodeCVESeverity{
		CriticalSeverityCount:  n.NodesWithCriticalSeverity,
		ImportantSeverityCount: n.NodesWithImportantSeverity,
		ModerateSeverityCount:  n.NodesWithModerateSeverity,
		LowSeverityCount:       n.NodesWithLowSeverity,
	}
}

// GetNodeIDs returns the node ids affected by the node CVE
func (n *nodeCVECoreResponse) GetNodeIDs() []string {
	return n.NodeIDs
}

// GetFirstDiscoveredTime returns the first time the node CVE was discovered in the system
func (n *nodeCVECoreResponse) GetFirstDiscoveredTime() *time.Time {
	return n.FirstDiscoveredTime
}

// GetOperatingSystemCount returns the number of operating systems of nodes affected by the node CVE
func (n *nodeCVECoreResponse) GetOperatingSystemCount() int {
	return n.OperatingSystemCount
}

type nodeCVECoreCount struct {
	CVECount int `db:"cve_id_count"`
}

type countByNodeCVESeverity struct {
	CriticalSeverityCount  int `db:"critical_severity_count"`
	ImportantSeverityCount int `db:"important_severity_count"`
	ModerateSeverityCount  int `db:"moderate_severity_count"`
	LowSeverityCount       int `db:"low_severity_count"`
}

func (c countByNodeCVESeverity) GetCriticalSeverityCount() int {
	return c.CriticalSeverityCount
}

func (c countByNodeCVESeverity) GetImportantSeverityCount() int {
	return c.ImportantSeverityCount
}

func (c countByNodeCVESeverity) GetModerateSeverityCount() int {
	return c.ModerateSeverityCount
}

func (c countByNodeCVESeverity) GetLowSeverityCount() int {
	return c.LowSeverityCount
}

type nodeResponse struct {
	NodeID string `db:"node_id"`

	// Following are supported sort options.
	// TBD
}
