package billingmetrics

import (
	"encoding/csv"
	"fmt"
	"io"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoconv"
)

var (
	zeroTime = time.Unix(0, 0).UTC()
)

func writeCSV(metrics []storage.BillingMetrics, wio io.Writer) error {
	w := csv.NewWriter(wio)
	record := []string{"UTC Timestamp", "Nodes", "Cores"}
	if err := w.Write(record); err != nil {
		return errors.Wrap(err, "failed to write CSV header")
	}
	for _, m := range metrics {
		record[0] = protoconv.ConvertTimestampToTimeOrDefault(m.Ts, zeroTime).UTC().Format(time.RFC3339)
		record[1] = fmt.Sprint(m.Sr.GetNodes())
		record[2] = fmt.Sprint(m.Sr.GetCores())
		if err := w.Write(record); err != nil {
			return errors.Wrap(err, "failed to write CSV record")
		}
	}
	w.Flush()
	return errors.Wrap(w.Error(), "failed to flush CSV buffer")
}
