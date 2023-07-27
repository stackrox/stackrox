package usagecsv

import (
	"net/http"
	"net/url"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/usage/datastore"
	"github.com/stackrox/rox/pkg/protoconv"
)

// CSVHandler returns an HTTP handler function that serves usage data as CSV.
func CSVHandler(ds datastore.DataStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		from, to, err := parseRequest(r)
		if err != nil {
			log.Error("Bad CSV usage metrics request: ", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		metrics, err := ds.Get(r.Context(), from, to)
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

func getTimeParameter(r url.Values, param string, def time.Time) (*types.Timestamp, error) {
	if v := r.Get(param); v != "" {
		var err error
		if def, err = time.Parse(time.RFC3339Nano, v); err != nil {
			return nil, errors.Wrapf(err, "failed to parse '%s' parameter", param)
		}
	}
	return protoconv.ConvertTimeToTimestamp(def), nil
}

func parseRequest(r *http.Request) (*types.Timestamp, *types.Timestamp, error) {
	if err := r.ParseForm(); err != nil {
		return nil, nil, errors.Wrap(err, "failed to parse request paremeters")
	}
	var err error
	var from, to *types.Timestamp
	if from, err = getTimeParameter(r.Form, "from", zeroTime); err != nil {
		return nil, nil, err
	}
	if to, err = getTimeParameter(r.Form, "to", time.Now()); err != nil {
		return nil, nil, err
	}
	return from, to, nil
}
