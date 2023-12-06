package manager

import (
	"encoding/json"
	"net/http"
)

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
		w.WriteHeader(http.StatusOK)
	})
	srv := &http.Server{Addr: "127.0.0.1:6067", Handler: handler}
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Warnf("Closing debugging server 6067: %v", err)
		}
	}()
	return srv
}
