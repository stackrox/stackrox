package backgroundworker

import (
	"encoding/json"
	"net/http"
)

// DebugHandler returns an http.Handler that serves the status of every
// registered worker as a JSON array.
func (r *Registry) DebugHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		statuses := r.All()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(statuses)
	})
}
