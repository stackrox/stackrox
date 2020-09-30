package renderer

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/zip"
	"helm.sh/helm/v3/pkg/chartutil"
)

func loadAndMergeValues(valuesFiles []*zip.File) (chartutil.Values, error) {
	var mergedValues chartutil.Values
	for _, file := range valuesFiles {
		values, err := chartutil.ReadValues(file.Content)
		if err != nil {
			return nil, errors.Wrapf(err, "reading helm values from %s", file.Name)
		}
		mergedValues = chartutil.CoalesceTables(mergedValues, values)
	}

	return mergedValues, nil
}
