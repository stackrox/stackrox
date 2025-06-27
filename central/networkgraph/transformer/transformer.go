package transformer

import (
	"github.com/stackrox/rox/generated/storage"
)

// A NetworkFlowTransformer will modify a list of flows for the
// purposes of masking or anonymization.
type NetworkFlowTransformer interface {
	Transform(flows []*storage.NetworkFlow) []*storage.NetworkFlow
}

func NewExternalDiscoveredTransformer() NetworkFlowTransformer {
	return &transformExternalDiscoveredImpl{}
}
