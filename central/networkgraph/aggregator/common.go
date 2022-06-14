package aggregator

import (
	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/networkgraph"
	"github.com/stackrox/stackrox/pkg/networkgraph/externalsrcs"
	"github.com/stackrox/stackrox/pkg/networkgraph/tree"
	"github.com/stackrox/stackrox/pkg/utils"
)

func mapToSupernet(networkTree tree.ReadOnlyNetworkTree,
	supernetCache map[string]*storage.NetworkEntityInfo,
	supernetPred func(e *storage.NetworkEntityInfo) bool, entities ...*storage.NetworkEntityInfo) {
	for _, entity := range entities {
		if !networkgraph.IsKnownExternalSrc(entity) {
			continue
		}

		cidr, err := externalsrcs.NetworkFromID(entity.GetId())
		if err != nil {
			utils.Should(errors.Wrapf(err, "getting CIDR from external source ID %s", entity.GetId()))
			*entity = *networkgraph.InternetEntity().ToProto()
			continue
		}
		*entity = *getSupernet(networkTree, supernetCache, cidr, supernetPred)
	}
}

func getSupernet(networkTree tree.ReadOnlyNetworkTree,
	cache map[string]*storage.NetworkEntityInfo,
	cidr string,
	supernetPred func(e *storage.NetworkEntityInfo) bool) *storage.NetworkEntityInfo {
	supernet := cache[cidr]
	if supernet == nil {
		supernet = networkTree.GetMatchingSupernetForCIDR(cidr, supernetPred)
		cache[cidr] = supernet
	}
	return supernet
}
