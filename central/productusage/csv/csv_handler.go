package usagecsv

import (
	"net/http"
	"net/url"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	datastore "github.com/stackrox/rox/central/productusage/datastore/securedunits"
	"github.com/stackrox/rox/pkg/protoconv"
)

var utf8BOM = ([]byte)("\uFEFF") // to please Windows CSV editors.

func writeError(w http.ResponseWriter, code int, err error, description string) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(code)
	_, _ = w.Write([]byte(errors.Wrap(err, description).Error()))
}

// CSVHandler returns an HTTP handler function that serves usage data as CSV.
func CSVHandler(ds datastore.DataStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		from, to, err := parseRequest(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, err, "bad CSV usage metrics request")
			return
		}

		metrics, err := ds.Get(r.Context(), from, to)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err, "failed to call usage metrics store")
			return
		}

		w.Header().Set("Content-Type", `text/csv; charset="utf-8"`)
		w.Header().Set("Content-Disposition", `attachment; filename="secured_units_usage.csv"`)

		if n, err := w.Write(utf8BOM); err != nil || n != len(utf8BOM) {
			writeError(w, http.StatusInternalServerError, err, "failed to write BOM header")
			return
		}
		if err := writeCSV(metrics, w); err != nil {
			writeError(w, http.StatusInternalServerError, err, "failed to send CSV data")
			return
		}
	}
}

func getTimeParameter(r url.Values, param string, defaultValue time.Time) (*types.Timestamp, error) {
	if v := r.Get(param); v != "" {
		var err error
		if defaultValue, err = time.Parse(time.RFC3339Nano, v); err != nil {
			return nil, errors.Wrapf(err, "failed to parse %q parameter", param)
		}
	}
	return protoconv.ConvertTimeToTimestamp(defaultValue), nil
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
