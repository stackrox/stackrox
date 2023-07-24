package billingmetrics

import (
	"net/http"
	"time"

	bmstore "github.com/stackrox/rox/central/billingmetrics/store"
	"github.com/stackrox/rox/pkg/protoconv"
)

// CSVHandler returns an HTTP handler function that serves billing data as CSV.
func CSVHandler() http.HandlerFunc {
	store := bmstore.Singleton()
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			w.WriteHeader(http.StatusBadRequest)
		}
		var err error
		from := time.Unix(0, 0).UTC()
		to := time.Now()
		if reqFrom := r.Form.Get("from"); reqFrom != "" {
			from, err = time.Parse(time.RFC3339, reqFrom)
		}
		if reqTo := r.Form.Get("to"); reqTo != "" {
			to, err = time.Parse(time.RFC3339, reqTo)
		}
		if err != nil {
			log.Error("Bad CSV billing metrics request: ", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		metrics, err := store.Get(r.Context(), protoconv.ConvertTimeToTimestamp(from), protoconv.ConvertTimeToTimestamp(to))
		if err != nil {
			log.Error("Failed to call billing metrics store: ", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Add("Content-Type", "text/csv")
		err = writeCSV(metrics, w)
		if err != nil {
			log.Error("Failed to send CSV data: ", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}
