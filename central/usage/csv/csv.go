package usagecsv

import (
	"encoding/csv"
	"fmt"
	"io"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protoconv"
)

var (
	zeroTime = time.Unix(0, 0).UTC()

	log = logging.LoggerForModule()
)

func writeCSV(metrics []*storage.Usage, wio io.Writer) error {
	w := csv.NewWriter(wio)
	record := []string{"Timestamp", "Nodes", "Cores"}
	if err := w.Write(record); err != nil {
		return errors.Wrap(err, "failed to write CSV header")
	}
	for _, m := range metrics {
		record[0] = protoconv.ConvertTimestampToTimeOrDefault(m.Timestamp, zeroTime).UTC().Format(time.RFC3339)
		record[1] = fmt.Sprint(m.GetNumNodes())
		record[2] = fmt.Sprint(m.GetNumCores())
		if err := w.Write(record); err != nil {
			return errors.Wrap(err, "failed to write CSV record")
		}
	}
	w.Flush()
	return errors.Wrap(w.Error(), "failed to flush CSV buffer")
}
