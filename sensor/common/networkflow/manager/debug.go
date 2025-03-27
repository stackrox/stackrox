package manager

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/pkg/errors"
)

// StartDebugServer starts HTTP server that allows to look inside the active connections and endpoints.
// This blocks and should be always started in a goroutine!
func (m *networkFlowManager) StartDebugServer(addr string) error {
	http.HandleFunc("/debug/netflow/state.json", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		n, err := fmt.Fprintf(w, "%s\n", m.Debug())
		log.Debugf("Serving debug http endpoint: n=%d, err=%v", n, err)
	})
	err := http.ListenAndServe(addr, nil)
	if err != nil {
		log.Error(errors.Wrap(err, "unable to start networkFlow manager debug server"))
	}
	return err
}

// Debug returns an object that represents the current state of the entire store
func (m *networkFlowManager) Debug() []byte {
	d := make(map[string]map[string]string)
	d["connections"] = make(map[string]string)
	for c, indicator := range m.activeConnections {
		d["connections"][c.String()] = indicator.String()
	}
	d["endpoints"] = make(map[string]string)
	for ep, indicator := range m.activeEndpoints {
		d["endpoints"][ep.String()] = indicator.String()
	}
	ret, err := json.Marshal(m)
	if err != nil {
		log.Errorf("Error marshalling networkFlowManager debug: %v", err)
	}
	return ret
}
