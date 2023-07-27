package usagecsv

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	datastore "github.com/stackrox/rox/central/productusage/datastore/securedunits"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/csv"
	"github.com/stackrox/rox/pkg/protoconv"
)

var zeroTime = time.Time{}

func writeError(w http.ResponseWriter, code int, err error, description string) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(code)
	_, _ = w.Write([]byte(errors.Wrap(err, description).Error()))
}

func makeCSVWriter(w io.Writer) csv.StreamWriter[storage.SecuredUnits] {
	record := make([]string, 3)
	return csv.NewStreamWriter[storage.SecuredUnits](w,
		csv.WithBOM(),
		csv.WithCRLF(),
		csv.WithHeader("Timestamp", "Nodes", "CPU Units"),
		csv.WithConverter[storage.SecuredUnits](func(m *storage.SecuredUnits) ([]string, error) {
			record[0] = protoconv.ConvertTimestampToTimeOrDefault(m.GetTimestamp(), zeroTime).UTC().Format(time.RFC3339)
			record[1] = fmt.Sprint(m.GetNumNodes())
			record[2] = fmt.Sprint(m.GetNumCpuUnits())
			return record, nil
		}),
	)
}

// CSVHandler returns an HTTP handler function that serves usage data as CSV.
func CSVHandler(ds datastore.DataStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		from, to, err := parseRequest(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, err, "bad CSV usage metrics request")
			return
		}

		csvWriter := makeCSVWriter(w)

		if err := ds.Walk(r.Context(), from, to, csvWriter.AddRow); err != nil {
			writeError(w, http.StatusInternalServerError, err, "failed to process secured units usage data")
			return
		}

		w.Header().Set("Content-Type", `text/csv; charset="utf-8"`)
		w.Header().Set("Content-Disposition", `attachment; filename="secured_units_usage.csv"`)
		csvWriter.Flush()
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
