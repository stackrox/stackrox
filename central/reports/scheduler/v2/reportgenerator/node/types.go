package node

import (
	"time"

	"github.com/stackrox/rox/generated/storage"
)

// NodeCVEQueryResponse contains the fields of a node CVE report query response.
type NodeCVEQueryResponse struct {
	Cluster          *string                        `db:"cluster"`
	Node             *string                        `db:"node"`
	Component        *string                        `db:"component"`
	ComponentVersion *string                        `db:"component_version"`
	CVEID            *string                        `db:"cve_id"`
	CVE              *string                        `db:"cve"`
	Fixable          *bool                          `db:"fixable"`
	FixedByVersion   *string                        `db:"fixed_by"`
	Severity         *storage.VulnerabilitySeverity `db:"severity"`
	CVSS             *float64                       `db:"cvss"`
	FirstOccurrence  *time.Time                     `db:"cve_created_time"`

	Link string
}

func (r *NodeCVEQueryResponse) GetCluster() string {
	if r.Cluster == nil {
		return ""
	}
	return *r.Cluster
}

func (r *NodeCVEQueryResponse) GetNode() string {
	if r.Node == nil {
		return ""
	}
	return *r.Node
}

func (r *NodeCVEQueryResponse) GetComponent() string {
	if r.Component == nil {
		return ""
	}
	return *r.Component
}

func (r *NodeCVEQueryResponse) GetComponentVersion() string {
	if r.ComponentVersion == nil {
		return ""
	}
	return *r.ComponentVersion
}

func (r *NodeCVEQueryResponse) GetCVEID() string {
	if r.CVEID == nil {
		return ""
	}
	return *r.CVEID
}

func (r *NodeCVEQueryResponse) GetCVE() string {
	if r.CVE == nil {
		return ""
	}
	return *r.CVE
}

func (r *NodeCVEQueryResponse) GetFixable() bool {
	if r.Fixable == nil {
		return false
	}
	return *r.Fixable
}

func (r *NodeCVEQueryResponse) GetFixedByVersion() string {
	if r.FixedByVersion == nil {
		return ""
	}
	return *r.FixedByVersion
}

func (r *NodeCVEQueryResponse) GetSeverity() storage.VulnerabilitySeverity {
	if r.Severity == nil {
		return storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY
	}
	return *r.Severity
}

func (r *NodeCVEQueryResponse) GetCVSS() float64 {
	if r.CVSS == nil {
		return 0.0
	}
	return *r.CVSS
}

func (r *NodeCVEQueryResponse) GetFirstOccurrence() string {
	if r.FirstOccurrence == nil {
		return "Not Available"
	}
	return r.FirstOccurrence.Format("January 02, 2006")
}

func (r *NodeCVEQueryResponse) GetLink() string {
	return r.Link
}
