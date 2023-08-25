package converter

import (
	"github.com/stackrox/rox/generated/storage"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
)

// ClusterCVEParts represents the pieces of data in an cluster CVE.
type ClusterCVEParts struct {
	CVE      *storage.ClusterCVE
	Children []EdgeParts
}

// EdgeParts represents the piece of data in cluster-cve edge
type EdgeParts struct {
	Edge      *storage.ClusterCVEEdge
	ClusterID string
}

// NewClusterCVEParts creates and returns a new instance of ClusterCVEParts
func NewClusterCVEParts(cve *storage.ClusterCVE, clusters []*storage.Cluster, fixVersions string) ClusterCVEParts {
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

func generateClusterCVEEdge(cluster *storage.Cluster, cve *storage.ClusterCVE, fixVersions string) *storage.ClusterCVEEdge {
	ret := &storage.ClusterCVEEdge{
		Id:        pgSearch.IDFromPks([]string{cluster.GetId(), cve.GetId()}),
		IsFixable: len(fixVersions) != 0,
		ClusterId: cluster.GetId(),
		CveId:     cve.GetId(),
	}

	if ret.IsFixable {
		ret.HasFixedBy = &storage.ClusterCVEEdge_FixedBy{
			FixedBy: fixVersions,
		}
	}
	return ret
}
