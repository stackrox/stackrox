package prometheusutil

import (
	"fmt"
	"io"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/expfmt"
	"github.com/stackrox/stackrox/pkg/errorhelpers"
)

// ExportText prometheus metrics to io.Writer in text format
func ExportText(w io.Writer) error {
	g := prometheus.DefaultGatherer
	mfs, err := g.Gather()
	if err != nil {
		// Failed to gather metrics.  Write the error to the file and return.  If we fail to write the error to the
		// file return both errors.
		_, writeErr := fmt.Fprintf(w, "# ERROR: %s\n", err.Error())
		return errorhelpers.NewErrorListWithErrors("gathering prometheus metrics", []error{err, writeErr}).ToError()
	}
	for _, mf := range mfs {
		if _, err := expfmt.MetricFamilyToText(w, mf); err != nil {
			// Failed to write a metric family.  Write the error to the file and continue
			if _, writeErr := w.Write([]byte(fmt.Sprintf("# ERROR: %s\n", err.Error()))); writeErr != nil {
				// Failed to write the error to the file.  Return both errors.
				errList := errorhelpers.NewErrorListWithErrors(fmt.Sprintf("writing metric family %s", mf.GetName()), []error{err, writeErr})
				return errList.ToError()
			}

		}
	}
	return nil
}
