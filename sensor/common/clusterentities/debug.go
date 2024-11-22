package clusterentities

import (
	"encoding/json"
)

// Debug returns an object that represents the current state of the entire store
func (e *Store) Debug() []byte {
	m := make(map[string]interface{})
	m["endpoints"] = e.endpointsStore.debug()
	m["IPs"] = e.ipsStore.debug()
	m["containerIDs"] = e.containerIDsStore.debug()
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

	for cID, metadata := range e.containerIDMap {
		dbg["containerIDMap"][cID] = metadata
	}
	for cID, submap := range e.historicalContainerIDs {
		for metadata, status := range submap {
			dbg["historicalContainerIDs"][cID] = map[string]interface{}{
				"metadata": metadata,
				"status":   status,
			}
		}
	}
	return dbg
}


func (e *ipsStore) debug() interface{} {
	dbg := make(map[string]map[string]interface{})
	dbg["ipMap"] = make(map[string]interface{})
	dbg["historicalIPs"] = make(map[string]interface{})

	for addr, deplSet := range e.ipMap {
			dbg["ipMap"][addr.AsNetIP().String()] = deplSet.AsSlice()
	}
	for addr, submap := range e.historicalIPs {
		for deplID, status := range submap {
			dbg["historicalIPs"][addr.AsNetIP().String()] = map[string]interface{}{
				"deplID": deplID,
				"status":   status,
			}
		}
	}
	return dbg
}

func (e *endpointsStore) debug() interface{} {
	dbg := make(map[string]map[string]map[string]interface{})
	dbg["endpointMap"] = make(map[string]map[string]interface{})
	dbg["historicalEndpoints"] = make(map[string]map[string]interface{})

	for ep, submap := range e.endpointMap {
		dbg["endpointMap"][ep.String()] = make(map[string]interface{})
		for deplID, targetInfoSet := range submap {
			dbg["endpointMap"][ep.String()][deplID] = targetInfoSet.AsSlice()
		}
	}
	for ep, submap := range e.historicalEndpoints {
		dbg["historicalEndpoints"][ep.String()] = make(map[string]interface{})
		for deplID, targetInfoSetMap := range submap {
			for targetInfo, status := range targetInfoSetMap {
				dbg["historicalEndpoints"][ep.String()][deplID] = map[string]interface{}{
					"targetInfo": targetInfo,
					"status":   status,
				}
			}
		}
	}
	return dbg
}
