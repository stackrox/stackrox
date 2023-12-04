package clusterentities

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/stackrox/rox/pkg/net"
)

func (e *Store) startDebugServer() *http.Server {
	handler := http.NewServeMux()
	handler.HandleFunc("/debug/endpoints", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/json")
		_, err := w.Write([]byte(e.dbgPrintEndpoints(e.endpointMap)))
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	handler.HandleFunc("/debug/ips", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/json")
		_, err := w.Write([]byte(e.dbgPrintIPs(e.ipMap)))
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	handler.HandleFunc("/debug/past/endpoints", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/json")
		_, err := w.Write([]byte(e.dbgPrintHistoricalEp(e.historicalEndpoints)))
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	handler.HandleFunc("/debug/past/ips", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/json")
		_, err := w.Write([]byte(e.dbgPrintHistoricalIPs(e.historicalIPs)))
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	srv := &http.Server{Addr: "127.0.0.1:6066", Handler: handler}
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Warnf("Closing debugging server: %v", err)
		}
	}()
	return srv
}

func (e *Store) dbgPrintEndpoints(endpointMap map[net.NumericEndpoint]map[string]map[EndpointTargetInfo]struct{}) string {
	arr0 := make([]string, 0, len(endpointMap))
	for ep, m := range endpointMap {
		arr1 := make([]string, 0, len(m))
		for deplID, eti := range m {
			arr2 := make([]string, 0, len(eti))
			for info := range eti {
				arr2 = append(arr2, fmt.Sprintf("%q", info.PortName))
			}
			repr2 := fmt.Sprintf("{%q:[%s]}", deplID, strings.Join(arr2, ","))
			arr1 = append(arr1, repr2)
		}
		repr3 := fmt.Sprintf("{%q: [%s]}", ep.String(), strings.Join(arr1, ","))
		arr0 = append(arr0, repr3)
	}
	return fmt.Sprintf("{\"Endpoints\": [%s]}", strings.Join(arr0, ","))
}

func (e *Store) dbgPrintIPs(ipMap map[net.IPAddress]map[string]struct{}) string {
	arr0 := make([]string, 0, len(ipMap))
	for ip, m := range ipMap {
		arr1 := make([]string, 0, len(m))
		for deplID := range m {
			arr1 = append(arr1, fmt.Sprintf("%q", deplID))
		}
		arr0 = append(arr0, fmt.Sprintf("{%q: [%s]}", ip.String(), strings.Join(arr1, ",")))
	}
	return fmt.Sprintf("{\"IPs\": [%s]}", strings.Join(arr0, ","))
}

func (e *Store) dbgPrintHistoricalEp(historicalEndpoints map[string]map[net.NumericEndpoint]*entityStatus) string {
	arr0 := make([]string, 0, len(historicalEndpoints))
	for deplID, m := range historicalEndpoints {
		arr1 := make([]string, 0, len(historicalEndpoints))
		for ep, status := range m {
			arr1 = append(arr1, fmt.Sprintf("{\"ep\":%q, \"ticksLeft\": %d}", ep.String(), status.ticksLeft))
		}
		arr0 = append(arr0, fmt.Sprintf("{%q: [%s]}", deplID, strings.Join(arr1, ",")))
	}
	return fmt.Sprintf("{\"historicalEndpoints\": [%s]}", strings.Join(arr0, ","))
}

func (e *Store) dbgPrintHistoricalIPs(historicalIPs map[net.IPAddress]map[string]*entityStatus) string {
	arr0 := make([]string, 0, len(historicalIPs))
	for ip, m := range historicalIPs {
		arr1 := make([]string, 0, len(m))
		for deplID, status := range m {
			arr1 = append(arr1, fmt.Sprintf("{\"deplID\":%q, \"ticksLeft\": %d}", deplID, status.ticksLeft))
		}
		arr0 = append(arr0, fmt.Sprintf("{%q: [%s]}", ip.String(), strings.Join(arr1, ",")))
	}
	return fmt.Sprintf("{\"historicalIPs\": [%s]}", strings.Join(arr0, ","))
}
