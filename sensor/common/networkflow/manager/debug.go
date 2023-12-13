package manager

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/stackrox/rox/pkg/timestamp"
)

func knownConnections2String(m map[connection]*connStatus) string {
	arr0 := make([]string, 0, len(m))
	for c, cs := range m {
		jsn, _ := cs.MarshalJSON()
		arr0 = append(arr0, fmt.Sprintf("%q: %s", c.String(), string(jsn)))
	}
	return strings.Join(arr0, ",")
}

func updatedConnections2String(m map[connection]timestamp.MicroTS) string {
	arr0 := make([]string, 0, len(m))
	for c, ts := range m {
		arr0 = append(arr0, fmt.Sprintf("%q: %s", c.String(), ts.GoTime().String()))
	}
	return strings.Join(arr0, ",")
}

func updatedEndpoints2String(m map[containerEndpoint]timestamp.MicroTS) string {
	arr0 := make([]string, 0, len(m))
	for ce, ts := range m {
		arr0 = append(arr0, fmt.Sprintf("%q: %s", ce.String(), ts.GoTime().String()))
	}
	return strings.Join(arr0, ",")
}

func (m *networkFlowManager) startDebugServer() *http.Server {
	handler := http.NewServeMux()
	handler.HandleFunc("/debug/connections", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		data, err := json.Marshal(m.connectionsByHost)
		if err != nil {
			log.Errorf("marshalling error: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, err = w.Write(data)
		if err != nil {
			log.Errorf("data writing error: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	})
	srv := &http.Server{Addr: "127.0.0.1:6067", Handler: handler}
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Warnf("Closing debugging server 6067: %v", err)
		}
	}()
	return srv
}

type dbgHostConnections struct {
	Hostname              string
	Connections           map[string]*connStatus
	Endpoints             map[string]*connStatus
	LastKnownTimestamp    timestamp.MicroTS
	ConnectionsSequenceID int64
	CurrentSequenceID     int64
}

func (h *hostConnections) MarshalJSON() ([]byte, error) {
	dbg := dbgHostConnections{
		Hostname:              h.hostname,
		Connections:           make(map[string]*connStatus),
		Endpoints:             make(map[string]*connStatus),
		LastKnownTimestamp:    h.lastKnownTimestamp,
		ConnectionsSequenceID: h.connectionsSequenceID,
		CurrentSequenceID:     h.connectionsSequenceID,
	}
	for c, status := range h.connections {
		dbg.Connections[c.String()] = status
	}
	for ce, status := range h.endpoints {
		dbg.Endpoints[ce.String()] = status
	}
	return json.Marshal(dbg)
}

func (cs *connStatus) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"used":        cs.used,
		"lastSeen":    cs.lastSeen,
		"rotten":      cs.rotten,
		"firstSeen":   cs.firstSeen,
		"usedProcess": cs.usedProcess,
	})
}
