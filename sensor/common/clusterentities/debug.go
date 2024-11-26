package clusterentities

import (
	"encoding/json"
)

// Debug returns an object that represents the current state of the entire store
func (e *Store) Debug() []byte {
	m := make(map[string]interface{})
	m["endpoints"] = e.endpointsStore.debug()
	m["IPs"] = e.podIPsStore.debug()
	m["containerIDs"] = e.containerIDsStore.debug()
	// json pretty-printer will sort it for us
	m["events"] = e.trace

	ret, err := json.Marshal(m)
	if err != nil {
		log.Errorf("Error marshalling store debug: %v", err)
	}
	return ret
}

func (e *containerIDsStore) debug() interface{} {
	dbg := make(map[string]map[string]interface{})
	dbg["containerIDMap"] = make(map[string]interface{})
	dbg["historicalContainerIDs"] = make(map[string]interface{})
	dbg["reverseContainerIDMap"] = make(map[string]interface{})

	for cID, metadata := range e.containerIDMap {
		dbg["containerIDMap"][cID] = metadata
	}
	for cID, submap := range e.historicalContainerIDs {
		for metadata, status := range submap {
			dbg["historicalContainerIDs"][cID] = map[string]interface{}{
				"metadata":  metadata,
				"ticksLeft": status.ticksLeft,
			}
		}
	}
	for deplID, cIDSet := range e.reverseContainerIDMap {
		dbg["reverseContainerIDMap"][deplID] = cIDSet.AsSlice()
	}
	return dbg
}

func (e *podIPsStore) debug() interface{} {
	dbg := make(map[string]map[string]interface{})
	dbg["ipMap"] = make(map[string]interface{})
	dbg["reverseIPMap"] = make(map[string]interface{})
	dbg["historicalIPs"] = make(map[string]interface{})

	for addr, deplSet := range e.ipMap {
		dbg["ipMap"][addr.AsNetIP().String()] = deplSet.AsSlice()
	}
	for deplID, addrSet := range e.reverseIPMap {
		// addrSet.AsSlice() does not print well
		arr := make([]string, 0, addrSet.Cardinality())
		for _, addr := range addrSet.AsSlice() {
			arr = append(arr, addr.String())
		}
		dbg["reverseIPMap"][deplID] = arr
	}
	for addr, submap := range e.historicalIPs {
		for deplID, status := range submap {
			dbg["historicalIPs"][addr.AsNetIP().String()] = map[string]interface{}{
				"deplID":    deplID,
				"ticksLeft": status.ticksLeft,
			}
		}
	}
	return dbg
}

func (e *endpointsStore) debug() interface{} {
	dbg := make(map[string]map[string]map[string]interface{})
	dbg["endpointMap"] = make(map[string]map[string]interface{})
	dbg["reverseEndpointMap"] = make(map[string]map[string]interface{})
	dbg["historicalEndpoints"] = make(map[string]map[string]interface{})
	dbg["reverseHistoricalEndpoints"] = make(map[string]map[string]interface{})

	for ep, submap := range e.endpointMap {
		dbg["endpointMap"][ep.String()] = make(map[string]interface{})
		for deplID, targetInfoSet := range submap {
			dbg["endpointMap"][ep.String()][deplID] = targetInfoSet.AsSlice()
		}
	}
	dbg["reverseEndpointMap"]["deployments"] = make(map[string]interface{})
	for deplID, setOfEp := range e.reverseEndpointMap {
		// setOfEp.AsSlice() does not print well
		arr := make([]string, 0, setOfEp.Cardinality())
		for _, ep := range setOfEp.AsSlice() {
			arr = append(arr, ep.String())
		}
		// we need dummy entry "deployments" to fit into the dbg declaration
		dbg["reverseEndpointMap"]["deployments"][deplID] = arr
	}
	for ep, submap := range e.historicalEndpoints {
		dbg["historicalEndpoints"][ep.String()] = make(map[string]interface{})
		for deplID, targetInfoSetMap := range submap {
			for targetInfo, status := range targetInfoSetMap {
				dbg["historicalEndpoints"][ep.String()][deplID] = map[string]interface{}{
					"targetInfo": targetInfo,
					"ticksLeft":  status.ticksLeft,
				}
			}
		}
	}
	dbg["reverseHistoricalEndpoints"] = make(map[string]map[string]interface{})
	for deplID, submap := range e.reverseHistoricalEndpoints {
		dbg["reverseHistoricalEndpoints"][deplID] = make(map[string]interface{})
		for ep, status := range submap {
			dbg["reverseHistoricalEndpoints"][deplID] = map[string]interface{}{
				"endpoint":  ep.String(),
				"ticksLeft": status.ticksLeft,
			}
		}
	}
	return dbg
}
