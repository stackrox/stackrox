package usagecsv

import (
	"net/http"
	"time"

	"github.com/stackrox/rox/central/usage/datastore"
	"github.com/stackrox/rox/pkg/protoconv"
)

// CSVHandler returns an HTTP handler function that serves usage data as CSV.
func CSVHandler(ds datastore.DataStore) http.HandlerFunc {
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
			log.Error("Bad CSV usage metrics request: ", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		metrics, err := ds.Get(r.Context(), protoconv.ConvertTimeToTimestamp(from), protoconv.ConvertTimeToTimestamp(to))
		if err != nil {
			log.Error("Failed to call usage metrics store: ", err)
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
