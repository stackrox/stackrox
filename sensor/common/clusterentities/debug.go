package clusterentities

import (
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/concurrency"
)

// StartDebugServer starts HTTP server that allows to look inside the clusterentities store.
// This blocks and should be always started in a goroutine!
func (e *Store) StartDebugServer() {
	http.HandleFunc("/debug/clusterentities/state.json", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		n, err := fmt.Fprintf(w, "%s\n", e.Debug())
		log.Debugf("Serving debug http endpoint: n=%d, err=%v", n, err)
	})
	err := http.ListenAndServe(":8099", nil)
	if err != nil {
		log.Error(errors.Wrap(err, "unable to start cluster entities store debug server"))
	}
}

// Debug returns an object that represents the current state of the entire store
func (e *Store) Debug() []byte {
	m := make(map[string]interface{})
	m["endpoints"] = e.endpointsStore.debug()
	m["IPs"] = e.podIPsStore.debug()
	m["containerIDs"] = e.containerIDsStore.debug()
	// json pretty-printer will sort it for us.
	concurrency.WithLock(&e.traceMutex, func() {
		// We need to clone the trace map, otherwise json.Marshal might panic when
		// reading the map if track is called at the same time.
		m["events"] = maps.Clone(e.trace)
	})

	ret, err := json.Marshal(m)
	if err != nil {
		log.Errorf("Error marshalling store debug: %v", err)
	}
	return ret
}

func (e *Store) track(format string, vals ...interface{}) {
	if !e.debugMode {
		return
	}
	e.traceMutex.Lock()
	defer e.traceMutex.Unlock()
	e.trace[time.Now().Format(time.RFC3339Nano)] = fmt.Sprintf(format, vals...)
}

func (e *containerIDsStore) debug() interface{} {
	dbg := make(map[string]map[string]interface{})
	dbg["containerIDMap"] = make(map[string]interface{})
	dbg["historicalContainerIDs"] = make(map[string]interface{})
	dbg["reverseContainerIDMap"] = make(map[string]interface{})

	concurrency.WithRLock(&e.mutex, func() {
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
	})
	return dbg
}

func (e *podIPsStore) debug() interface{} {
	dbg := make(map[string]map[string]interface{})
	dbg["ipMap"] = make(map[string]interface{})
	dbg["reverseIPMap"] = make(map[string]interface{})
	dbg["historicalIPs"] = make(map[string]interface{})

	concurrency.WithRLock(&e.mutex, func() {
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
	})
	return dbg
}

func (e *endpointsStore) debug() interface{} {
	dbg := make(map[string]map[string]map[string]interface{})
	dbg["endpointMap"] = make(map[string]map[string]interface{})
	dbg["reverseEndpointMap"] = make(map[string]map[string]interface{})
	dbg["historicalEndpoints"] = make(map[string]map[string]interface{})
	dbg["reverseHistoricalEndpoints"] = make(map[string]map[string]interface{})

	concurrency.WithRLock(&e.mutex, func() {
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
	})
	return dbg
}
