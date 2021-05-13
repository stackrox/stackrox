package converter

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/dackbox/edges"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// ClusterCVEParts represents the pieces of data in an cluster CVE.
type ClusterCVEParts struct {
	CVE      *storage.CVE
	Children []EdgeParts
}

// EdgeParts represents the piece of data in cluster-cve edge
type EdgeParts struct {
	Edge      *storage.ClusterCVEEdge
	ClusterID string
}

// NewClusterCVEParts creates and returns a new instance of ClusterCVEParts
func NewClusterCVEParts(cve *storage.CVE, clusters []*storage.Cluster, fixVersions string) ClusterCVEParts {
	ret := ClusterCVEParts{
		CVE: cve,
	}
	for _, cluster := range clusters {
		ret.Children = append(ret.Children, EdgeParts{
			Edge:      generateClusterCVEEdge(cluster, cve, fixVersions),
			ClusterID: cluster.GetId(),
		})
	}
	return ret
}

func generateClusterCVEEdge(cluster *storage.Cluster, cve *storage.CVE, fixVersions string) *storage.ClusterCVEEdge {
	ret := &storage.ClusterCVEEdge{
		Id:        edges.EdgeID{ParentID: cluster.GetId(), ChildID: cve.GetId()}.ToString(),
		IsFixable: len(fixVersions) != 0,
	}

	if ret.IsFixable {
		ret.HasFixedBy = &storage.ClusterCVEEdge_FixedBy{
			FixedBy: fixVersions,
		}
	}
	return ret
}
