package billingmetrics

import (
	"encoding/csv"
	"fmt"
	"io"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoconv"
)

func writeCSV(metrics []storage.BillingMetrics, wio io.Writer) error {
	w := csv.NewWriter(wio)
	record := []string{"UTC Timestamp", "Nodes", "Millicores"}
	if err := w.Write(record); err != nil {
		return err
	}
	for _, m := range metrics {
		record[0] = protoconv.ConvertTimestampToTimeOrNow(m.Ts).UTC().Format(time.RFC3339)
		record[1] = fmt.Sprint(m.Sr.GetNodes())
		record[2] = fmt.Sprint(m.Sr.GetMillicores())
		if err := w.Write(record); err != nil {
			return err
		}
	}
	w.Flush()
	return w.Error()
}
