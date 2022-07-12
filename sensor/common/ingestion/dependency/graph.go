package dependency

import "github.com/stackrox/rox/sensor/common/ingestion"

type Graph struct {
	stores *ingestion.ResourceStore
}

func NewGraph(stores *ingestion.ResourceStore) *Graph {
	return &Graph{
		stores: stores,
	}
}
