package usagecsv

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	datastore "github.com/stackrox/rox/central/administration/usage/datastore/securedunits"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/csv"
	"github.com/stackrox/rox/pkg/errox"
	grpcErrors "github.com/stackrox/rox/pkg/grpc/errors"
)

var (
	zeroTime  = time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC)
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

// CSVHandler returns an HTTP handler function that serves administration usage data as CSV.
// The handler accepts two optional string parameters "from" and "to", which are
// expected to represent a timestamp in RFC3339 (or 'ISO') format.
// The parameters define a time range in which the data is searched. The result
// includes data not older than "from" and older than "to", i.e., [from, to).
// If "from" is ommited, an 0 timestamp is taken instead. If "to" is ommited,
// the current time is taken.
func CSVHandler(ds datastore.DataStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		from, to, err := parseRequest(r)
		if err != nil {
			err = errox.InvalidArgs.New("bad CSV administration usage request").CausedBy(err)
			http.Error(w, err.Error(), grpcErrors.ErrToHTTPStatus(err))
			return
		}

		csvWriter := csv.NewHTTPWriter(w,
			fmt.Sprintf("secured_units_usage-%s--%s.csv",
				from.Format(time.DateOnly),
				to.Format(time.DateOnly)),
			getSecuredUnitsConverter(), csvHeader)

		if err := ds.Walk(r.Context(), from, to, csvWriter.Write); err != nil {
			_ = csvWriter.SetHTTPError(errors.Wrap(err,
				"failed to retrieve secured units usage data"))
			return
		}
		csvWriter.Flush()
	}
}

func getTimeParameter(values url.Values, param string, defaultValue time.Time) (time.Time, error) {
	result := defaultValue
	if v := values.Get(param); v != "" {
		var err error
		if result, err = time.Parse(time.RFC3339Nano, v); err != nil {
			return zeroTime, errors.Wrapf(err, "failed to parse %q parameter", param)
		}
	}
	return result, nil
}

func parseRequest(r *http.Request) (time.Time, time.Time, error) {
	if err := r.ParseForm(); err != nil {
		return zeroTime, zeroTime, errors.Wrap(err, "failed to parse request paremeters")
	}
	var err error
	var from, to time.Time
	if from, err = getTimeParameter(r.Form, "from", zeroTime); err != nil {
		return zeroTime, zeroTime, err
	}
	if to, err = getTimeParameter(r.Form, "to", time.Now()); err != nil {
		return zeroTime, zeroTime, err
	}
	return from, to, nil
}
