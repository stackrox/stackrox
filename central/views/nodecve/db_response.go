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
	OperatingSystemCount       int        `db:"operating_system_count"`
	FirstDiscoveredInSystem    *time.Time `db:"cve_created_time_min"`
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

// GetFirstDiscoveredInSystem returns the first time the node CVE was discovered in the system
func (n *nodeCVECoreResponse) GetFirstDiscoveredInSystem() *time.Time {
	return n.FirstDiscoveredInSystem
}

// GetOperatingSystemCount returns the number of operating systems of nodeMap affected by the node CVE
func (n *nodeCVECoreResponse) GetOperatingSystemCount() int {
	return n.OperatingSystemCount
}

type nodeCVECoreCount struct {
	CVECount int `db:"cve_count"`
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
	NodeName        string     `db:"node_name"`
	ClusterName     string     `db:"cluster"`
	OperatingSystem string     `db:"operating_system"`
	ScanTime        *time.Time `db:"node_scan_time"`
	// CVEBySeverity are with another jira <link>
}

func (r nodeResponse) GetNodeID() string {
	return r.NodeID
}
