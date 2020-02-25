package converter

import (
	"strings"

	"github.com/facebookincubator/nvdtools/cvefeed/nvd/schema"
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

// NewClusterCVEParts returns new instance of ClusterCVEParts
func NewClusterCVEParts(cve *storage.CVE, clusters []*storage.Cluster, nvdCVE *schema.NVDCVEFeedJSON10DefCVEItem) ClusterCVEParts {
	ret := ClusterCVEParts{
		CVE: cve,
	}
	for _, cluster := range clusters {
		ret.Children = append(ret.Children, EdgeParts{
			Edge:      generateClusterCVEEdge(cluster, cve, nvdCVE),
			ClusterID: cluster.GetId(),
		})
	}
	return ret
}

func generateClusterCVEEdge(cluster *storage.Cluster, cve *storage.CVE, nvdCVE *schema.NVDCVEFeedJSON10DefCVEItem) *storage.ClusterCVEEdge {
	fixVersions := getFixedVersions(nvdCVE)
	ret := &storage.ClusterCVEEdge{
		Id:        edges.EdgeID{ParentID: cluster.GetId(), ChildID: cve.GetId()}.ToString(),
		IsFixable: len(fixVersions) != 0,
	}

	if ret.IsFixable {
		ret.HasFixedBy = &storage.ClusterCVEEdge_FixedBy{
			FixedBy: strings.Join(fixVersions, ","),
		}
	}
	return ret
}
