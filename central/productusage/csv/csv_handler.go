package usagecsv

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	datastore "github.com/stackrox/rox/central/productusage/datastore/securedunits"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/csv"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/protoconv"
)

var (
	zeroTime  = time.Time{}
	csvHeader = csv.Row{"Timestamp", "Nodes", "CPU Units"}
)

func getSecuredUnitsConverter() csv.Converter[storage.SecuredUnits] {
	record := make(csv.Row, 3)
	return func(m *storage.SecuredUnits) csv.Row {
		record[0] = ""
		if m.GetTimestamp() != nil {
			if t, err := types.TimestampFromProto(m.GetTimestamp()); err == nil {
				record[0] = t.UTC().Format(time.RFC3339)
			}
		}
		record[1] = fmt.Sprint(m.GetNumNodes())
		record[2] = fmt.Sprint(m.GetNumCpuUnits())
		return record
	}
}

// CSVHandler returns an HTTP handler function that serves product usage data as CSV.
func CSVHandler(ds datastore.DataStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		from, to, err := parseRequest(r)
		if err != nil {
			err = errox.InvalidArgs.New("bad CSV product usage request").CausedBy(err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		var filename string
		{
			fromts, _ := types.TimestampFromProto(from)
			tots, _ := types.TimestampFromProto(to)
			filename = fmt.Sprintf("secured_units_usage-%s-%s.csv", fromts.Format(time.DateOnly), tots.Format(time.DateOnly))
		}
		csvWriter := csv.NewHTTPWriter(w, filename,
			getSecuredUnitsConverter(), csvHeader)

		if err := ds.Walk(r.Context(), from, to, csvWriter.Write); err != nil {
			_ = csvWriter.SetHTTPError(errors.WithMessage(err,
				"failed to retrieve secured units usage data"))
			return
		}
		csvWriter.Flush()
	}
}

func getTimeParameter(values url.Values, param string, defaultValue time.Time) (*types.Timestamp, error) {
	result := defaultValue
	if v := values.Get(param); v != "" {
		var err error
		if result, err = time.Parse(time.RFC3339Nano, v); err != nil {
			return nil, errors.Wrapf(err, "failed to parse %q parameter", param)
		}
	}
	return protoconv.ConvertTimeToTimestamp(result), nil
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
