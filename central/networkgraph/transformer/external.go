package transformer

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/networkgraph"
)

type transformExternalDiscoveredImpl struct{}

// Transform will convert any discovered external flows to the Internet entity
// for masking/anonymization within the network graph.
func (t *transformExternalDiscoveredImpl) Transform(flows []*storage.NetworkFlow) []*storage.NetworkFlow {
	ret := make([]*storage.NetworkFlow, 0, len(flows))

	for _, flow := range flows {
		ret = append(ret, anonymizeDiscoveredFlow(flow))
	}

	return ret
}

// Return NetworkEntityInfo_INTERNET if entity is a 'discovered' external entity
// Otherwise, return entity.
func anonymizeDiscoveredEntity(entity *storage.NetworkEntityInfo) *storage.NetworkEntityInfo {
	if networkgraph.IsExternalDiscovered(entity) {
		return networkgraph.InternetEntity().ToProto()
	}
	return entity
}

// Transform the Src or Dst entities to INTERNET if they are 'discovered' external
// entities, otherwise the flow is unmodified.
func anonymizeDiscoveredFlow(flow *storage.NetworkFlow) *storage.NetworkFlow {
	props := flow.GetProps()
	src, dst := props.GetSrcEntity(), props.GetDstEntity()

	if !networkgraph.IsExternalDiscovered(src) && !networkgraph.IsExternalDiscovered(dst) {
		return flow
	}

	props = flow.GetProps()

	props.SrcEntity = anonymizeDiscoveredEntity(props.GetSrcEntity())
	props.DstEntity = anonymizeDiscoveredEntity(props.GetDstEntity())

	return flow
}
